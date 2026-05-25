package handlers

import (
	"html/template"
	"net/http"

	"dedsite/internal/db"
)

type Handler struct {
	store             db.Store
	templates         *template.Template
	trustProxyHeaders bool
}

func New(store db.Store, templates *template.Template, trustProxyHeaders bool) Handler {
	return Handler{
		store:             store,
		templates:         templates,
		trustProxyHeaders: trustProxyHeaders,
	}
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
