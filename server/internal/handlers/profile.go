package handlers

import (
	"database/sql"
	"net/http"
	"time"
	"github.com/labstack/echo/v4"
)

type ProfileHandler struct {
	DB *sql.DB
}

func (h *ProfileHandler) markStalePresenceOffline() {
	h.DB.Exec(`UPDATE profiles SET status = 'offline' WHERE status = 'online' AND last_seen < NOW(3) - INTERVAL 90 SECOND`)
}

type profileResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Status    string  `json:"status"`
	LastSeen  string  `json:"last_seen"`
}

func (h *ProfileHandler) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"}})
	}
	h.markStalePresenceOffline()
	rows, err := h.DB.Query(`SELECT id, username, email, avatar_url, CASE WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 'online' ELSE 'offline' END AS status, last_seen FROM profiles WHERE id != ? ORDER BY CASE WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 0 ELSE 1 END, username ASC`, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profiles"}})
	}
	defer rows.Close()
	var profiles []profileResponse
	for rows.Next() {
		var p profileResponse
		var lastSeen sql.NullTime
		if err := rows.Scan(&p.ID, &p.Username, &p.Email, &p.AvatarURL, &p.Status, &lastSeen); err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorBody{Code: "INTERNAL", Message: "Failed to scan profile"}})
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
	err := h.DB.QueryRow(`SELECT id, username, email, avatar_url, CASE WHEN status = 'online' AND last_seen >= NOW(3) - INTERVAL 90 SECOND THEN 'online' ELSE 'offline' END AS status, last_seen FROM profiles WHERE id = ?`, id).Scan(&p.ID, &p.Username, &p.Email, &p.AvatarURL, &p.Status, &lastSeen)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: ErrorBody{Code: "NOT_FOUND", Message: "Profile not found"}})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch profile"}})
	}
	if lastSeen.Valid {
		p.LastSeen = lastSeen.Time.UTC().Format(time.RFC3339)
	}
	return c.JSON(http.StatusOK, p)
}
