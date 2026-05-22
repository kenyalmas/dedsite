package handlers

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

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

type LoginPage struct {
	Error string
}

type AdminPage struct {
	User      db.AdminUser
	Site      db.Site
	ItemCount int
	TagCount  int
}

type EntryFormPage struct {
	Section db.Section
	Item    db.Item
	Error   string
	Success string
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

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

func (h Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.render(w, "admin_login.html", LoginPage{})
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not read login form", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	user, ok, err := h.store.AuthenticateAdmin(username, password)
	if err != nil {
		http.Error(w, "Could not validate login", http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		h.render(w, "admin_login.html", LoginPage{Error: "invalid username or password"})
		return
	}

	token, expires, err := h.store.CreateAdminSession(user.ID, 12*time.Hour)
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dedsite_admin",
		Value:    token,
		Path:     "/admin",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h Handler) Admin(w http.ResponseWriter, r *http.Request) {
	user, ok, err := h.currentAdmin(r)
	if err != nil {
		http.Error(w, "Could not read session", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}

	site, err := h.store.Site("")
	if err != nil {
		http.Error(w, "Could not load dashboard", http.StatusInternalServerError)
		return
	}

	itemCount, tagCount := adminCounts(site)
	h.render(w, "admin.html", AdminPage{
		User:      user,
		Site:      site,
		ItemCount: itemCount,
		TagCount:  tagCount,
	})
}

func (h Handler) AdminEntryForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	section, err := h.store.Section(r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Could not load section", http.StatusInternalServerError)
		return
	}

	h.render(w, "admin_entry_form.html", EntryFormPage{Section: section})
}

func (h Handler) AdminCreateEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	section, err := h.store.Section(r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Could not load section", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not read entry form", http.StatusBadRequest)
		return
	}

	item := itemFromForm(r)
	if item.Title == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.render(w, "admin_entry_form.html", EntryFormPage{
			Section: section,
			Item:    item,
			Error:   "title is required",
		})
		return
	}

	if item.Slug == "" {
		item.Slug = slugify(item.Title)
	}

	if err := h.store.AddItem(section.Slug, item); err != nil {
		http.Error(w, "Could not save entry", http.StatusInternalServerError)
		return
	}

	section, err = h.store.Section(section.Slug)
	if err != nil {
		http.Error(w, "Could not reload section", http.StatusInternalServerError)
		return
	}

	h.render(w, "admin_entry_form.html", EntryFormPage{
		Section: section,
		Success: "entry added to " + section.Title,
	})
}

func (h Handler) currentAdmin(r *http.Request) (db.AdminUser, bool, error) {
	cookie, err := r.Cookie("dedsite_admin")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return db.AdminUser{}, false, nil
		}
		return db.AdminUser{}, false, err
	}

	return h.store.AdminUserForSession(cookie.Value)
}

func (h Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	_, ok, err := h.currentAdmin(r)
	if err != nil {
		http.Error(w, "Could not read session", http.StatusInternalServerError)
		return false
	}
	if !ok {
		w.Header().Set("HX-Redirect", "/admin/login")
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}
	return true
}

func adminCounts(site db.Site) (int, int) {
	itemCount := 0
	tags := map[string]bool{}
	for _, section := range site.Sections {
		itemCount += len(section.Items)
		for _, item := range section.Items {
			for _, tag := range item.Tags {
				tags[tag] = true
			}
		}
	}
	return itemCount, len(tags)
}

func itemFromForm(r *http.Request) db.Item {
	return db.Item{
		Slug:        strings.TrimSpace(r.FormValue("slug")),
		Title:       strings.TrimSpace(r.FormValue("title")),
		Subtitle:    strings.TrimSpace(r.FormValue("subtitle")),
		Period:      strings.TrimSpace(r.FormValue("period")),
		Description: strings.TrimSpace(r.FormValue("description")),
		URL:         strings.TrimSpace(r.FormValue("url")),
		ImageURL:    strings.TrimSpace(r.FormValue("image_url")),
		ImageAlt:    strings.TrimSpace(r.FormValue("image_alt")),
		Problem:     strings.TrimSpace(r.FormValue("problem")),
		Built:       strings.TrimSpace(r.FormValue("built")),
		Learned:     strings.TrimSpace(r.FormValue("learned")),
		TechStack:   splitCSV(r.FormValue("tech_stack")),
		Tags:        splitCSV(r.FormValue("tags")),
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = slugPattern.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
