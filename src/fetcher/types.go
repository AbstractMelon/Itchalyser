package fetcher

// JamEntriesResponse represents the JSON structure returned by itch.io's entries.json endpoint
type JamEntriesResponse struct {
	JamGames    []JamGame `json:"jam_games"`
	GeneratedOn float64   `json:"generated_on"`
}

// JamGame represents a game entry in the jam
type JamGame struct {
	CreatedAt    string        `json:"created_at"`
	ID           int           `json:"id"`
	Contributors []Contributor `json:"contributors,omitempty"`
	Coolness     int           `json:"coolness"`
	RatingCount  int           `json:"rating_count"`
	URL          string        `json:"url"`
	Game         Game          `json:"game"`
}

// Game represents a game in the jam
type Game struct {
	CoverColor string `json:"cover_color"`
	ID         int    `json:"id"`
	User       User   `json:"user"`
	Platforms  []string `json:"platforms"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	ShortText  string `json:"short_text,omitempty"`
	Cover      string `json:"cover"`
}

// User represents a user on itch.io
type User struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
	URL  string `json:"url"`
}

// Contributor represents a contributor to a game
type Contributor struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// JamMetadata represents metadata about a jam
type JamMetadata struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Hosts           []Host `json:"hosts"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	SubmissionDate  string `json:"submission_date"`
	Theme           string `json:"theme"`
	SubmissionCount string `json:"submission_count"`
	RatingCount     string `json:"rating_count"`
	CommentsCount   string `json:"comments_count"`
	CoverImageURL   string `json:"cover_image_url"`
	InternalID      string `json:"internal_id"`
}

// Host represents a jam host
type Host struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GameSubmission represents a game submission with detailed information
type GameSubmission struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	URL              string            `json:"url"`
	Description      string            `json:"description"`
	Authors          []User            `json:"authors"`
	Platforms        []string          `json:"platforms"`
	CreatedAt        string            `json:"created_at"`
	Coolness         int               `json:"coolness"`
	RatingCount      int               `json:"rating_count"`
	Cover            CoverImage        `json:"cover"`
	Screenshots      []string          `json:"screenshots"`
	Downloads        []Download        `json:"downloads"`
	Comments         []Comment         `json:"comments"`
	CriteriaResponses map[string]string `json:"criteria_responses"`
}

// CoverImage represents a game's cover image
type CoverImage struct {
	URL   string `json:"url"`
	Color string `json:"color"`
}

// Download represents a downloadable file for a game
type Download struct {
	Filename  string   `json:"filename"`
	Size      string   `json:"size"`
	Platforms []string `json:"platforms"`
	UploadDate string  `json:"upload_date"`
}

// Comment represents a comment on a game
type Comment struct {
	Author    string         `json:"author"`
	Content   string         `json:"content"`
	Timestamp string         `json:"timestamp"`
	Ratings   map[string]int `json:"ratings,omitempty"`
}