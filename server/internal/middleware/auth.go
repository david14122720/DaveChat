package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := ""

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenStr = parts[1]
				}
			}

			if tokenStr == "" {
				tokenStr = c.QueryParam("token")
			}

			if tokenStr == "" {
				return c.JSON(http.StatusUnauthorized, errorResponse{
					Error: errorBody{Code: "AUTH_REQUIRED", Message: "Missing Authorization header or token query param"},
				})
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, errorResponse{
					Error: errorBody{Code: "INVALID_TOKEN", Message: "Invalid or expired token"},
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, errorResponse{
					Error: errorBody{Code: "INVALID_TOKEN", Message: "Invalid token claims"},
				})
			}

			userID, ok := claims["sub"].(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, errorResponse{
					Error: errorBody{Code: "INVALID_TOKEN", Message: "Missing user_id in token"},
				})
			}

			c.Set("user_id", userID)
			return next(c)
		}
	}
}
