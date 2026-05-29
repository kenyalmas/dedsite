package db

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSectionsLoadsItemsInSectionOrder(t *testing.T) {
	store := newTestStore(t)
	seedTestProfile(t, store.conn)

	insertSection(t, store.conn, "work", "Experience", 0)
	insertSection(t, store.conn, "empty", "Empty", 1)
	insertSection(t, store.conn, "projects", "Projects", 2)

	insertItem(t, store.conn, "projects", "beta", "Beta", 1, "Go, HTMX", "tools, security")
	insertItem(t, store.conn, "projects", "alpha", "Alpha", 0, "SQLite", "infra")
	insertItem(t, store.conn, "work", "first-role", "First Role", 0, "", "")

	sections, err := store.Sections()
	if err != nil {
		t.Fatalf("Sections() error = %v", err)
	}

	if got, want := len(sections), 3; got != want {
		t.Fatalf("len(Sections()) = %d, want %d", got, want)
	}

	if sections[0].Slug != "work" || sections[1].Slug != "empty" || sections[2].Slug != "projects" {
		t.Fatalf("section order = %q, %q, %q", sections[0].Slug, sections[1].Slug, sections[2].Slug)
	}

	if got, want := len(sections[1].Items), 0; got != want {
		t.Fatalf("empty section item count = %d, want %d", got, want)
	}

	projectTitles := []string{sections[2].Items[0].Title, sections[2].Items[1].Title}
	if !reflect.DeepEqual(projectTitles, []string{"Alpha", "Beta"}) {
		t.Fatalf("project item order = %v, want [Alpha Beta]", projectTitles)
	}

	if !reflect.DeepEqual(sections[2].Items[1].TechStack, []string{"Go", "HTMX"}) {
		t.Fatalf("project tech stack = %v", sections[2].Items[1].TechStack)
	}
}

func TestSectionRefReturnsMetadata(t *testing.T) {
	store := newTestStore(t)
	seedTestProfile(t, store.conn)
	insertSection(t, store.conn, "projects", "Projects", 0)
	insertItem(t, store.conn, "projects", "alpha", "Alpha", 0, "Go", "security")

	section, err := store.SectionRef("projects")
	if err != nil {
		t.Fatalf("SectionRef() error = %v", err)
	}

	if section.ID == 0 {
		t.Fatal("expected section ID to be populated")
	}
	if section.Slug != "projects" || section.Title != "Projects" {
		t.Fatalf("SectionRef() = %+v", section)
	}

	full, err := store.Section("projects")
	if err != nil {
		t.Fatalf("Section() error = %v", err)
	}
	if got, want := len(full.Items), 1; got != want {
		t.Fatalf("len(Section().Items) = %d, want %d", got, want)
	}
}

func newTestStore(t *testing.T) Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return NewStore(conn)
}

func seedTestProfile(t *testing.T, conn *sql.DB) {
	t.Helper()

	_, err := conn.Exec(`
		INSERT INTO profile (id, name, title, email, summary)
		VALUES (1, 'Test User', 'Tester', 'test@example.com', 'summary')
	`)
	if err != nil {
		t.Fatalf("insert profile error = %v", err)
	}
}

func insertSection(t *testing.T, conn *sql.DB, slug string, title string, sortOrder int) {
	t.Helper()

	_, err := conn.Exec(`
		INSERT INTO sections (slug, title, sort_order)
		VALUES (?, ?, ?)
	`, slug, title, sortOrder)
	if err != nil {
		t.Fatalf("insert section %q error = %v", slug, err)
	}
}

func insertItem(t *testing.T, conn *sql.DB, sectionSlug string, slug string, title string, sortOrder int, techStack string, tags string) {
	t.Helper()

	var sectionID int64
	if err := conn.QueryRow(`SELECT id FROM sections WHERE slug = ?`, sectionSlug).Scan(&sectionID); err != nil {
		t.Fatalf("lookup section %q error = %v", sectionSlug, err)
	}

	_, err := conn.Exec(`
		INSERT INTO items (section_id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags, sort_order)
		VALUES (?, ?, ?, '', '', '', '', '', '', '', '', '', ?, ?, ?)
	`, sectionID, slug, title, techStack, tags, sortOrder)
	if err != nil {
		t.Fatalf("insert item %q error = %v", slug, err)
	}
}
