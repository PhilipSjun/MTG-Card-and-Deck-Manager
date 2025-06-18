package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	metadataURL    = "https://api.scryfall.com/bulk-data"
	downloadDir    = "./data/scryfall_dumps"
	retentionCount = 5
)

type BulkEntry struct {
	Type        string `json:"type"`
	DownloadURI string `json:"download_uri"`
}

type BulkData struct {
	Data []BulkEntry `json:"data"`
}

func main() {
	err := os.MkdirAll(downloadDir, 0755)
	if err != nil {
		panic(err)
	}

	// Step 1: Fetch metadata
	resp, err := http.Get(metadataURL)
	if err != nil {
		panic(fmt.Errorf("failed to fetch metadata: %w", err))
	}
	defer resp.Body.Close()

	var bulkData BulkData
	if err := json.NewDecoder(resp.Body).Decode(&bulkData); err != nil {
		panic(fmt.Errorf("failed to decode metadata: %w", err))
	}

	var downloadURL string
	for _, entry := range bulkData.Data {
		if entry.Type == "default_cards" {
			downloadURL = entry.DownloadURI
			break
		}
	}
	if downloadURL == "" {
		panic("could not find 'default_cards' entry")
	}

	// Step 2: Download the JSON
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("scryfall_cards_%s.json", timestamp)
	outPath := filepath.Join(downloadDir, filename)

	fmt.Printf("Downloading to %s...\n", outPath)
	out, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	resp, err = http.Get(downloadURL)
	if err != nil {
		panic(fmt.Errorf("failed to download JSON: %w", err))
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(fmt.Errorf("failed to write file: %w", err))
	}
	fmt.Println("Download complete.")

	// Step 3: Rotate old files
	files, err := filepath.Glob(filepath.Join(downloadDir, "scryfall_cards_*.json"))
	if err != nil {
		panic(err)
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.Compare(files[j], files[i]) < 0 // newer first
	})

	if len(files) > retentionCount {
		for _, f := range files[retentionCount:] {
			fmt.Println("Deleting old backup:", f)
			_ = os.Remove(f)
		}
	}
}
