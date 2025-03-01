package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"Itchalyser/config"
	"Itchalyser/fetcher"
	"Itchalyser/processor"
	"Itchalyser/storage"
)

func main() {
	// Default user agent
	DefaultUserAgent := "Itchalyser/1.0 (https://github.com/Abstractmelon/Itchalyser)"

	// Parse command line flags
	jamURLs := flag.String("jam", "", "Comma-separated list of jam URLs")
	outputFormat := flag.String("output", "json", "Output format (json, jsonl, markdown)")
	outputDir := flag.String("dir", "../data", "Directory to store output")
	workers := flag.Int("workers", 2, "Number of concurrent workers")
	userAgent := flag.String("user-agent", DefaultUserAgent, "User agent string for HTTP requests")
	requestDelay := flag.Int("delay", 1500, "Delay between requests in milliseconds (default: 1500)")
	downloadMedia := flag.Bool("media", true, "Download media files")
	downloadGames := flag.Bool("games", false, "Download game files")
	flag.Parse()

	if *jamURLs == "" {
		log.Fatal("Please provide at least one jam URL using the -jam flag")
	}

	// Initialize configuration
	cfg := config.Config{
		OutputFormat:  *outputFormat,
		OutputDir:     *outputDir,
		Workers:       *workers,
		UserAgent:     *userAgent,
		RequestDelay:  *requestDelay,
		DownloadMedia: *downloadMedia,
		DownloadGames: *downloadGames,
	}

	// Create storage manager
	store := storage.NewManager(cfg.OutputDir)

	// Initialize jam processor
	proc := processor.NewProcessor(store, cfg)

	// Process each jam URL
	urls := strings.Split(*jamURLs, ",")
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Workers)

	for _, url := range urls {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		
		go func(jamURL string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore
			
			jamURL = strings.TrimSpace(jamURL)
			
			// Extract jam ID from URL
			jamID, err := fetcher.ExtractJamID(jamURL)
			if err != nil {
				log.Printf("Error extracting jam ID from %s: %v", jamURL, err)
				return
			}
			
			fmt.Printf("Processing jam: %s (ID: %s)\n", jamURL, jamID)
			
			// Process jam
			if err := proc.ProcessJam(jamID); err != nil {
				log.Printf("Error processing jam %s: %v", jamID, err)
			}
		}(url)
	}

	wg.Wait()
	fmt.Println("All jams processed successfully!")
}
