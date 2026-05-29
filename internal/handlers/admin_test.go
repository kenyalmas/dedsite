package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"dedsite/internal/db"

	_ "modernc.org/sqlite"
)

func TestAdminCreateEntryNormalizesAndPersistsDeterministically(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	user, err := store.EnsureOAuthAdmin("admin@example.com")
	if err != nil {
		t.Fatalf("EnsureOAuthAdmin() error = %v", err)
	}
	sessionToken, csrfToken, _, err := store.CreateAdminSession(user.ID, time.Hour)
	if err != nil {
		t.Fatalf("CreateAdminSession() error = %v", err)
	}

	form := url.Values{
		"title":      {"Field Notes"},
		"tech_stack": {" Go, HTMX, , SQLite "},
		"tags":       {" Security, RF , , Testing "},
		"url":        {"javascript:alert(1)"},
		"csrf_token": {csrfToken},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/sections/projects/entries", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "dedsite_admin", Value: sessionToken, Path: "/admin"})
	req.AddCookie(&http.Cookie{Name: "dedsite_csrf", Value: csrfToken, Path: "/admin"})
	req.SetPathValue("slug", "projects")

	rec := httptest.NewRecorder()
	handler.AdminCreateEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("AdminCreateEntry() status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "entry added to Projects") {
		t.Fatalf("AdminCreateEntry() body did not include success message: %q", rec.Body.String())
	}

	item, err := store.Item("field-notes")
	if err != nil {
		t.Fatalf("Item() error = %v", err)
	}

	if item.URL != "" {
		t.Fatalf("stored URL = %q, want empty string", item.URL)
	}
	if !reflect.DeepEqual(item.TechStack, []string{"Go", "HTMX", "SQLite"}) {
		t.Fatalf("stored tech stack = %v", item.TechStack)
	}
	if !reflect.DeepEqual(item.Tags, []string{"Security", "RF", "Testing"}) {
		t.Fatalf("stored tags = %v", item.Tags)
	}
}

func TestSectionHTMXResponseRendersActiveTabAndSectionContent(t *testing.T) {
	handler, store := newSectionTestHandler(t)

	if err := store.AddItem("projects", db.Item{
		Slug:        "field-notes",
		Title:       "Field Notes",
		Description: "A deterministic section test entry.",
		Tags:        []string{"Security"},
	}); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/section/projects", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("slug", "projects")

	rec := httptest.NewRecorder()
	handler.Section(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("Section() status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `hx-swap-oob="true"`) {
		t.Fatalf("Section() body missing HTMX OOB tabs: %q", body)
	}
	if !strings.Contains(body, `class="tab is-active"`) || !strings.Contains(body, `href="/section/projects"`) {
		t.Fatalf("Section() body missing active projects tab: %q", body)
	}
	if !strings.Contains(body, "<h2>Projects</h2>") || !strings.Contains(body, "Field Notes") {
		t.Fatalf("Section() body missing section content: %q", body)
	}
}

func newAdminTestHandler(t *testing.T) (Handler, db.Store) {
	t.Helper()

	conn := newAdminTestDB(t)
	store := db.NewStore(conn)
	seedAdminTestData(t, conn)

	templates, err := template.ParseFiles(filepath.Join("..", "..", "templates", "partials", "admin_entry_form.html"))
	if err != nil {
		t.Fatalf("ParseFiles() error = %v", err)
	}

	return New(store, templates, false, GoogleOAuthConfig{}), store
}

func newSectionTestHandler(t *testing.T) (Handler, db.Store) {
	t.Helper()

	conn := newAdminTestDB(t)
	store := db.NewStore(conn)
	seedAdminTestData(t, conn)

	insertSectionForHandlerTest(t, conn, "work", "Experience", 0)
	updateSectionOrderForHandlerTest(t, conn, "projects", 1)

	templates, err := template.ParseFiles(
		filepath.Join("..", "..", "templates", "partials", "section_response.html"),
		filepath.Join("..", "..", "templates", "partials", "section.html"),
	)
	if err != nil {
		t.Fatalf("ParseFiles() error = %v", err)
	}

	return New(store, templates, false, GoogleOAuthConfig{}), store
}

func newAdminTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "admin-test.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	if err := db.Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return conn
}

func seedAdminTestData(t *testing.T, conn *sql.DB) {
	t.Helper()

	_, err := conn.Exec(`
		INSERT INTO profile (id, name, title, email, summary)
		VALUES (1, 'Test User', 'Tester', 'test@example.com', 'summary');
		INSERT INTO sections (slug, title, sort_order)
		VALUES ('projects', 'Projects', 0);
	`)
	if err != nil {
		t.Fatalf("seed admin test data error = %v", err)
	}
}

func insertSectionForHandlerTest(t *testing.T, conn *sql.DB, slug string, title string, sortOrder int) {
	t.Helper()

	_, err := conn.Exec(`
		INSERT INTO sections (slug, title, sort_order)
		VALUES (?, ?, ?)
	`, slug, title, sortOrder)
	if err != nil {
		t.Fatalf("insert section %q error = %v", slug, err)
	}
}

func updateSectionOrderForHandlerTest(t *testing.T, conn *sql.DB, slug string, sortOrder int) {
	t.Helper()

	_, err := conn.Exec(`UPDATE sections SET sort_order = ? WHERE slug = ?`, sortOrder, slug)
	if err != nil {
		t.Fatalf("update section %q order error = %v", slug, err)
	}
}
