package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"DaveChat/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB     *sql.DB
	Config *config.Config
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "username, email, and password are required"},
		})
	}

	if len(req.Password) < 6 {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Password must be at least 6 characters"},
		})
	}

	// Check duplicate email
	var exists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE email = ?)", req.Email).Scan(&exists)
	if exists {
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorBody{Code: "DUPLICATE_EMAIL", Message: "Email already registered"},
		})
	}

	// Check duplicate username
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE username = ?)", req.Username).Scan(&exists)
	if exists {
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorBody{Code: "DUPLICATE_USERNAME", Message: "Username already taken"},
		})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to hash password"},
		})
	}

	// Generate UUID v4
	id := newUUID()

	// Insert profile
	_, err = h.DB.Exec(
		"INSERT INTO profiles (id, username, email, password_hash, status) VALUES (?, ?, ?, ?, 'online')",
		id, req.Username, req.Email, string(hash),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to create user"},
		})
	}

	// Generate JWT
	token, err := generateJWT(id, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	return c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User: userResponse{
			ID:       id,
			Username: req.Username,
			Email:    req.Email,
		},
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "email and password are required"},
		})
	}

	var user struct {
		ID           string
		Username     string
		Email        string
		PasswordHash string
		AvatarURL    *string
	}

	err := h.DB.QueryRow(
		"SELECT id, username, email, password_hash, avatar_url FROM profiles WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Database error"},
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"},
		})
	}

	// Update status to online
	h.DB.Exec("UPDATE profiles SET status = 'online' WHERE id = ?", user.ID)

	token, err := generateJWT(user.ID, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User: userResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			AvatarURL: user.AvatarURL,
		},
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "UNAUTHORIZED", Message: "Missing user ID in context"},
		})
	}

	var user userResponse
	var avatar *string
	err := h.DB.QueryRow(
		"SELECT id, username, email, avatar_url FROM profiles WHERE id = ?", userID,
	).Scan(&user.ID, &user.Username, &user.Email, &avatar)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorBody{Code: "NOT_FOUND", Message: "User not found"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Database error"},
		})
	}

	user.AvatarURL = avatar
	return c.JSON(http.StatusOK, user)
}

// --- Helpers ---

func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate UUID: " + err.Error())
	}
	// Set UUID version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits (RFC 4122)
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}

func formatUUID(b []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "refresh_token is required"},
		})
	}

	// Hash the provided token
	tokenHash := sha256Hex(req.RefreshToken)

	// Look up in DB
	var storedID, userID, expiresAt string
	err := h.DB.QueryRow(
		"SELECT id, user_id, expires_at FROM refresh_tokens WHERE token_hash = ?", tokenHash,
	).Scan(&storedID, &userID, &expiresAt)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_TOKEN", Message: "Invalid refresh token"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Database error"},
		})
	}

	// Check expiry
	expTime, _ := time.Parse("2006-01-02 15:04:05", expiresAt)
	if time.Now().After(expTime) {
		// Delete expired token
		h.DB.Exec("DELETE FROM refresh_tokens WHERE id = ?", storedID)
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "TOKEN_EXPIRED", Message: "Refresh token expired, please login again"},
		})
	}

	// Delete the used refresh token (rotation)
	h.DB.Exec("DELETE FROM refresh_tokens WHERE id = ?", storedID)

	// Fetch user
	var user userResponse
	var avatar *string
	err = h.DB.QueryRow(
		"SELECT id, username, email, avatar_url FROM profiles WHERE id = ?", userID,
	).Scan(&user.ID, &user.Username, &user.Email, &avatar)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorBody{Code: "NOT_FOUND", Message: "User not found"},
		})
	}
	user.AvatarURL = avatar

	// Generate new tokens
	accessToken, err := generateJWT(userID, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	newRefreshToken := newUUID()
	newHash := sha256Hex(newRefreshToken)
	h.DB.Exec(
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		newUUID(), userID, newHash, time.Now().Add(30*24*time.Hour),
	)

	return c.JSON(http.StatusOK, refreshResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	})
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func generateJWT(userID, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
