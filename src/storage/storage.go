package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Itchalyser/fetcher"
)

// Manager handles storage operations
type Manager struct {
	baseDir string
}

// NewManager creates a new storage manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
	}
}

// CreateDirectory creates a directory if it doesn't exist
func (m *Manager) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// SaveJamMetadata saves jam metadata to a file
func (m *Manager) SaveJamMetadata(jamID string, metadata *fetcher.JamMetadata) error {
	jamDir := filepath.Join(m.baseDir, "jams", jamID)
	if err := m.CreateDirectory(jamDir); err != nil {
		return err
	}
	
	metaPath := filepath.Join(jamDir, "meta.json")
	return m.saveJSONToFile(metaPath, metadata)
}

// SaveGameSubmission saves a game submission to a file
func (m *Manager) SaveGameSubmission(jamID, gameID string, game *fetcher.GameSubmission) error {
	gameDir := filepath.Join(m.baseDir, "jams", jamID, "submissions", gameID)
	if err := m.CreateDirectory(gameDir); err != nil {
		return err
	}
	
	// Create media directory
	mediaDir := filepath.Join(gameDir, "media")
	if err := m.CreateDirectory(mediaDir); err != nil {
		return err
	}
	
	gamePath := filepath.Join(gameDir, "game.json")
	return m.saveJSONToFile(gamePath, game)
}

// AppendToJSONL appends a JSON object to a JSON Lines file
func (m *Manager) AppendToJSONL(filePath string, obj interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := m.CreateDirectory(dir); err != nil {
		return err
	}
	
	// Convert object to JSON
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	
	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Append JSON line
	_, err = file.WriteString(string(jsonBytes) + "\n")
	return err
}

// GenerateMarkdownReport generates a markdown report for a jam
func (m *Manager) GenerateMarkdownReport(jamID string, metadata *fetcher.JamMetadata, games []*fetcher.GameSubmission) error {
	reportDir := filepath.Join(m.baseDir, "reports")
	if err := m.CreateDirectory(reportDir); err != nil {
		return err
	}
	
	reportPath := filepath.Join(reportDir, fmt.Sprintf("%s-report.md", jamID))
	
	// Open file for writing
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write markdown header
	file.WriteString(fmt.Sprintf("# %s\n\n", metadata.Title))
	
	// Write jam metadata
	file.WriteString("## Jam Details\n\n")
	file.WriteString(fmt.Sprintf("- **ID**: %s\n", metadata.ID))
	file.WriteString(fmt.Sprintf("- **Theme**: %s\n", metadata.Theme))
	
	// Write host information
	if len(metadata.Hosts) > 0 {
		file.WriteString("- **Hosts**: ")
		hosts := make([]string, 0, len(metadata.Hosts))
		for _, host := range metadata.Hosts {
			hosts = append(hosts, fmt.Sprintf("[%s](%s)", host.Name, host.URL))
		}
		file.WriteString(strings.Join(hosts, ", ") + "\n")
	}
	
	// Write dates
	file.WriteString(fmt.Sprintf("- **Start Date**: %s\n", metadata.StartDate))
	file.WriteString(fmt.Sprintf("- **End Date**: %s\n", metadata.EndDate))
	file.WriteString(fmt.Sprintf("- **Submission Date**: %s\n", metadata.SubmissionDate))
	
	// Write stats
	file.WriteString(fmt.Sprintf("- **Submissions**: %s\n", metadata.SubmissionCount))
	file.WriteString(fmt.Sprintf("- **Ratings**: %s\n", metadata.RatingCount))
	file.WriteString(fmt.Sprintf("- **Comments**: %s\n", metadata.CommentsCount))
	
	// Write submissions
	file.WriteString("\n## Game Submissions\n\n")
	
	for _, game := range games {
		file.WriteString(fmt.Sprintf("### %s\n\n", game.Title))
		
		file.WriteString(fmt.Sprintf("- **URL**: [Play on itch.io](%s)\n", game.URL))
		
		// Write authors
		if len(game.Authors) > 0 {
			file.WriteString("- **Authors**: ")
			authors := make([]string, 0, len(game.Authors))
			for _, author := range game.Authors {
				authors = append(authors, fmt.Sprintf("[%s](%s)", author.Name, author.URL))
			}
			file.WriteString(strings.Join(authors, ", ") + "\n")
		}
		
		file.WriteString(fmt.Sprintf("- **Platforms**: %s\n", strings.Join(game.Platforms, ", ")))
		file.WriteString(fmt.Sprintf("- **Created**: %s\n", game.CreatedAt))
		file.WriteString(fmt.Sprintf("- **Ratings**: %d\n", game.RatingCount))
		
		// Write description
		if game.Description != "" {
			file.WriteString("\n**Description**:\n\n")
			file.WriteString(game.Description + "\n\n")
		}
		
		// Write criteria responses
		if len(game.CriteriaResponses) > 0 {
			file.WriteString("**Criteria Responses**:\n\n")
			for question, answer := range game.CriteriaResponses {
				file.WriteString(fmt.Sprintf("- **%s**: %s\n", formatCriteriaKey(question), answer))
			}
			file.WriteString("\n")
		}
		
		// Write downloads
		if len(game.Downloads) > 0 {
			file.WriteString("**Downloads**:\n\n")
			for _, download := range game.Downloads {
				file.WriteString(fmt.Sprintf("- %s (%s) - For %s\n", 
					download.Filename, 
					download.Size, 
					strings.Join(download.Platforms, ", ")))
			}
			file.WriteString("\n")
		}
		
		file.WriteString("---\n\n")
	}
	
	return nil
}

// saveJSONToFile saves an object as JSON to a file
func (m *Manager) saveJSONToFile(path string, obj interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := m.CreateDirectory(dir); err != nil {
		return err
	}
	
	// Convert object to JSON
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file
	return os.WriteFile(path, jsonBytes, 0644)
}

// formatCriteriaKey formats a criteria key for better readability
func formatCriteriaKey(key string) string {
	// Replace underscores with spaces
	key = strings.ReplaceAll(key, "_", " ")
	
	// Capitalize first letter of each word
	words := strings.Split(key, " ")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	
	return strings.Join(words, " ")
}