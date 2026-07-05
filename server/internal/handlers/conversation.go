package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type ConversationHandler struct {
	DB *sql.DB
}

type createConvRequest struct {
	ParticipantID string `json:"participant_id"`
}

type conversationResponse struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	CreatedAt     string            `json:"created_at"`
	LastMessageAt string            `json:"last_message_at"`
	Participants  []profileResponse `json:"participants"`
	LastMessage   *string           `json:"last_message,omitempty"`
}

func (h *ConversationHandler) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 0 {
		page = 0
	}
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := page * pageSize

	rows, err := h.DB.Query(`
		SELECT c.id, c.type, c.created_at, c.last_message_at
		FROM conversations c
		JOIN participants p ON c.id = p.conversation_id
		WHERE p.user_id = ?
		ORDER BY c.last_message_at DESC
		LIMIT ? OFFSET ?
	`, userID, pageSize, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch conversations"},
		})
	}
	defer rows.Close()

	var convs []conversationResponse
	for rows.Next() {
		var conv conversationResponse
		var createdAt, lastMsgAt time.Time
		if err := rows.Scan(&conv.ID, &conv.Type, &createdAt, &lastMsgAt); err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: ErrorBody{Code: "INTERNAL", Message: "Failed to scan conversation"},
			})
		}
		conv.CreatedAt = createdAt.Format("2006-01-02T15:04:05Z")
		conv.LastMessageAt = lastMsgAt.Format("2006-01-02T15:04:05Z")
		convs = append(convs, conv)
	}

	if convs == nil {
		convs = []conversationResponse{}
	}

	return c.JSON(http.StatusOK, convs)
}

func (h *ConversationHandler) Create(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	var req createConvRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.ParticipantID == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "participant_id is required"},
		})
	}

	// Check if direct conversation already exists between these two users
	var existingID string
	err := h.DB.QueryRow(`
		SELECT c.id FROM conversations c
		WHERE c.type = 'direct'
		AND EXISTS (SELECT 1 FROM participants WHERE conversation_id = c.id AND user_id = ?)
		AND EXISTS (SELECT 1 FROM participants WHERE conversation_id = c.id AND user_id = ?)
	`, userID, req.ParticipantID).Scan(&existingID)

	if err == nil {
		// Return existing conversation
		return c.JSON(http.StatusOK, map[string]string{"id": existingID})
	}

	// Create new conversation
	id := newUUID()
	now := time.Now()

	_, err = h.DB.Exec(
		"INSERT INTO conversations (id, type, created_at, last_message_at) VALUES (?, 'direct', ?, ?)",
		id, now, now,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to create conversation"},
		})
	}

	// Add participants
	_, err = h.DB.Exec(
		"INSERT INTO participants (conversation_id, user_id) VALUES (?, ?), (?, ?)",
		id, userID, id, req.ParticipantID,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to add participants"},
		})
	}

	return c.JSON(http.StatusCreated, conversationResponse{
		ID:            id,
		Type:          "direct",
		CreatedAt:     now.Format("2006-01-02T15:04:05Z"),
		LastMessageAt: now.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ConversationHandler) Get(c echo.Context) error {
	id := c.Param("id")
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	// Verify user is participant
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM participants WHERE conversation_id = ? AND user_id = ?", id, userID).Scan(&count)
	if count == 0 {
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorBody{Code: "FORBIDDEN", Message: "You are not a participant"},
		})
	}

	var cResp conversationResponse
	var createdAt, lastMsgAt time.Time
	err := h.DB.QueryRow(
		"SELECT id, type, created_at, last_message_at FROM conversations WHERE id = ?", id,
	).Scan(&cResp.ID, &cResp.Type, &createdAt, &lastMsgAt)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorBody{Code: "NOT_FOUND", Message: "Conversation not found"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to fetch conversation"},
		})
	}

	cResp.CreatedAt = createdAt.Format("2006-01-02T15:04:05Z")
	cResp.LastMessageAt = lastMsgAt.Format("2006-01-02T15:04:05Z")

	// Get participants
	pRows, err := h.DB.Query(`
		SELECT p.id, p.username, p.email, p.avatar_url, p.status, p.last_seen
		FROM profiles p
		JOIN participants pt ON p.id = pt.user_id
		WHERE pt.conversation_id = ?
	`, id)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pr profileResponse
			var lastSeen sql.NullTime
			if pRows.Scan(&pr.ID, &pr.Username, &pr.Email, &pr.AvatarURL, &pr.Status, &lastSeen) == nil {
				if lastSeen.Valid {
					pr.LastSeen = lastSeen.Time.Format("2006-01-02T15:04:05Z")
				}
				cResp.Participants = append(cResp.Participants, pr)
			}
		}
	}

	return c.JSON(http.StatusOK, cResp)
}
