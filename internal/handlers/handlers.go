package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"

	"dedsite/internal/db"
)

type Handler struct {
	store             db.Store
	templates         *template.Template
	trustProxyHeaders bool
	googleOAuth       googleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID           string
	ClientSecret       string
	AllowedEmailHashes []string
	PublicOrigin       string
}

type googleOAuthConfig struct {
	clientID           string
	clientSecret       string
	allowedEmailHashes map[string]bool
	publicOrigin       string
}

func New(store db.Store, templates *template.Template, trustProxyHeaders bool, googleOAuth GoogleOAuthConfig) Handler {
	return Handler{
		store:             store,
		templates:         templates,
		trustProxyHeaders: trustProxyHeaders,
		googleOAuth:       newGoogleOAuthConfig(googleOAuth),
	}
}

func newGoogleOAuthConfig(config GoogleOAuthConfig) googleOAuthConfig {
	allowed := make(map[string]bool, len(config.AllowedEmailHashes))
	for _, hash := range config.AllowedEmailHashes {
		hash = strings.ToLower(strings.TrimSpace(hash))
		if hash != "" {
			allowed[hash] = true
		}
	}
	return googleOAuthConfig{
		clientID:           config.ClientID,
		clientSecret:       config.ClientSecret,
		allowedEmailHashes: allowed,
		publicOrigin:       normalizePublicOrigin(config.PublicOrigin),
	}
}

func (c googleOAuthConfig) enabled() bool {
	return c.clientID != "" && c.clientSecret != "" && len(c.allowedEmailHashes) > 0
}

func (c googleOAuthConfig) allows(email string) bool {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return c.allowedEmailHashes[hex.EncodeToString(sum[:])]
}

func normalizePublicOrigin(origin string) string {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return ""
	}
	if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
		return origin
	}
	return "https://" + origin
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
