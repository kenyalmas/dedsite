package handlers

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

func TestIsSecureRequest(t *testing.T) {
	t.Run("tls request is always secure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.test", nil)
		req.TLS = &tls.ConnectionState{}
		if !isSecureRequest(req, false) {
			t.Fatal("expected TLS request to be secure")
		}
	})

	t.Run("forwarded proto ignored when trust disabled", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		if isSecureRequest(req, false) {
			t.Fatal("expected forwarded proto to be ignored when trust is disabled")
		}
	})

	t.Run("forwarded proto honored when trust enabled", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		if !isSecureRequest(req, true) {
			t.Fatal("expected forwarded proto to be honored when trust is enabled")
		}
	})
}
