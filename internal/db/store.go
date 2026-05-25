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

func (s Store) AddItem(sectionSlug string, item Item) error {
	var sectionID int64
	if err := s.conn.QueryRow(`SELECT id FROM sections WHERE slug = ?`, sectionSlug).Scan(&sectionID); err != nil {
		return err
	}

	var sortOrder int
	if err := s.conn.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM items WHERE section_id = ?`, sectionID).Scan(&sortOrder); err != nil {
		return err
	}

	_, err := s.conn.Exec(`
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

func (s Store) UpdateItem(sectionSlug string, item Item) error {
	var sectionID int64
	if err := s.conn.QueryRow(`SELECT id FROM sections WHERE slug = ?`, sectionSlug).Scan(&sectionID); err != nil {
		return err
	}

	_, err := s.conn.Exec(`
		UPDATE items
		SET slug = ?, title = ?, subtitle = ?, period = ?, description = ?, url = ?, image_url = ?, image_alt = ?, problem = ?, built = ?, learned = ?, tech_stack = ?, tags = ?
		WHERE id = ? AND section_id = ?
	`,
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
		item.ID,
		sectionID,
	)
	return err
}

func (s Store) DeleteItem(id int64) error {
	_, err := s.conn.Exec(`DELETE FROM items WHERE id = ?`, id)
	return err
}

func (s Store) ItemByID(id int64) (Item, error) {
	return scanItem(s.conn.QueryRow(`
		SELECT id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags
		FROM items
		WHERE id = ?
	`, id))
}

func (s Store) Item(slug string) (Item, error) {
	return scanItem(s.conn.QueryRow(`
		SELECT id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags
		FROM items
		WHERE slug = ?
	`, slug))
}

func (s Store) items(sectionID int64) ([]Item, error) {
	rows, err := s.conn.Query(`
		SELECT id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags
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
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

type itemScanner interface {
	Scan(dest ...any) error
}

func scanItem(scanner itemScanner) (Item, error) {
	var item Item
	var rawSlug sql.NullString
	var rawProblem sql.NullString
	var rawBuilt sql.NullString
	var rawLearned sql.NullString
	var rawTechStack string
	var rawTags string
	if err := scanner.Scan(
		&item.ID,
		&rawSlug,
		&item.Title,
		&item.Subtitle,
		&item.Period,
		&item.Description,
		&item.URL,
		&item.ImageURL,
		&item.ImageAlt,
		&rawProblem,
		&rawBuilt,
		&rawLearned,
		&rawTechStack,
		&rawTags,
	); err != nil {
		return Item{}, err
	}

	item.Slug = rawSlug.String
	item.Problem = rawProblem.String
	item.Built = rawBuilt.String
	item.Learned = rawLearned.String
	item.TechStack = splitTags(rawTechStack)
	item.Tags = splitTags(rawTags)
	return item, nil
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

func joinValues(values []string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return strings.Join(clean, ",")
}
