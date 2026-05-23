package db

import "database/sql"

func Migrate(conn *sql.DB) error {
	statements := []string{
		// Cascading deletes keep section item cleanup inside SQLite.
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS profile (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			name TEXT NOT NULL,
			title TEXT NOT NULL,
			email TEXT NOT NULL,
			summary TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id INTEGER NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			subtitle TEXT NOT NULL DEFAULT '',
			period TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			image_alt TEXT NOT NULL DEFAULT '',
			slug TEXT DEFAULT '',
			problem TEXT DEFAULT '',
			built TEXT DEFAULT '',
			learned TEXT DEFAULT '',
			tech_stack TEXT DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS admin_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS admin_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			csrf_hash TEXT NOT NULL DEFAULT '',
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			return err
		}
	}

	// Existing local databases need these optional media fields added in place.
	if err := ensureColumn(conn, "items", "image_url", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "image_alt", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "slug", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "problem", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "built", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "learned", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "items", "tech_stack", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "admin_sessions", "csrf_hash", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := dropProfileLocation(conn); err != nil {
		return err
	}

	return nil
}

func dropProfileLocation(conn *sql.DB) error {
	if !hasColumn(conn, "profile", "location") {
		return nil
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE profile_new (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			name TEXT NOT NULL,
			title TEXT NOT NULL,
			email TEXT NOT NULL,
			summary TEXT NOT NULL
		);
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO profile_new (id, name, title, email, summary)
		SELECT id, name, title, email, summary
		FROM profile;
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`DROP TABLE profile;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE profile_new RENAME TO profile;`); err != nil {
		return err
	}

	return tx.Commit()
}

func ensureColumn(conn *sql.DB, table string, column string, definition string) error {
	rows, err := conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = conn.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func hasColumn(conn *sql.DB, table string, column string) bool {
	rows, err := conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}

	return false
}
