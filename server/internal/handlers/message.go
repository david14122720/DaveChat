package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type MessageHandler struct {
	DB *sql.DB
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type messageResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	CreatedAt      string `json:"created_at"`
}

func (h *MessageHandler) List(c echo.Context) error {
	convID := c.Param("id")
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Verify user is participant
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM participants WHERE conversation_id = ? AND user_id = ?", convID, userID).Scan(&count)
	if count == 0 {
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorBody{Code: "FORBIDDEN", Message: "You are not a participant"},
		})
	}

	cursor := c.QueryParam("cursor")
	limit := 50

	var rows *sql.Rows
	var err error

	if cursor != "" {
		rows, err = h.DB.Query(`
            SELECT id, conversation_id, sender_id, content, type, created_at
            FROM messages
            WHERE conversation_id = ? AND created_at < ?
            ORDER BY created_at DESC
            LIMIT ?
        `, convID, cursor, limit)
	} else {
		rows, err = h.DB.Query(`
            SELECT id, conversation_id, sender_id, content, type, created_at
            FROM messages
            WHERE conversation_id = ?
            ORDER BY created_at DESC
            LIMIT ?
        `, convID, limit)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch messages"},
		})
	}
	defer rows.Close()

	var messages []messageResponse
	for rows.Next() {
		var m messageResponse
		var createdAt time.Time
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &createdAt); err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: ErrorBody{Code: "INTERNAL", Message: "Failed to scan message"},
			})
		}
		m.CreatedAt = createdAt.Format("2006-01-02T15:04:05Z")
		messages = append(messages, m)
	}

	if messages == nil {
		messages = []messageResponse{}
	}

	// Get oldest message created_at as next cursor
	var nextCursor string
	if len(messages) > 0 {
		nextCursor = messages[len(messages)-1].CreatedAt
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"messages":    messages,
		"next_cursor": nextCursor,
	})
}

func (h *MessageHandler) Search(c echo.Context) error {
	convID := c.Param("id")
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "query parameter q is required"},
		})
	}

	// Verify user is participant
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM participants WHERE conversation_id = ? AND user_id = ?", convID, userID).Scan(&count)
	if count == 0 {
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorBody{Code: "FORBIDDEN", Message: "You are not a participant"},
		})
	}

	rows, err := h.DB.Query(`
		SELECT id, conversation_id, sender_id, content, type, created_at
		FROM messages
		WHERE conversation_id = ?
		  AND MATCH(content) AGAINST(? IN BOOLEAN MODE)
		ORDER BY created_at DESC
		LIMIT 50
	`, convID, query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Search failed"},
		})
	}
	defer rows.Close()

	var messages []messageResponse
	for rows.Next() {
		var m messageResponse
		var createdAt time.Time
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Type, &createdAt); err != nil {
			continue
		}
		m.CreatedAt = createdAt.Format("2006-01-02T15:04:05Z")
		messages = append(messages, m)
	}

	if messages == nil {
		messages = []messageResponse{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"messages": messages,
		"query":    query,
	})
}

func (h *MessageHandler) MarkRead(c echo.Context) error {
	convID := c.Param("id")
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Verify user is participant
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM participants WHERE conversation_id = ? AND user_id = ?", convID, userID).Scan(&count)
	if count == 0 {
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorBody{Code: "FORBIDDEN", Message: "You are not a participant"},
		})
	}

	// Mark all unread messages in this conversation as read
	_, err := h.DB.Exec(`
		INSERT IGNORE INTO read_receipts (message_id, user_id, read_at)
		SELECT m.id, ?, NOW()
		FROM messages m
		WHERE m.conversation_id = ?
		  AND m.sender_id != ?
		  AND NOT EXISTS (
			  SELECT 1 FROM read_receipts rr
			  WHERE rr.message_id = m.id AND rr.user_id = ?
		  )
	`, userID, convID, userID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to mark messages as read"},
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessageHandler) Create(c echo.Context) error {
	convID := c.Param("id")
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Verify user is participant
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM participants WHERE conversation_id = ? AND user_id = ?", convID, userID).Scan(&count)
	if count == 0 {
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorBody{Code: "FORBIDDEN", Message: "You are not a participant"},
		})
	}

	var req sendMessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.Content == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "content is required"},
		})
	}

	id := newUUID()
	now := time.Now()

	_, err := h.DB.Exec(
		"INSERT INTO messages (id, conversation_id, sender_id, content, type, created_at) VALUES (?, ?, ?, ?, 'text', ?)",
		id, convID, userID, req.Content, now,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to create message"},
		})
	}

	// Update conversation's last_message_at
	h.DB.Exec("UPDATE conversations SET last_message_at = ? WHERE id = ?", now, convID)

	return c.JSON(http.StatusCreated, messageResponse{
		ID:             id,
		ConversationID: convID,
		SenderID:       userID,
		Content:        req.Content,
		Type:           "text",
		CreatedAt:      now.Format("2006-01-02T15:04:05Z"),
	})
}
