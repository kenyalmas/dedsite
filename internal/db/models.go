package db

type Profile struct {
	Name    string
	Title   string
	Email   string
	Summary string
}

type Section struct {
	Slug  string
	Title string
	Items []Item
}

type SectionRef struct {
	ID    int64
	Slug  string
	Title string
}

type Item struct {
	ID          int64
	Slug        string
	Title       string
	Subtitle    string
	Period      string
	Description string
	URL         string
	ImageURL    string
	ImageAlt    string
	Problem     string
	Built       string
	Learned     string
	TechStack   []string
	Tags        []string
}

type Site struct {
	Profile  Profile
	Sections []Section
	Active   string
}
