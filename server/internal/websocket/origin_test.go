package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin_AllowedOrigin_ReturnsTrue(t *testing.T) {
	allowed := []string{"https://davechat.example.com", "http://localhost:5173"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://davechat.example.com")

	if !checkOrigin(req, allowed) {
		t.Error("expected allowed origin to return true")
	}
}

func TestCheckOrigin_DisallowedOrigin_ReturnsFalse(t *testing.T) {
	allowed := []string{"https://davechat.example.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://evil.com")

	if checkOrigin(req, allowed) {
		t.Error("expected disallowed origin to return false")
	}
}

func TestCheckOrigin_MissingOrigin_ReturnsFalse(t *testing.T) {
	allowed := []string{"https://davechat.example.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	// No Origin header set

	if checkOrigin(req, allowed) {
		t.Error("expected missing origin to return false when allowedOrigins is non-empty")
	}
}

func TestCheckOrigin_EmptyAllowedOrigins_FallbackToHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Host = "localhost:8080"

	if !checkOrigin(req, nil) {
		t.Error("expected origin matching host to return true")
	}
	if !checkOrigin(req, []string{}) {
		t.Error("expected origin matching host to return true with empty slice")
	}
}

func TestCheckOrigin_EmptyAllowedOrigins_MissingOrigin_ReturnsTrue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	// No Origin header

	if !checkOrigin(req, nil) {
		t.Error("expected missing origin to return true with no allowed origins (unconfigured server)")
	}
}

func TestCheckOrigin_AllowedOriginsWithWhitespace_MatchesCorrectly(t *testing.T) {
	allowed := []string{" https://davechat.example.com ", "http://localhost:5173"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://davechat.example.com")

	if !checkOrigin(req, allowed) {
		t.Error("expected origin to match despite whitespace in allowed list")
	}
}

func TestCheckOrigin_AllowedOrigins_EmptyEntry_Skipped(t *testing.T) {
	allowed := []string{"", "https://davechat.example.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://davechat.example.com")

	if !checkOrigin(req, allowed) {
		t.Error("expected origin to match despite empty entry in allowed list")
	}
}
