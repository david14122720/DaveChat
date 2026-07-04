package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAuthMiddleware_RejectsTokenQueryParam(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/protected?token=some.jwt.token", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := AuthMiddleware("test-secret")
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err != nil {
		t.Fatalf("expected error response via JSON, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != `{"error":{"code":"AUTH_REQUIRED","message":"Authorization header required"}}`+"\n" {
		t.Errorf("unexpected body: %s", body)
	}
}
