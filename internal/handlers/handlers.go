package handlers

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"

	"dedsite/internal/db"
)

type Handler struct {
	store     db.Store
	templates *template.Template
}

type SectionResponse struct {
	Site    db.Site
	Section db.Section
}

type ProjectPage struct {
	Site db.Site
	Item db.Item
}

func New(store db.Store, templates *template.Template) Handler {
	return Handler{
		store:     store,
		templates: templates,
	}
}

func (h Handler) Home(w http.ResponseWriter, r *http.Request) {
	site, err := h.store.Site("")
	if err != nil {
		http.Error(w, "Could not load site", http.StatusInternalServerError)
		return
	}

	h.render(w, "home.html", site)
}

func (h Handler) Section(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	section, err := h.store.Section(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Could not load section", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		// HTMX receives a small response: the selected section plus an out-of-band tab refresh.
		site, err := h.store.Site(slug)
		if err != nil {
			http.Error(w, "Could not load site", http.StatusInternalServerError)
			return
		}

		h.render(w, "section_response.html", SectionResponse{
			Site:    site,
			Section: section,
		})
		return
	}

	site, err := h.store.Site(slug)
	if err != nil {
		http.Error(w, "Could not load site", http.StatusInternalServerError)
		return
	}
	h.render(w, "home.html", site)
}

func (h Handler) Project(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	item, err := h.store.Item(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Could not load project", http.StatusInternalServerError)
		return
	}

	site, err := h.store.Site("projects")
	if err != nil {
		http.Error(w, "Could not load site", http.StatusInternalServerError)
		return
	}

	h.render(w, "project.html", ProjectPage{
		Site: site,
		Item: item,
	})
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
