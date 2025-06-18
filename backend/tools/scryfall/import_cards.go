package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	dumpDir = "./data/scryfall_dumps"
)

type Card struct {
	ID            string            `json:"id"`
	OracleID      string            `json:"oracle_id"`
	Name          string            `json:"name"`
	OracleText    string            `json:"oracle_text"`
	Layout        string            `json:"layout"`
	ManaCost      string            `json:"mana_cost"`
	CMC           float64           `json:"cmc"`
	TypeLine      string            `json:"type_line"`
	Power         string            `json:"power"`
	Toughness     string            `json:"toughness"`
	Loyalty       string            `json:"loyalty"`
	Defense       string            `json:"defense"`
	Colors        []string          `json:"colors"`
	ColorIdentity []string          `json:"color_identity"`
	Keywords      []string          `json:"keywords"`
	Set           string            `json:"set"`
	CollectorNum  string            `json:"collector_number"`
	Rarity        string            `json:"rarity"`
	Artist        string            `json:"artist"`
	ImageURIs     map[string]string `json:"image_uris"`
	Legalities    map[string]string `json:"legalities"`
}

func findLatestDump() (string, error) {
	files, err := filepath.Glob(filepath.Join(dumpDir, "scryfall_cards_*.json"))
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no dump files found")
	}

	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().After(fj.ModTime())
	})

	return files[0], nil
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(".env.local"); err != nil {
		fmt.Printf("Warning: .env.local file not found: %v\n", err)
	}

	// Get database connection parameters with defaults
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "mtgcards"
	}

	// Construct connection string
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	latestDump, err := findLatestDump()
	if err != nil {
		panic(err)
	}

	fmt.Println("Using latest dump:", latestDump)
	file, err := os.Open(latestDump)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Read array start
	if _, err := decoder.Token(); err != nil {
		panic(err)
	}

	count := 0
	skipped := 0
	for decoder.More() {
		var card Card
		if err := decoder.Decode(&card); err != nil {
			fmt.Printf("Error decoding card: %v\n", err)
			continue
		}

		// Skip cards with empty IDs
		if card.ID == "" {
			skipped++
			continue
		}

		// Handle empty oracle_id by using the card's ID
		if card.OracleID == "" {
			card.OracleID = card.ID
		}

		// Handle potential null values
		if card.ImageURIs == nil {
			card.ImageURIs = make(map[string]string)
		}
		if card.Legalities == nil {
			card.Legalities = make(map[string]string)
		}
		if card.Colors == nil {
			card.Colors = make([]string, 0)
		}
		if card.ColorIdentity == nil {
			card.ColorIdentity = make([]string, 0)
		}
		if card.Keywords == nil {
			card.Keywords = make([]string, 0)
		}

		// Trim whitespace from oracle_text
		card.OracleText = strings.TrimSpace(card.OracleText)

		_, err = db.Exec(ctx, `
			INSERT INTO cards (
				id, oracle_id, name, oracle_text, layout, mana_cost, cmc, type_line, power, toughness,
				loyalty, defense, colors, color_identity, keywords, set_code, collector_number,
				rarity, artist, image_uris, legalities, full_data, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16,
				$17, $18, $19, $20, $21, $22, $23
			)
			ON CONFLICT (id) DO UPDATE SET
				oracle_id = EXCLUDED.oracle_id,
				name = EXCLUDED.name,
				oracle_text = EXCLUDED.oracle_text,
				layout = EXCLUDED.layout,
				mana_cost = EXCLUDED.mana_cost,
				cmc = EXCLUDED.cmc,
				type_line = EXCLUDED.type_line,
				power = EXCLUDED.power,
				toughness = EXCLUDED.toughness,
				loyalty = EXCLUDED.loyalty,
				defense = EXCLUDED.defense,
				colors = EXCLUDED.colors,
				color_identity = EXCLUDED.color_identity,
				keywords = EXCLUDED.keywords,
				set_code = EXCLUDED.set_code,
				collector_number = EXCLUDED.collector_number,
				rarity = EXCLUDED.rarity,
				artist = EXCLUDED.artist,
				image_uris = EXCLUDED.image_uris,
				legalities = EXCLUDED.legalities,
				full_data = EXCLUDED.full_data,
				updated_at = NOW()
		`, card.ID, card.OracleID, card.Name, card.OracleText, card.Layout, card.ManaCost, card.CMC, card.TypeLine,
			card.Power, card.Toughness, card.Loyalty, card.Defense,
			card.Colors, card.ColorIdentity, card.Keywords, card.Set, card.CollectorNum,
			card.Rarity, card.Artist, card.ImageURIs, card.Legalities, card, time.Now())
		if err != nil {
			fmt.Printf("Error inserting card %s: %v\n", card.Name, err)
			continue
		}
		count++
		if count%1000 == 0 {
			fmt.Printf("Processed %d cards...\n", count)
		}
	}
	fmt.Printf("Import complete. Successfully imported %d cards. Skipped %d invalid cards.\n", count, skipped)
}
