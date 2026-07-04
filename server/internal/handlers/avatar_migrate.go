package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"image"
	"image/jpeg"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// MigrateLegacyAvatars reads all profiles with /uploads/ avatar_urls,
// loads the image file from disk, resizes/compresses, and saves the
// base64 data URI to the database. Once migrated, the filesystem-based
// static middleware can be removed.
func MigrateLegacyAvatars(db *sql.DB, uploadDir string) {
	rows, err := db.Query("SELECT id, avatar_url FROM profiles WHERE avatar_url LIKE '/uploads/%'")
	if err != nil {
		slog.Error("avatar migration: query failed", "error", err)
		return
	}
	defer rows.Close()

	var migrated int
	for rows.Next() {
		var userID, avatarPath string
		if err := rows.Scan(&userID, &avatarPath); err != nil {
			slog.Error("avatar migration: scan failed", "error", err)
			continue
		}

		diskPath := filepath.Join(uploadDir, strings.TrimPrefix(avatarPath, "/uploads/"))
		dataURI, err := processAvatarFile(diskPath)
		if err != nil {
			slog.Warn("avatar migration: skipping file", "path", diskPath, "error", err)
			continue
		}

		if _, err := db.Exec("UPDATE profiles SET avatar_url = ? WHERE id = ?", dataURI, userID); err != nil {
			slog.Error("avatar migration: update failed", "user_id", userID, "error", err)
			continue
		}
		migrated++
		slog.Info("avatar migration: migrated", "user_id", userID)
	}

	if err := rows.Err(); err != nil {
		slog.Error("avatar migration: rows iteration error", "error", err)
	}

	slog.Info("avatar migration: complete", "migrated", migrated)
}

// processAvatarFile reads an image file from disk, resizes to max 200px,
// encodes as JPEG q80, converts to a base64 data URI, and returns it.
func processAvatarFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Resize if larger than maxAvatarDim
	if w > maxAvatarDim || h > maxAvatarDim {
		ratio := math.Min(float64(maxAvatarDim)/float64(w), float64(maxAvatarDim)/float64(h))
		newW := int(math.Round(float64(w) * ratio))
		newH := int(math.Round(float64(h) * ratio))
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		src = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 80}); err != nil {
		return "", err
	}

	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
