package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"DaveChat/internal/config"
	"github.com/labstack/echo/v4"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ProfileHandler struct {
	DB     *sql.DB
	Config *config.Config
}

// maxAvatarDim is the maximum width or height for avatar images.
const maxAvatarDim = 200

func (h *ProfileHandler) markStalePresenceOffline() {
	h.DB.Exec(`
		UPDATE profiles
		SET status = 'offline'
		WHERE status = 'online'
		  AND last_seen < NOW(3) - INTERVAL 90 SECOND
	`)
}

type profileResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
	Status    string  `json:"status"`
	LastSeen  string  `json:"last_seen"`
}

func (h *ProfileHandler) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	h.markStalePresenceOffline()

	rows, err := h.DB.Query(
		`SELECT id, username, email, avatar_url,
			CASE
				WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 'online'
				ELSE 'offline'
			END AS status,
			last_seen
		 FROM profiles
		 WHERE id != ?
		 ORDER BY
			CASE WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 0 ELSE 1 END,
			username ASC`,
		userID,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profiles"},
		})
	}
	defer rows.Close()

	var profiles []profileResponse
	for rows.Next() {
		var p profileResponse
		var lastSeen sql.NullTime
		if err := rows.Scan(&p.ID, &p.Username, &p.Email, &p.AvatarURL, &p.Status, &lastSeen); err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: ErrorBody{Code: "INTERNAL", Message: "Failed to scan profile"},
			})
		}
		if lastSeen.Valid {
			p.LastSeen = lastSeen.Time.UTC().Format(time.RFC3339)
		}
		profiles = append(profiles, p)
	}

	if profiles == nil {
		profiles = []profileResponse{}
	}

	return c.JSON(http.StatusOK, profiles)
}

func (h *ProfileHandler) Get(c echo.Context) error {
	id := c.Param("id")

	var p profileResponse
	var lastSeen sql.NullTime
	h.markStalePresenceOffline()

	err := h.DB.QueryRow(
		`SELECT id, username, email, avatar_url,
			CASE
				WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 'online'
				ELSE 'offline'
			END AS status,
			last_seen
		 FROM profiles WHERE id = ?`, id,
	).Scan(&p.ID, &p.Username, &p.Email, &p.AvatarURL, &p.Status, &lastSeen)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorBody{Code: "NOT_FOUND", Message: "Profile not found"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profile"},
		})
	}

	if lastSeen.Valid {
		p.LastSeen = lastSeen.Time.UTC().Format(time.RFC3339)
	}

	return c.JSON(http.StatusOK, p)
}

func (h *ProfileHandler) getProfileResponse(userID string) (*profileResponse, error) {
	var p profileResponse
	var lastSeen sql.NullTime
	h.markStalePresenceOffline()
	err := h.DB.QueryRow(
		`SELECT id, username, email, avatar_url,
			CASE
				WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 'online'
				ELSE 'offline'
			END AS status,
			last_seen
		 FROM profiles WHERE id = ?`, userID,
	).Scan(&p.ID, &p.Username, &p.Email, &p.AvatarURL, &p.Status, &lastSeen)
	if err != nil {
		return nil, err
	}
	if lastSeen.Valid {
		p.LastSeen = lastSeen.Time.UTC().Format(time.RFC3339)
	}
	return &p, nil
}

// decodeImage decodes an image from raw bytes, supporting JPEG, PNG, and WebP.
func decodeImage(data []byte) (image.Image, error) {
	// Try JPEG first (most common for avatars)
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err == nil {
		return img, nil
	}
	// Try PNG
	img, err = png.Decode(bytes.NewReader(data))
	if err == nil {
		return img, nil
	}
	// Try WebP (registered via blank import)
	img, _, err = image.Decode(bytes.NewReader(data))
	if err == nil {
		return img, nil
	}
	return nil, errors.New("unsupported image format: only JPEG, PNG, and WebP accepted")
}

// resizeImage scales img to fit within maxDim x maxDim while maintaining aspect ratio.
func resizeImage(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxDim && h <= maxDim {
		return img // no resize needed
	}

	ratio := math.Min(float64(maxDim)/float64(w), float64(maxDim)/float64(h))
	newW := int(float64(w) * ratio)
	newH := int(float64(h) * ratio)

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func (h *ProfileHandler) AvatarUpload(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Limit request body to MaxFileSize + 512 bytes margin
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, h.Config.MaxFileSize+512)

	// Parse multipart form
	file, _, err := c.Request().FormFile("avatar")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error: ErrorBody{Code: "FILE_TOO_LARGE", Message: "File exceeds maximum size"},
			})
		}
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "MISSING_FILE", Message: "No avatar file provided"},
		})
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "READ_ERROR", Message: "Failed to read file"},
		})
	}

	// Validate MIME type via magic bytes
	if len(data) > 512 {
		buf := data[:512]
		mimeType := http.DetectContentType(buf)
		if !strings.HasPrefix(mimeType, "image/") {
			return c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{
				Error: ErrorBody{Code: "INVALID_MIME", Message: "Only JPEG, PNG, WebP accepted"},
			})
		}
	}

	// Decode, resize, and re-encode as JPEG quality 80
	img, err := decodeImage(data)
	if err != nil {
		return c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_IMAGE", Message: err.Error()},
		})
	}

	resized := resizeImage(img, maxAvatarDim)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 80}); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "ENCODE_FAILED", Message: "Failed to process image"},
		})
	}

	// Base64 encode
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := "data:image/jpeg;base64," + base64Data

	// Check size of final data URI (warn if approaching TEXT limit ~65KB)
	if len(dataURI) > 60000 {
		slog.Warn("avatar data URI is large", "user_id", userID, "bytes", len(dataURI))
	}

	// Save directly to DB
	_, err = h.DB.Exec("UPDATE profiles SET avatar_url = ? WHERE id = ?", dataURI, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to update profile"},
		})
	}

	// Return full profile
	p, err := h.getProfileResponse(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profile"},
		})
	}
	return c.JSON(http.StatusOK, p)
}

func (h *ProfileHandler) AvatarDelete(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Set avatar_url to NULL in database (no files to delete — all in DB now)
	_, err := h.DB.Exec("UPDATE profiles SET avatar_url = NULL WHERE id = ?", userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to update profile"},
		})
	}

	// Return full profile with null avatar_url
	p, err := h.getProfileResponse(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profile"},
		})
	}
	return c.JSON(http.StatusOK, p)
}
