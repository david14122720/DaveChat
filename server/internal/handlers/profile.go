package handlers

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"DaveChat/internal/config"
	"github.com/labstack/echo/v4"
)

type ProfileHandler struct {
	DB     *sql.DB
	Config *config.Config
}

var mimeExtMap = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

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

func (h *ProfileHandler) AvatarUpload(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Ensure upload directory exists — idempotent
	if err := os.MkdirAll(h.Config.UploadDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "UPLOAD_FAILED", Message: "Failed to create upload directory"},
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

	// Validate MIME type via magic bytes (first 512 bytes)
	buf := data
	if len(buf) > 512 {
		buf = buf[:512]
	}
	mimeType := http.DetectContentType(buf)
	ext, ok := mimeExtMap[mimeType]
	if !ok {
		return c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_MIME", Message: "Only JPEG, PNG, WebP accepted"},
		})
	}

	// Generate UUID-based filename
	filename := newUUID() + ext
	newPath := filepath.Join(h.Config.UploadDir, filename)

	// Write file to disk
	if err := os.WriteFile(newPath, data, 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "UPLOAD_FAILED", Message: "Failed to save file"},
		})
	}

	// Fetch current avatar_url for old file cleanup
	var oldAvatarURL *string
	if err := h.DB.QueryRow("SELECT avatar_url FROM profiles WHERE id = ?", userID).Scan(&oldAvatarURL); err != nil {
		slog.Warn("avatar upload: failed to query old avatar_url", "user_id", userID, "error", err)
	}

	// Update database
	avatarURL := "/uploads/" + filename
	_, err = h.DB.Exec("UPDATE profiles SET avatar_url = ? WHERE id = ?", avatarURL, userID)
	if err != nil {
		// Orphan cleanup: remove the file we just wrote
		os.Remove(newPath)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to update profile"},
		})
	}

	// Delete old file if it exists and is a local upload
	if oldAvatarURL != nil && *oldAvatarURL != "" && strings.HasPrefix(*oldAvatarURL, "/uploads/") {
		oldPath := filepath.Join(h.Config.UploadDir, strings.TrimPrefix(*oldAvatarURL, "/uploads/"))
		os.Remove(oldPath)
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

	// Fetch current avatar_url
	var avatarURL *string
	err := h.DB.QueryRow("SELECT avatar_url FROM profiles WHERE id = ?", userID).Scan(&avatarURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profile"},
		})
	}

	// Delete file from disk if it is a local upload
	if avatarURL != nil && *avatarURL != "" && strings.HasPrefix(*avatarURL, "/uploads/") {
		diskPath := filepath.Join(h.Config.UploadDir, strings.TrimPrefix(*avatarURL, "/uploads/"))
		os.Remove(diskPath) // best-effort — file may already be gone
	}

	// Set avatar_url to NULL in database
	_, err = h.DB.Exec("UPDATE profiles SET avatar_url = NULL WHERE id = ?", userID)
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
