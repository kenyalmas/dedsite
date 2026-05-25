package handlers

import (
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
	ClientID      string
	ClientSecret  string
	AllowedEmails []string
}

type googleOAuthConfig struct {
	clientID      string
	clientSecret  string
	allowedEmails map[string]bool
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
	allowed := make(map[string]bool, len(config.AllowedEmails))
	for _, email := range config.AllowedEmails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" {
			allowed[email] = true
		}
	}
	return googleOAuthConfig{
		clientID:      config.ClientID,
		clientSecret:  config.ClientSecret,
		allowedEmails: allowed,
	}
}

func (c googleOAuthConfig) enabled() bool {
	return c.clientID != "" && c.clientSecret != "" && len(c.allowedEmails) > 0
}

func (c googleOAuthConfig) allows(email string) bool {
	return c.allowedEmails[email]
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
