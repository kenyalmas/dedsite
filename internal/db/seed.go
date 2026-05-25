package db

import "database/sql"

var defaultProfile = Profile{
	Name:    "Kenneth Almas",
	Title:   "Security Researcher | Hardware Enthusiast",
	Email:   "kennethalmas232@gmail.com",
	Summary: "I am a person of intrigue. I enjoy taking things apart & understanding how they work. I have a strong interest in security research, particularly in IOT and embedded systems.",
}

type defaultSection struct {
	slug  string
	title string
	items []Item
}

var defaultSections = []defaultSection{
	{
		slug:  "work",
		title: "Experience",
		items: []Item{
			{
				Slug:        "aas-hardware-software-support",
				Title:       "AAS in Hardware & Software Support",
				Subtitle:    "Sandhill Community College",
				Period:      "2024 - 2026",
				Description: "Graduate with honors.",
				Tags:        []string{"Hardware", "Software"},
			},
		},
	},
	{
		slug:  "projects",
		title: "Projects",
		items: []Item{
			{
				Slug:        "manet-autonomous-drone-coordination",
				Title:       "MaNet for autonomous drone coordination",
				Subtitle:    "Personal Work",
				Period:      "2025 - ongoing",
				Description: "Private communication network for autonomous drone coordination based off of 802.11ah standard. Purpose: gather greater understanding of protocol standards and RF communications troubleshooting.",
				URL:         "[PRIVATE]",
				Problem:     "Autonomous systems need a resilient local coordination layer that can keep nodes talking when normal infrastructure is unavailable, unreliable, or intentionally out of scope.",
				Built:       "Designed a private mobile ad hoc network concept around 802.11ah constraints, mapped the coordination requirements, and documented RF troubleshooting paths for range, interference, and device discovery.",
				Learned:     "The strongest lesson so far is that protocol work is equal parts standards reading, environmental testing, and failure-mode logging. RF behavior becomes easier to reason about when every assumption is tied to a measurement.",
				TechStack:   []string{"802.11ah", "RF", "Networking", "Embedded Systems", "Go"},
				Tags:        []string{"RF", "Networking"},
			},
		},
	},
	{
		slug:  "security",
		title: "Security",
		items: []Item{
			{
				Slug:        "ftcc-ncng-ctf-champion",
				Title:       "FTCC X NCNG CTF Champion",
				Description: "I was part of the team that won the 2026 FTCC X NCNG CTF, which featured a variety of challenges in areas such as reverse engineering, packet analysis, and cryptography.",
				Tags:        []string{"CTF", "Security", "Networking"},
			},
		},
	},
	{
		slug:  "ai",
		title: "AI",
		items: []Item{
			{
				Slug:        "ai-vision-research-program",
				Title:       "AI Vision Research Program",
				Description: "Trained visual AI models on custom datasets & Researched the capabilities of neural networks in limited resource environments.",
				Tags:        []string{"Machine Learning", "AI Training"},
			},
		},
	},
}

func (s Store) SeedDefaults() error {
	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := seedProfile(tx); err != nil {
		return err
	}
	if err := seedSections(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sections WHERE slug = ?`, "dev"); err != nil {
		return err
	}

	return tx.Commit()
}

func (s Store) EnsureDefaultSections() error {
	return s.SeedDefaults()
}

func seedProfile(conn itemInserter) error {
	_, err := conn.Exec(`
		INSERT INTO profile (id, name, title, email, summary)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			title = excluded.title,
			email = excluded.email,
			summary = excluded.summary
	`, defaultProfile.Name, defaultProfile.Title, defaultProfile.Email, defaultProfile.Summary)
	return err
}

func seedSections(conn itemInserter) error {
	for sectionIndex, section := range defaultSections {
		if _, err := conn.Exec(`
			INSERT INTO sections (slug, title, sort_order)
			VALUES (?, ?, ?)
			ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
		`, section.slug, section.title, sectionIndex); err != nil {
			return err
		}

		for itemIndex, item := range section.items {
			if err := ensureItem(conn, section.slug, item, itemIndex); err != nil {
				return err
			}
		}
	}
	return nil
}

type itemInserter interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func ensureItem(conn itemInserter, sectionSlug string, item Item, sortOrder int) error {
	var sectionID int64
	if err := conn.QueryRow(`SELECT id FROM sections WHERE slug = ?`, sectionSlug).Scan(&sectionID); err != nil {
		return err
	}

	if item.Slug != "" {
		var existingID int64
		err := conn.QueryRow(`SELECT id FROM items WHERE section_id = ? AND slug = ? ORDER BY id LIMIT 1`, sectionID, item.Slug).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			if _, err := conn.Exec(`DELETE FROM items WHERE section_id = ? AND slug = ? AND id <> ?`, sectionID, item.Slug, existingID); err != nil {
				return err
			}
			_, err := conn.Exec(`
				UPDATE items
				SET title = ?, subtitle = ?, period = ?, description = ?, url = ?, image_url = ?, image_alt = ?, problem = ?, built = ?, learned = ?, tech_stack = ?, tags = ?, sort_order = ?
				WHERE id = ?
			`,
				item.Title,
				item.Subtitle,
				item.Period,
				item.Description,
				item.URL,
				item.ImageURL,
				item.ImageAlt,
				item.Problem,
				item.Built,
				item.Learned,
				joinValues(item.TechStack),
				joinValues(item.Tags),
				sortOrder,
				existingID,
			)
			return err
		}
	}

	if item.Slug == "" {
		return nil
	}

	_, err := conn.Exec(`
		INSERT INTO items (section_id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sectionID,
		item.Slug,
		item.Title,
		item.Subtitle,
		item.Period,
		item.Description,
		item.URL,
		item.ImageURL,
		item.ImageAlt,
		item.Problem,
		item.Built,
		item.Learned,
		joinValues(item.TechStack),
		joinValues(item.Tags),
		sortOrder,
	)
	return err
}
