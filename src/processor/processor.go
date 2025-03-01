package processor

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"sync"

	"Itchalyser/config"
	"Itchalyser/fetcher"
	"Itchalyser/storage"
)

// Processor handles the processing of jams and games
type Processor struct {
	fetcher         *fetcher.JamFetcher
	storage         *storage.Manager
	config          config.Config
	gameCache       map[string]bool
	gameCacheMutex  sync.RWMutex
}

// NewProcessor creates a new Processor
func NewProcessor(storage *storage.Manager, cfg config.Config) *Processor {
	return &Processor{
		fetcher:        fetcher.NewFetcher(cfg.UserAgent, cfg.RequestDelay),
		storage:        storage,
		config:         cfg,
		gameCache:      make(map[string]bool),
		gameCacheMutex: sync.RWMutex{},
	}
}

// ProcessJam processes a single jam
func (p *Processor) ProcessJam(jamID string) error {
	log.Printf("Starting processing for jam: %s", jamID)
	
	// Create jam directory
	jamDir := filepath.Join(p.config.OutputDir, "jams", jamID)
	if err := p.storage.CreateDirectory(jamDir); err != nil {
		log.Printf("Error: Failed to create jam directory for %s: %v", jamID, err)
		return fmt.Errorf("failed to create jam directory: %w", err)
	}
	log.Printf("Jam directory created for: %s", jamID)

	// Fetch jam metadata
	metadata, err := p.fetcher.FetchJamMetadata(jamID)
	if err != nil {
		log.Printf("Error: Failed to fetch jam metadata for %s: %v", jamID, err)
		return fmt.Errorf("failed to fetch jam metadata: %w", err)
	}
	log.Printf("Fetched metadata for jam: %s, Title: %s", jamID, metadata.Title)

	// Save jam metadata
	if err := p.storage.SaveJamMetadata(jamID, metadata); err != nil {
		log.Printf("Error: Failed to save jam metadata for %s: %v", jamID, err)
		return fmt.Errorf("failed to save jam metadata: %w", err)
	}
	log.Printf("Saved metadata for jam: %s", jamID)

	// Download jam cover image if available
	if metadata.CoverImageURL != "" && p.config.DownloadMedia {
		log.Printf("Downloading cover image for jam: %s from URL: %s", jamID, metadata.CoverImageURL)
		coverPath := filepath.Join(jamDir, "cover"+filepath.Ext(metadata.CoverImageURL))
		if err := p.fetcher.DownloadFile(metadata.CoverImageURL, coverPath); err != nil {
			log.Printf("Warning: Failed to download jam cover image for %s: %v", jamID, err)
		} else {
			log.Printf("Downloaded cover image for jam: %s", jamID)
		}
	}

	// Use InternalID to fetch jam entries
	entriesResponse, err := p.fetcher.FetchJamEntries(metadata.InternalID)
	if err != nil {
		log.Printf("Error: Failed to fetch jam entries from URL https://itch.io/jam/%s/entries.json for %s: %v", metadata.InternalID, jamID, err)
		return fmt.Errorf("failed to fetch jam entries: %w", err)
	}
	log.Printf("Fetched %d entries for jam: %s", len(entriesResponse.JamGames), jamID)

	// Process each game
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.config.Workers)

	log.Printf("Beginning processing of games for jam: %s", jamID)

	for _, jamGame := range entriesResponse.JamGames {
		gameID := strconv.Itoa(jamGame.Game.ID)

		// Check if we've already processed this game
		p.gameCacheMutex.RLock()
		alreadyProcessed := p.gameCache[gameID]
		p.gameCacheMutex.RUnlock()

		if alreadyProcessed {
			log.Printf("Skipping game %s as it has already been processed", gameID)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(jg fetcher.JamGame) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			gameID := strconv.Itoa(jg.Game.ID)
			log.Printf("Starting processing for game: %s", gameID)

			// Mark this game as processed
			p.gameCacheMutex.Lock()
			p.gameCache[gameID] = true
			p.gameCacheMutex.Unlock()


			// Create basic game submission from jam game
			submission := &fetcher.GameSubmission{
				ID:          gameID,
				Title:       jg.Game.Title,
				URL:         jg.Game.URL,
				Platforms:   jg.Game.Platforms,
				CreatedAt:   jg.CreatedAt,
				Coolness:    jg.Coolness,
				RatingCount: jg.RatingCount,
				Cover: fetcher.CoverImage{
					URL:   jg.Game.Cover,
					Color: jg.Game.CoverColor,
				},
			}
			log.Printf("Created submission for game: %s - %s", gameID, submission.Title)

			// Add authors
			submission.Authors = append(submission.Authors, jg.Game.User)

			// Add contributors if available
			for _, contributor := range jg.Contributors {
				submission.Authors = append(submission.Authors, fetcher.User{
					Name: contributor.Name,
					URL:  contributor.URL,
				})
			}
			log.Printf("Added authors and contributors for game: %s", gameID)

			// Try to get more details
			gameDetails, err := p.fetcher.FetchGameDetails(jamID, gameID)
			if err != nil {
				log.Printf("Warning: Failed to fetch details for game %s: %v", gameID, err)
			} else {
				// Update submission with additional details
				submission.Description = gameDetails.Description
				submission.Screenshots = gameDetails.Screenshots
				submission.Downloads = gameDetails.Downloads
				submission.Comments = gameDetails.Comments
				submission.CriteriaResponses = gameDetails.CriteriaResponses
				log.Printf("Fetched additional details for game: %s", gameID)
			}

			// Save game submission
			if err := p.storage.SaveGameSubmission(jamID, gameID, submission); err != nil {
				log.Printf("Warning: Failed to save game submission %s: %v", gameID, err)
				return
			}
			log.Printf("Saved game submission for game: %s", gameID)

			// Download media if configured
			if p.config.DownloadMedia {
				log.Printf("Downloading media for game: %s", gameID)
				p.downloadGameMedia(jamID, gameID, submission)
			}

			// Download game files if configured
			if p.config.DownloadGames && len(submission.Downloads) > 0 {
				log.Printf("Downloading game files for game: %s", gameID)
				p.downloadGameFiles(jamID, gameID, submission)
			}

			log.Printf("Finished processing game: %s - %s", gameID, submission.Title)
		}(jamGame)
	}

	wg.Wait()
	log.Printf("Finished processing all games for jam: %s", jamID)

	return nil
}

