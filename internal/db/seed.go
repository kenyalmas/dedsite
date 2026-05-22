package db

import "database/sql"

func (s Store) SeedDefaults() error {
	var count int
	if err := s.conn.QueryRow(`SELECT COUNT(*) FROM profile`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		// Existing databases are synchronized instead of recreated so local content survives.
		return s.EnsureDefaultSections()
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO profile (id, name, title, email, summary)
		VALUES (1, ?, ?, ?, ?)
	`,
		"Kenneth Almas",
		"Security Researcher | Hardware Enthusiast",
		"kennethalmas232@gmail.com",
		"I am a person of intrigue. I enjoy taking things apart & understanding how they work. I have a strong interest in security research, particularly in IOT and embedded systems.",
	); err != nil {
		return err
	}

	sections := []struct {
		slug  string
		title string
		items []Item
	}{
		{
			slug:  "work",
			title: "Experience",
			items: []Item{
				{
					Title:       "Associates in Applied Science in Hardware & Software Support",
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
					Title:       "MaNet for autonomous drone coordination",
					Subtitle:    "Personal Work",
					Period:      "2025 - ongoing",
					Description: "Private communication network for autonomous drone coordination based off of 802.11ah standard. Purpose: gather greater understanding of protocol standards and RF communications troubleshooting.",
					URL:         "[PRIVATE]",
					Tags:        []string{"RF", "Networking"},
				},
			},
		},
		{
			slug:  "security",
			title: "Security",
			items: []Item{
				{
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
					Title:       "AI Vision Research Program",
					Description: "Trained visual AI models on custom datasets & Researched the capabilities of neural networks in limited resource environments.",
					Tags:        []string{"Machine Learning", "AI Training"},
				},
			},
		},
	}

	for sectionIndex, section := range sections {
		result, err := tx.Exec(`INSERT INTO sections (slug, title, sort_order) VALUES (?, ?, ?)`, section.slug, section.title, sectionIndex)
		if err != nil {
			return err
		}

		sectionID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		for itemIndex, item := range section.items {
			if _, err := tx.Exec(`
				INSERT INTO items (section_id, title, subtitle, period, description, url, image_url, image_alt, tags, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				sectionID,
				item.Title,
				item.Subtitle,
				item.Period,
				item.Description,
				item.URL,
				item.ImageURL,
				item.ImageAlt,
				joinTags(item.Tags),
				itemIndex,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s Store) EnsureDefaultSections() error {
	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE profile
		SET name = ?, title = ?, email = ?, summary = ?
		WHERE id = 1
	`,
		"Kenneth Almas",
		"Hacker | Hardware Enthusiast",
		"kennethalmas232@gmail.com",
		"I am a person of intrigue with a deep passion for electronics. I enjoy taking things apart & understanding how they work. I have a strong interest in security research, particularly in the realm of hardware and embedded systems. I am always eager to learn and explore new technologies, and I thrive on solving complex problems.",
	); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO sections (slug, title, sort_order)
		VALUES (?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
	`, "work", "Experience", 0); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO sections (slug, title, sort_order)
		VALUES (?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
	`, "projects", "Projects", 1); err != nil {
		return err
	}

	if err := ensureItem(tx, "work", Item{
		Title:       "Associates in Applied Science in Hardware & Software Support",
		Subtitle:    "Sandhill Community College",
		Period:      "2024 - 2026",
		Description: "Graduate with honors.",
		Tags:        []string{"Hardware", "Software"},
	}, 1); err != nil {
		return err
	}

	if err := ensureItem(tx, "projects", Item{
		Title:       "MaNet for autonomous drone coordination",
		Subtitle:    "Personal Work",
		Period:      "2025 - ongoing",
		Description: "Private communication network for autonomous drone coordination based off of 802.11ah standard. Purpose: gather greater understanding of protocol standards and RF communications troubleshooting.",
		URL:         "[PRIVATE]",
		Tags:        []string{"RF", "Networking"},
	}, 0); err != nil {
		return err
	}

	sections := []struct {
		slug        string
		title       string
		sortOrder   int
		placeholder Item
	}{
		{
			slug:      "security",
			title:     "Security",
			sortOrder: 2,
			placeholder: Item{
				Title:       "FTCC X NCNG CTF Champion",
				Description: "I was part of the team that won the 2026 FTCC X NCNG CTF, which featured a variety of challenges in areas such as reverse engineering, packet analysis, and cryptography.",
				Tags:        []string{"CTF", "Security", "Networking"},
			},
		},
		{
			slug:      "ai",
			title:     "AI",
			sortOrder: 3,
			placeholder: Item{
				Title:       "AI Vision Research Program",
				Description: "Trained visual AI models on custom datasets & Researched the capabilities of neural networks in limited resource environments.",
				Tags:        []string{"Machine Learning", "AI Training"},
			},
		},
	}

	for _, section := range sections {
		if _, err := tx.Exec(`
			INSERT INTO sections (slug, title, sort_order)
			VALUES (?, ?, ?)
			ON CONFLICT(slug) DO UPDATE SET title = excluded.title, sort_order = excluded.sort_order
		`, section.slug, section.title, section.sortOrder); err != nil {
			return err
		}

		var sectionID int64
		if err := tx.QueryRow(`SELECT id FROM sections WHERE slug = ?`, section.slug).Scan(&sectionID); err != nil {
			return err
		}

		var itemCount int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM items WHERE section_id = ?`, sectionID).Scan(&itemCount); err != nil {
			return err
		}
		if itemCount == 0 {
			item := section.placeholder
			if _, err := tx.Exec(`
				INSERT INTO items (section_id, title, subtitle, period, description, url, image_url, image_alt, tags, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
			`,
				sectionID,
				item.Title,
				item.Subtitle,
				item.Period,
				item.Description,
				item.URL,
				item.ImageURL,
				item.ImageAlt,
				joinTags(item.Tags),
			); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(`DELETE FROM sections WHERE slug = ?`, "dev"); err != nil {
		return err
	}

	return tx.Commit()
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

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM items WHERE section_id = ? AND title = ?`, sectionID, item.Title).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := conn.Exec(`
		INSERT INTO items (section_id, title, subtitle, period, description, url, image_url, image_alt, tags, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sectionID,
		item.Title,
		item.Subtitle,
		item.Period,
		item.Description,
		item.URL,
		item.ImageURL,
		item.ImageAlt,
		joinTags(item.Tags),
		sortOrder,
	)
	return err
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	value := tags[0]
	for _, tag := range tags[1:] {
		value += "," + tag
	}
	return value
}
