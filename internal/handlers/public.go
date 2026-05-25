package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"dedsite/internal/db"
)

type SectionResponse struct {
	Site    db.Site
	Section db.Section
}

type ProjectPage struct {
	Site db.Site
	Item db.Item
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

func (h Handler) Awareness(w http.ResponseWriter, r *http.Request) {
	h.render(w, "awareness.html", nil)
}

func (h Handler) PasswordRoast(w http.ResponseWriter, r *http.Request) {
	h.render(w, "password_roast.html", nil)
}

func (h Handler) AwarenessData(w http.ResponseWriter, r *http.Request) {
	ip := requestIP(r, h.trustProxyHeaders)
	payload := map[string]string{
		"ip":                 ip,
		"sec_ch_ua_mobile":   r.Header.Get("Sec-CH-UA-Mobile"),
		"sec_ch_ua_platform": r.Header.Get("Sec-CH-UA-Platform"),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
	}
}

func requestIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
			parts := strings.Split(value, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
		if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
			return value
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
