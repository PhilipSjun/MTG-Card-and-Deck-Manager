package scryfall

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
	bulkMetadataURL = "https://api.scryfall.com/bulk-data"
	dumpDir         = "./data/scryfall_dumps"
	retentionCount  = 5
)

type bulkEntry struct {
	Type        string `json:"type"`
	DownloadURI string `json:"download_uri"`
}

type bulkData struct {
	Data []bulkEntry `json:"data"`
}

func DumpBulkCards() error {
	if err := os.MkdirAll(dumpDir, 0755); err != nil {
		return err
	}

	resp, err := http.Get(bulkMetadataURL)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	var meta bulkData
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	var downloadURL string
	for _, entry := range meta.Data {
		if entry.Type == "default_cards" {
			downloadURL = entry.DownloadURI
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("could not find 'default_cards' entry")
	}

	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("scryfall_cards_%s.json", timestamp)
	outPath := filepath.Join(dumpDir, filename)

	fmt.Printf("Downloading to %s...\n", outPath)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download JSON: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	fmt.Println("Download complete.")

	files, err := filepath.Glob(filepath.Join(dumpDir, "scryfall_cards_*.json"))
	if err != nil {
		return err
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

	return nil
}
