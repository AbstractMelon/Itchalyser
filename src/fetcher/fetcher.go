package fetcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// JamFetcher handles fetching data from itch.io
type JamFetcher struct {
	client     *http.Client
	userAgent  string
	requestDelay time.Duration
}

// NewFetcher creates a new JamFetcher with the given user agent and delay
func NewFetcher(userAgent string, delayMS int) *JamFetcher {
	if delayMS <= 0 {
		delayMS = 500 // Default delay of 0.5 seconds
	}
	
	return &JamFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent:    userAgent,
		requestDelay: time.Duration(delayMS) * time.Millisecond,
	}
}

// ExtractJamID extracts the jam ID from a jam URL
func ExtractJamID(jamURL string) (string, error) {
	// Handle URLs like https://itch.io/jam/brackeys-13
	re := regexp.MustCompile(`itch\.io/jam/([^/]+)`)
	matches := re.FindStringSubmatch(jamURL)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	// Try to extract from different URL format or from the page content
	doc, err := fetchHTMLDoc(jamURL)
	if err != nil {
		return "", err
	}

	// Look for randomizer link which contains the jam ID
	randomizerLink := doc.Find("a.randomizer_link").AttrOr("href", "")
	if randomizerLink != "" {
		re = regexp.MustCompile(`jam_id=(\d+)`)
		matches = re.FindStringSubmatch(randomizerLink)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", errors.New("could not extract jam ID from URL")
}

// FetchJamEntries fetches entries from the JSON endpoint
func (f *JamFetcher) FetchJamEntries(jamID string) (*JamEntriesResponse, error) {
	url := fmt.Sprintf("https://itch.io/jam/%s/entries.json", jamID)
	
	time.Sleep(f.requestDelay) // Respect rate limiting
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", f.userAgent)
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch jam entries, status code: %d", resp.StatusCode)
	}
	
	var entriesResponse JamEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&entriesResponse); err != nil {
		return nil, err
	}
	
	return &entriesResponse, nil
}

// FetchJamMetadata fetches metadata about the jam
func (f *JamFetcher) FetchJamMetadata(jamID string) (*JamMetadata, error) {
	url := fmt.Sprintf("https://itch.io/jam/%s", jamID)
	
	time.Sleep(f.requestDelay) // Respect rate limiting
	
	doc, err := f.fetchHTMLDoc(url)
	if err != nil {
		return nil, err
	}
	
	metadata := &JamMetadata{
		ID:    jamID,
		Title: strings.TrimSpace(doc.Find(".jam_title_header").Text()),
	}
	
	// Try to extract internal ID
	internalID, err := extractInternalIDFromPage(doc)
	if err != nil {
		return nil, err
	}
	if internalID != "" {
		metadata.InternalID = internalID
	}
	
	// Extract host information
	doc.Find(".jam_host_header a").Each(func(i int, s *goquery.Selection) {
		host := Host{
			Name: strings.TrimSpace(s.Text()),
			URL:  s.AttrOr("href", ""),
		}
		metadata.Hosts = append(metadata.Hosts, host)
	})
	
	// Extract stats
	doc.Find(".stat_box").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find(".stat_label").Text())
		value := strings.TrimSpace(s.Find(".stat_value").Text())
		
		switch strings.ToLower(label) {
		case "entries":
			metadata.SubmissionCount = value
		case "ratings":
			metadata.RatingCount = value
		case "comments":
			metadata.CommentsCount = value
		}
	})
	
	// Extract dates
	doc.Find(".jam_details_widget .line").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find(".label").Text())
		value := strings.TrimSpace(s.Find(".date_countdown").Text())
		
		switch {
		case strings.Contains(strings.ToLower(label), "start"):
			metadata.StartDate = value
		case strings.Contains(strings.ToLower(label), "end"):
			metadata.EndDate = value
		case strings.Contains(strings.ToLower(label), "submission"):
			metadata.SubmissionDate = value
		}
	})
	
	// Extract theme
	metadata.Theme = strings.TrimSpace(doc.Find(".jam_theme_display").Text())
	
	// Extract cover image
	coverImgSrc := doc.Find(".jam_cover").AttrOr("src", "")
	if coverImgSrc != "" {
		metadata.CoverImageURL = coverImgSrc
	}
	
	return metadata, nil
}

// extractInternalIDFromPage tries to extract the internal ID from the page content
func extractInternalIDFromPage(doc *goquery.Document) (string, error) {
	script := doc.Find("script").Text()
	re := regexp.MustCompile(`"id":(\d+)`)
	matches := re.FindStringSubmatch(script)
	if len(matches) >= 2 {
		return matches[1], nil
	}
	
	return "", errors.New("could not extract internal ID from page")
}

