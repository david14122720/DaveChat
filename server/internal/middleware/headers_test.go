package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSecurityHeadersMiddleware_Production_HeadersPresent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMiddleware(false)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	expectedHeaders := map[string]string{
		"Content-Security-Policy-Report-Only": "default-src 'self'; img-src 'self' data: blob:; media-src 'self'; connect-src 'self' ws: wss:; style-src 'self' 'unsafe-inline'; script-src 'self'",
		"Strict-Transport-Security":           "max-age=31536000; includeSubDomains",
		"X-Frame-Options":                     "DENY",
		"X-Content-Type-Options":              "nosniff",
		"Referrer-Policy":                     "strict-origin-when-cross-origin",
		"Permissions-Policy":                  "camera=(), microphone=(), geolocation=()",
	}

	for header, expectedValue := range expectedHeaders {
		got := rec.Header().Get(header)
		if got != expectedValue {
			t.Errorf("header %q = %q, want %q", header, got, expectedValue)
		}
	}

	// Verify Content-Type from the handler still works
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSecurityHeadersMiddleware_DevMode_NoHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMiddleware(true)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatal(err)
	}

	securityHeaders := []string{
		"Content-Security-Policy-Report-Only",
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range securityHeaders {
		if v := rec.Header().Get(header); v != "" {
			t.Errorf("header %q should be empty in dev mode, got %q", header, v)
		}
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
