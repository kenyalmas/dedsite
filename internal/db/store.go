package db

import (
	"database/sql"
	"strings"
)

const itemColumns = `
	id, slug, title, subtitle, period, description, url, image_url, image_alt, problem, built, learned, tech_stack, tags
`

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
	records, err := s.sectionRecords()
	if err != nil {
		return nil, err
	}

	sections := make([]Section, len(records))
	indexByID := make(map[int64]int, len(records))
	for i, record := range records {
		sections[i] = Section{
			Slug:  record.Slug,
			Title: record.Title,
		}
		indexByID[record.ID] = i
	}

	if len(sections) == 0 {
		return sections, nil
	}

	rows, err := s.conn.Query(`
		SELECT section_id, ` + itemColumns + `
		FROM items
		ORDER BY section_id, sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sectionID int64
		item, err := scanItemWithSection(rows, &sectionID)
		if err != nil {
			return nil, err
		}
		index, ok := indexByID[sectionID]
		if !ok {
			continue
		}
		sections[index].Items = append(sections[index].Items, item)
	}

	return sections, rows.Err()
}

func (s Store) Section(slug string) (Section, error) {
	record, err := s.SectionRef(slug)
	if err != nil {
		return Section{}, err
	}

	items, err := s.items(record.ID)
	if err != nil {
		return Section{}, err
	}

	return Section{
		Slug:  record.Slug,
		Title: record.Title,
		Items: items,
	}, nil
}

func (s Store) SectionRef(slug string) (SectionRef, error) {
	var section SectionRef
	err := s.conn.QueryRow(`SELECT id, slug, title FROM sections WHERE slug = ?`, slug).Scan(&section.ID, &section.Slug, &section.Title)
	if err != nil {
		return SectionRef{}, err
	}
	return section, nil
}

func (s Store) AddItem(sectionSlug string, item Item) error {
	sectionID, err := s.sectionIDBySlug(sectionSlug)
	if err != nil {
		return err
	}

	var sortOrder int
	if err := s.conn.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM items WHERE section_id = ?`, sectionID).Scan(&sortOrder); err != nil {
		return err
	}

	_, err = s.conn.Exec(`
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
		JoinList(item.TechStack),
		JoinList(item.Tags),
		sortOrder,
	)
	return err
}

func (s Store) UpdateItem(sectionSlug string, item Item) error {
	sectionID, err := s.sectionIDBySlug(sectionSlug)
	if err != nil {
		return err
	}

	_, err = s.conn.Exec(`
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
		JoinList(item.TechStack),
		JoinList(item.Tags),
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
		SELECT `+itemColumns+`
		FROM items
		WHERE id = ?
	`, id))
}

func (s Store) Item(slug string) (Item, error) {
	return scanItem(s.conn.QueryRow(`
		SELECT `+itemColumns+`
		FROM items
		WHERE slug = ?
	`, slug))
}

func (s Store) items(sectionID int64) ([]Item, error) {
	rows, err := s.conn.Query(`
		SELECT `+itemColumns+`
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

func (s Store) sectionRecords() ([]SectionRef, error) {
	rows, err := s.conn.Query(`SELECT id, slug, title FROM sections ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SectionRef
	for rows.Next() {
		var record SectionRef
		if err := rows.Scan(&record.ID, &record.Slug, &record.Title); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (s Store) sectionIDBySlug(slug string) (int64, error) {
	var sectionID int64
	err := s.conn.QueryRow(`SELECT id FROM sections WHERE slug = ?`, slug).Scan(&sectionID)
	return sectionID, err
}

type itemScanner interface {
	Scan(dest ...any) error
}

func scanItem(scanner itemScanner) (Item, error) {
	return scanItemWithSection(scanner, nil)
}

func scanItemWithSection(scanner itemScanner, sectionID *int64) (Item, error) {
	var item Item
	var destinations []any
	var rawSlug sql.NullString
	var rawProblem sql.NullString
	var rawBuilt sql.NullString
	var rawLearned sql.NullString
	var rawTechStack string
	var rawTags string
	if sectionID != nil {
		destinations = append(destinations, sectionID)
	}
	destinations = append(destinations,
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
	)
	if err := scanner.Scan(destinations...); err != nil {
		return Item{}, err
	}

	item.Slug = rawSlug.String
	item.Problem = rawProblem.String
	item.Built = rawBuilt.String
	item.Learned = rawLearned.String
	item.TechStack = ParseList(rawTechStack)
	item.Tags = ParseList(rawTags)
	return item, nil
}

func ParseList(raw string) []string {
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

func JoinList(values []string) string {
	return strings.Join(NormalizeList(values), ",")
}

func NormalizeList(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return clean
}