// FetchGameDetails fetches detailed information about a game submission
func (f *JamFetcher) FetchGameDetails(jamID, gameID string) (*GameSubmission, error) {
	url := fmt.Sprintf("https://itch.io/jam/%s/rate/%s", jamID, gameID)
	
	time.Sleep(f.requestDelay) // Respect rate limiting
	
	doc, err := f.fetchHTMLDoc(url)
	if err != nil {
		return nil, err
	}
	
	game := &GameSubmission{
		ID: gameID,
	}
	
	// Extract description
	game.Description = strings.TrimSpace(doc.Find(".formatted_description").Text())
	
	// Extract screenshots
	doc.Find("[data-screenshot_id]").Each(func(i int, s *goquery.Selection) {
		screenshotURL := s.AttrOr("data-screenshot_src", "")
		if screenshotURL != "" {
			game.Screenshots = append(game.Screenshots, screenshotURL)
		}
	})
	
	// Extract downloads
	doc.Find(".upload_list_widget .upload").Each(func(i int, s *goquery.Selection) {
		download := Download{
			Filename: strings.TrimSpace(s.Find(".upload_name").Text()),
			Size:     strings.TrimSpace(s.Find(".file_size").Text()),
		}
		
		// Extract platforms
		s.Find(".download_platforms .platform_tag").Each(func(j int, p *goquery.Selection) {
			platform := strings.TrimSpace(p.Text())
			download.Platforms = append(download.Platforms, platform)
		})
		
		// Extract upload date
		uploadDateText := strings.TrimSpace(s.Find(".upload_date").Text())
		download.UploadDate = uploadDateText
		
		game.Downloads = append(game.Downloads, download)
	})
	
	// Extract criteria responses
	doc.Find(".field_responses p").Each(func(i int, s *goquery.Selection) {
		questionText := strings.TrimSpace(s.Find("strong").Text())
		// Remove the question and any HTML tags for the answer
		s.Find("strong").Remove()
		answerText := strings.TrimSpace(s.Text())
		
		if questionText != "" && answerText != "" {
			if game.CriteriaResponses == nil {
				game.CriteriaResponses = make(map[string]string)
			}
			
			// Convert question to a key
			key := strings.ToLower(questionText)
			key = strings.Replace(key, "?", "", -1)
			key = strings.Replace(key, " ", "_", -1)
			
			game.CriteriaResponses[key] = answerText
		}
	})
	
	// Extract comments
	doc.Find(".community_post").Each(func(i int, s *goquery.Selection) {
		comment := Comment{
			Author:    strings.TrimSpace(s.Find(".post_author").Text()),
			Content:   strings.TrimSpace(s.Find(".post_body").Text()),
			Timestamp: strings.TrimSpace(s.Find(".post_date").Text()),
		}
		
		// Try to extract upvotes
		upvotesText := s.Find(".vote_button_count").Text()
		if upvotesText != "" {
			comment.Ratings = map[string]int{"upvotes": parseInt(upvotesText)}
		}
		
		game.Comments = append(game.Comments, comment)
	})
	
	return game, nil
}

// DownloadFile downloads a file from a URL to the specified path
func (f *JamFetcher) DownloadFile(url, destPath string) error {
	time.Sleep(f.requestDelay) // Respect rate limiting
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("User-Agent", f.userAgent)
	
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}
	
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = io.Copy(file, resp.Body)
	return err
}

// Helper to fetch HTML document
func (f *JamFetcher) fetchHTMLDoc(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", f.userAgent)
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// fmt.Printf("Fetching HTML document: %s", url)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page, status code: %d", resp.StatusCode)
	}
	
	return goquery.NewDocumentFromReader(resp.Body)
}

// Helper function for non-method contexts
func fetchHTMLDoc(url string) (*goquery.Document, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "ItchJamScraper/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page, status code: %d", resp.StatusCode)
	}
	
	return goquery.NewDocumentFromReader(resp.Body)
}

// Helper function to parse int from string, returns 0 if parsing fails
func parseInt(s string) int {
	// Remove non-numeric characters
	re := regexp.MustCompile(`\D+`)
	numStr := re.ReplaceAllString(s, "")
	
	var result int
	fmt.Sscanf(numStr, "%d", &result)
	return result
}

// IsAbsoluteURL checks if a URL is absolute
func IsAbsoluteURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

