package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
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
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

type refreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// Me returns the authenticated user's profile
func (h *AuthHandler) Me(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var profile struct {
		ID       string
		Username string
		Email    string
		Avatar   string
		Status   string
		LastSeen time.Time
	}
	err := h.DB.QueryRow(
		"SELECT id, username, email, COALESCE(avatar_url,''), status, last_seen FROM profiles WHERE id = ?",
		userID,
	).Scan(&profile.ID, &profile.Username, &profile.Email, &profile.Avatar, &profile.Status, &profile.LastSeen)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorBody{Code: "NOT_FOUND", Message: "User not found"},
		})
	}

	return c.JSON(http.StatusOK, userResponse{
		ID:       profile.ID,
		Username: profile.Username,
		Email:    profile.Email,
		Avatar:   profile.Avatar,
	})
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "MISSING_FIELDS", Message: "Username, email, and password are required"},
		})
	}

	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "WEAK_PASSWORD", Message: "Password must be at least 6 characters"},
		})
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_EMAIL", Message: "Invalid email format"},
		})
	}

	if len(req.Username) < 3 || len(req.Username) > 50 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_USERNAME", Message: "Username must be between 3 and 50 characters"},
		})
	}

	var exists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE email = ?)", req.Email).Scan(&exists)
	if exists {
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorBody{Code: "DUPLICATE_EMAIL", Message: "Email already registered"},
		})
	}

	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE username = ?)", req.Username).Scan(&exists)
	if exists {
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorBody{Code: "DUPLICATE_USERNAME", Message: "Username already taken"},
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to hash password"},
		})
	}

	id := newUUID()

	_, err = h.DB.Exec(
		"INSERT INTO profiles (id, username, email, password_hash, status, last_seen) VALUES (?, ?, ?, ?, 'offline', NOW(3))",
		id, req.Username, req.Email, string(hash),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to create user"},
		})
	}

	token, err := generateJWT(id, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	h.DB.Exec("UPDATE profiles SET last_seen = NOW(3) WHERE id = ?", id)

	return c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User: userResponse{
			ID:       id,
			Username: req.Username,
			Email:    req.Email,
		},
	})
}

// Login handles user login
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "MISSING_FIELDS", Message: "Email and password are required"},
		})
	}

	var profile struct {
		ID           string
		Username     string
		Email        string
		PasswordHash string
	}

	err := h.DB.QueryRow(
		"SELECT id, username, email, password_hash FROM profiles WHERE email = ?",
		req.Email,
	).Scan(&profile.ID, &profile.Username, &profile.Email, &profile.PasswordHash)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Error al buscar usuario"},
		})
	}

	err = bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(req.Password))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"},
		})
	}

	token, err := generateJWT(profile.ID, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	h.DB.Exec("UPDATE profiles SET status = 'online', last_seen = NOW(3) WHERE id = ?", profile.ID)

	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User: userResponse{
			ID:       profile.ID,
			Username: profile.Username,
			Email:    profile.Email,
		},
	})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshTokenInput
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_INPUT", Message: "Invalid request body"},
		})
	}
	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorBody{Code: "MISSING_FIELDS", Message: "Refresh token is required"},
		})
	}

	hashedToken := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := fmt.Sprintf("%x", hashedToken)

	var storedToken struct {
		ID        string
		UserID    string
		ExpiresAt time.Time
	}

	err := h.DB.QueryRow(
		"SELECT id, user_id, expires_at FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	).Scan(&storedToken.ID, &storedToken.UserID, &storedToken.ExpiresAt)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "INVALID_TOKEN", Message: "Invalid or expired refresh token"},
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Database error"},
		})
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorBody{Code: "EXPIRED_TOKEN", Message: "Refresh token has expired"},
		})
	}

	// Delete old token
	_, _ = h.DB.Exec("DELETE FROM refresh_tokens WHERE id = ?", storedToken.ID)

	// Generate new tokens
	token, err := generateJWT(storedToken.UserID, h.Config.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate token"},
		})
	}

	newRefresh, err := generateRefreshToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to generate refresh token"},
		})
	}

	newHash := sha256.Sum256([]byte(newRefresh))
	_, err = h.DB.Exec(
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		newUUID(), storedToken.UserID, fmt.Sprintf("%x", newHash), time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: "INTERNAL", Message: "Failed to store refresh token"},
		})
	}

	return c.JSON(http.StatusOK, refreshTokenResponse{
		Token:        token,
		RefreshToken: newRefresh,
	})
}

func generateJWT(userID, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate UUID: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}

func formatUUID(b []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}