// downloadGameMedia downloads media files for a game
func (p *Processor) downloadGameMedia(jamID, gameID string, game *fetcher.GameSubmission) {
	gameMediaDir := filepath.Join(p.config.OutputDir, "jams", jamID, "submissions", gameID, "media")
	
	// Create media directory
	if err := p.storage.CreateDirectory(gameMediaDir); err != nil {
		log.Printf("Warning: Failed to create media directory for game %s: %v", gameID, err)
		return
	}
	
	// Download cover image
	if game.Cover.URL != "" {
		coverPath := filepath.Join(gameMediaDir, "cover"+filepath.Ext(game.Cover.URL))
		if err := p.fetcher.DownloadFile(game.Cover.URL, coverPath); err != nil {
			log.Printf("Warning: Failed to download cover for game %s: %v", gameID, err)
		}
	}
	
	// Download screenshots
	for i, screenshot := range game.Screenshots {
		screenshotPath := filepath.Join(gameMediaDir, fmt.Sprintf("screenshot%d%s", i+1, filepath.Ext(screenshot)))
		if err := p.fetcher.DownloadFile(screenshot, screenshotPath); err != nil {
			log.Printf("Warning: Failed to download screenshot %d for game %s: %v", i+1, gameID, err)
		}
	}
}

// downloadGameFiles downloads game files
func (p *Processor) downloadGameFiles(jamID, gameID string, game *fetcher.GameSubmission) {
	gameFilesDir := filepath.Join(p.config.OutputDir, "jams", jamID, "submissions", gameID, "files")
	
	// Create game files directory
	if err := p.storage.CreateDirectory(gameFilesDir); err != nil {
		log.Printf("Warning: Failed to create game files directory for game %s: %v", gameID, err)
		return
	}
	
	for _, download := range game.Downloads {
		// Extract download URL from game page
		// Note: This would require additional scraping logic as download URLs
		// are typically not directly accessible without login
		log.Printf("Note: Game file downloading requires additional authentication and is not fully implemented: %s", download.Filename)
	}
}




