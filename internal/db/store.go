package db

import (
	"database/sql"
	"strings"
)

type Store struct {
	conn *sql.DB
}

func NewStore(conn *sql.DB) Store {
	return Store{conn: conn}
}

func (s Store) Site(active string) (Site, error) {
	profile, err := s.Profile()
	if err != nil {
		return Site{}, err
	}

	sections, err := s.Sections()
	if err != nil {
		return Site{}, err
	}

	if active == "" && len(sections) > 0 {
		// Default to the first ordered section so the home page always has content.
		active = sections[0].Slug
	}

	return Site{
		Profile:  profile,
		Sections: sections,
		Active:   active,
	}, nil
}

func (s Store) Profile() (Profile, error) {
	var profile Profile
	err := s.conn.QueryRow(`SELECT name, title, email, summary FROM profile WHERE id = 1`).Scan(
		&profile.Name,
		&profile.Title,
		&profile.Email,
		&profile.Summary,
	)
	return profile, err
}

func (s Store) Sections() ([]Section, error) {
	rows, err := s.conn.Query(`SELECT id, slug, title FROM sections ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var id int64
		var section Section
		if err := rows.Scan(&id, &section.Slug, &section.Title); err != nil {
			return nil, err
		}

		// Items are loaded per section because the UI renders section cards as a nested tree.
		items, err := s.items(id)
		if err != nil {
			return nil, err
		}
		section.Items = items
		sections = append(sections, section)
	}

	return sections, rows.Err()
}

func (s Store) Section(slug string) (Section, error) {
	var id int64
	var section Section
	err := s.conn.QueryRow(`SELECT id, slug, title FROM sections WHERE slug = ?`, slug).Scan(&id, &section.Slug, &section.Title)
	if err != nil {
		return Section{}, err
	}

	section.Items, err = s.items(id)
	if err != nil {
		return Section{}, err
	}

	return section, nil
}

func (s Store) items(sectionID int64) ([]Item, error) {
	rows, err := s.conn.Query(`
		SELECT title, subtitle, period, description, url, image_url, image_alt, tags
		FROM items
		WHERE section_id = ?
		ORDER BY sort_order, id
	`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var rawTags string
		if err := rows.Scan(&item.Title, &item.Subtitle, &item.Period, &item.Description, &item.URL, &item.ImageURL, &item.ImageAlt, &rawTags); err != nil {
			return nil, err
		}
		item.Tags = splitTags(rawTags)
		items = append(items, item)
	}

	return items, rows.Err()
}

func splitTags(raw string) []string {
	if raw == "" {
		return nil
	}

	// Tags are stored compactly in SQLite and expanded for template rendering.
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
