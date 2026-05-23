package handlers

import (
	"database/sql"
	"dedsite/internal/auth"
	"errors"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	Site      db.Site
	CSRFToken string
}

type EntryFormPage struct {
	Section   db.Section
	Item      db.Item
	Error     string
	Success   string
	CSRFToken string
	IsEdit    bool
}

type AdminSectionsPage struct {
	Site      db.Site
	CSRFToken string
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

var (
	adminLoginLimiterMu sync.Mutex
	adminLoginAttempts  = map[string]loginAttempt{}
)

type loginAttempt struct {
	Count     int
	BlockedTo time.Time
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
	if blocked, wait := adminLoginBlocked(loginKey(r, username)); blocked {
		w.WriteHeader(http.StatusTooManyRequests)
		h.render(w, "admin_login.html", LoginPage{Error: "too many login attempts; wait " + wait})
		return
	}
	user, ok, err := h.store.AuthenticateAdmin(username, password)
	if err != nil {
		http.Error(w, "Could not validate login", http.StatusInternalServerError)
		return
	}
	if !ok {
		recordAdminLoginFailure(loginKey(r, username))
		w.WriteHeader(http.StatusUnauthorized)
		h.render(w, "admin_login.html", LoginPage{Error: "invalid username or password"})
		return
	}
	clearAdminLoginFailures(loginKey(r, username))

	token, csrfToken, expires, err := h.store.CreateAdminSession(user.ID, 12*time.Hour)
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
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "dedsite_csrf",
		Value:    csrfToken,
		Path:     "/admin",
		Expires:  expires,
		HttpOnly: false,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h Handler) Admin(w http.ResponseWriter, r *http.Request) {
	session, ok, err := h.currentAdminSession(r)
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

	csrfToken, err := h.ensureCSRFToken(w, r, session)
	if err != nil {
		http.Error(w, "Could not prepare admin session", http.StatusInternalServerError)
		return
	}
	h.render(w, "admin.html", AdminPage{
		Site:      site,
		CSRFToken: csrfToken,
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

	h.render(w, "admin_entry_form.html", EntryFormPage{
		Section:   section,
		CSRFToken: h.csrfToken(r),
	})
}

func (h Handler) AdminEditEntryForm(w http.ResponseWriter, r *http.Request) {
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

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	item, err := h.store.ItemByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Could not load entry", http.StatusInternalServerError)
		return
	}

	h.render(w, "admin_entry_form.html", EntryFormPage{
		Section:   section,
		Item:      item,
		CSRFToken: h.csrfToken(r),
		IsEdit:    true,
	})
}

func (h Handler) AdminCreateEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	if !h.requireCSRF(w, r) {
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
			Section:   section,
			Item:      item,
			Error:     "title is required",
			CSRFToken: h.csrfToken(r),
		})
		return
	}

	if item.Slug == "" {
		item.Slug = slugify(item.Title)
	}
	item.URL = normalizeAllowedURL(item.URL)
	item.ImageURL = normalizeAllowedURL(item.ImageURL)

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
		Section:   section,
		Success:   "entry added to " + section.Title,
		CSRFToken: h.csrfToken(r),
	})
}

func (h Handler) AdminUpdateEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	if !h.requireCSRF(w, r) {
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

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not read entry form", http.StatusBadRequest)
		return
	}

	item := itemFromForm(r)
	item.ID = id
	if item.Title == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.render(w, "admin_entry_form.html", EntryFormPage{
			Section:   section,
			Item:      item,
			Error:     "title is required",
			CSRFToken: h.csrfToken(r),
			IsEdit:    true,
		})
		return
	}
	if item.Slug == "" {
		item.Slug = slugify(item.Title)
	}
	item.URL = normalizeAllowedURL(item.URL)
	item.ImageURL = normalizeAllowedURL(item.ImageURL)

	if err := h.store.UpdateItem(section.Slug, item); err != nil {
		http.Error(w, "Could not update entry", http.StatusInternalServerError)
		return
	}

	item, err = h.store.ItemByID(id)
	if err != nil {
		http.Error(w, "Could not reload entry", http.StatusInternalServerError)
		return
	}

	h.render(w, "admin_entry_form.html", EntryFormPage{
		Section:   section,
		Item:      item,
		Success:   "entry updated in " + section.Title,
		CSRFToken: h.csrfToken(r),
		IsEdit:    true,
	})
}

