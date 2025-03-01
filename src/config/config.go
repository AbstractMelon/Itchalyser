package config

// Config holds all configuration settings for the scraper
type Config struct {
	// Output configuration
	OutputFormat  string // json, jsonl, or markdown
	OutputDir     string // Where to store the data

	// Network configuration
	Workers       int    // Number of concurrent workers
	UserAgent     string // User agent string for HTTP requests
	RequestDelay  int    // Delay between requests in milliseconds (default: 1500)

	// Feature flags
	DownloadMedia bool // Whether to download media files
	DownloadGames bool // Whether to download game files
}

