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

type Item struct {
	Title       string
	Subtitle    string
	Period      string
	Description string
	URL         string
	ImageURL    string
	ImageAlt    string
	Tags        []string
}

type Site struct {
	Profile  Profile
	Sections []Section
	Active   string
}
