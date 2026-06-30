package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type CallLogHandler struct {
	DB *sql.DB
}

type createCallLogRequest struct {
	CalleeID string `json:"callee_id"`
	Type     string `json:"type"`     // "audio" or "video"
	Status   string `json:"status"`   // "missed", "completed", "rejected", "cancelled", "busy"
	Duration int    `json:"duration_secs"`
}

type callLogResponse struct {
	ID           string  `json:"id"`
	CallerID     string  `json:"caller_id"`
	CalleeID     string  `json:"callee_id"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	StartedAt    string  `json:"started_at"`
	EndedAt      *string `json:"ended_at,omitempty"`
	DurationSecs int     `json:"duration_secs"`
}

func (h *CallLogHandler) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	rows, err := h.DB.Query(`
		SELECT id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs
		FROM call_logs
		WHERE caller_id = ? OR callee_id = ?
		ORDER BY started_at DESC
		LIMIT 50
	`, userID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch call logs"},
		})
	}
	defer rows.Close()

	var logs []callLogResponse
	for rows.Next() {
		var l callLogResponse
		var startedAt time.Time
		var endedAt sql.NullTime
		if err := rows.Scan(&l.ID, &l.CallerID, &l.CalleeID, &l.Type, &l.Status, &startedAt, &endedAt, &l.DurationSecs); err != nil {
			continue
		}
		l.StartedAt = startedAt.Format("2006-01-02T15:04:05Z")
		if endedAt.Valid {
			s := endedAt.Time.Format("2006-01-02T15:04:05Z")
			l.EndedAt = &s
		}
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []callLogResponse{}
	}

	return c.JSON(http.StatusOK, logs)
}

func (h *CallLogHandler) Create(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	var req createCallLogRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.CalleeID == "" || req.Type == "" || req.Status == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "callee_id, type, and status are required"},
		})
	}

	id := newUUID()
	now := time.Now()

	_, err := h.DB.Exec(`
		INSERT INTO call_logs (id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, userID, req.CalleeID, req.Type, req.Status, now, now, req.Duration)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to create call log"},
		})
	}

	return c.JSON(http.StatusCreated, callLogResponse{
		ID:           id,
		CallerID:     userID,
		CalleeID:     req.CalleeID,
		Type:         req.Type,
		Status:       req.Status,
		StartedAt:    now.Format("2006-01-02T15:04:05Z"),
		DurationSecs: req.Duration,
	})
}