func (h Handler) AdminDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	if !h.requireCSRF(w, r) {
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	if err := h.store.DeleteItem(id); err != nil {
		http.Error(w, "Could not delete entry", http.StatusInternalServerError)
		return
	}

	site, err := h.store.Site("")
	if err != nil {
		http.Error(w, "Could not reload entries", http.StatusInternalServerError)
		return
	}

	h.render(w, "admin_sections_list.html", AdminSectionsPage{
		Site:      site,
		CSRFToken: h.csrfToken(r),
	})
}

func (h Handler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	if !h.requireCSRF(w, r) {
		return
	}

	cookie, _ := r.Cookie("dedsite_admin")
	if cookie != nil {
		if err := h.store.DeleteAdminSession(cookie.Value); err != nil {
			http.Error(w, "Could not log out", http.StatusInternalServerError)
			return
		}
	}

	expireCookie(w, "dedsite_admin", true, isSecureRequest(r))
	expireCookie(w, "dedsite_csrf", false, isSecureRequest(r))
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (h Handler) currentAdminSession(r *http.Request) (db.AdminSession, bool, error) {
	cookie, err := r.Cookie("dedsite_admin")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return db.AdminSession{}, false, nil
		}
		return db.AdminSession{}, false, err
	}

	return h.store.AdminSessionForToken(cookie.Value)
}

func (h Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	_, ok, err := h.currentAdminSession(r)
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

func (h Handler) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	session, ok, err := h.currentAdminSession(r)
	if err != nil {
		http.Error(w, "Could not read session", http.StatusInternalServerError)
		return false
	}
	if !ok {
		w.Header().Set("HX-Redirect", "/admin/login")
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}

	token := r.FormValue("csrf_token")
	if token == "" {
		token = r.Header.Get("X-CSRF-Token")
	}
	if token == "" || auth.HashToken(token) != session.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return false
	}
	return true
}

func (h Handler) csrfToken(r *http.Request) string {
	cookie, err := r.Cookie("dedsite_csrf")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (h Handler) ensureCSRFToken(w http.ResponseWriter, r *http.Request, session db.AdminSession) (string, error) {
	token := h.csrfToken(r)
	if token != "" && auth.HashToken(token) == session.CSRFToken {
		return token, nil
	}

	token, err := auth.RandomToken()
	if err != nil {
		return "", err
	}

	sessionCookie, err := r.Cookie("dedsite_admin")
	if err != nil {
		return "", err
	}
	if err := h.store.SetAdminSessionCSRF(sessionCookie.Value, token); err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dedsite_csrf",
		Value:    token,
		Path:     "/admin",
		Expires:  session.ExpiresAt.UTC(),
		HttpOnly: false,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}

func expireCookie(w http.ResponseWriter, name string, httpOnly bool, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
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

func normalizeAllowedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return raw
	default:
		return ""
	}
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func loginKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.ToLower(strings.TrimSpace(username)) + "|" + host
}

func adminLoginBlocked(key string) (bool, string) {
	adminLoginLimiterMu.Lock()
	defer adminLoginLimiterMu.Unlock()

	attempt := adminLoginAttempts[key]
	now := time.Now()
	if attempt.BlockedTo.After(now) {
		return true, attempt.BlockedTo.Sub(now).Round(time.Second).String()
	}
	if !attempt.BlockedTo.IsZero() && !attempt.BlockedTo.After(now) {
		delete(adminLoginAttempts, key)
	}
	return false, ""
}

func recordAdminLoginFailure(key string) {
	adminLoginLimiterMu.Lock()
	defer adminLoginLimiterMu.Unlock()

	attempt := adminLoginAttempts[key]
	attempt.Count++
	if attempt.Count >= 5 {
		backoff := time.Duration(attempt.Count-4) * 15 * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		attempt.BlockedTo = time.Now().Add(backoff)
	}
	adminLoginAttempts[key] = attempt
}

func clearAdminLoginFailures(key string) {
	adminLoginLimiterMu.Lock()
	defer adminLoginLimiterMu.Unlock()
	delete(adminLoginAttempts, key)
}

func (h Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not render page", http.StatusInternalServerError)
	}
}
