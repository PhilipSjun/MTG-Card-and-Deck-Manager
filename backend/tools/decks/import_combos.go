package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Deck struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

type Card struct {
	Name      string
	BoardType string
}

type Combo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
	Produces    []struct {
		Feature struct {
			Name string `json:"name"`
		} `json:"feature"`
	} `json:"produces"`
	Requires []struct {
		Template struct {
			Name string `json:"name"`
		} `json:"template"`
	} `json:"requires"`
	Uses []struct {
		Card struct {
			Name string `json:"name"`
		} `json:"card"`
	} `json:"uses"`
	NotablePrerequisites string `json:"notablePrerequisites"`
	BracketTag           string `json:"bracketTag"`
}

type ComboBuckets struct {
	Included                                          []Combo `json:"included"`
	IncludedByChangingCommanders                      []Combo `json:"includedByChangingCommanders"`
	AlmostIncluded                                    []Combo `json:"almostIncluded"`
	AlmostIncludedByAddingColors                      []Combo `json:"almostIncludedByAddingColors"`
	AlmostIncludedByChangingCommanders                []Combo `json:"almostIncludedByChangingCommanders"`
	AlmostIncludedByAddingColorsAndChangingCommanders []Combo `json:"almostIncludedByAddingColorsAndChangingCommanders"`
}

type SpellbookResponse struct {
	Count    int          `json:"count"`
	Next     *string      `json:"next"`
	Previous *string      `json:"previous"`
	Results  ComboBuckets `json:"results"`
}

func main() {
	err := godotenv.Load(".env.local")
	if err != nil {
		fmt.Println("Warning: .env.local file not found or could not be loaded")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := os.Getenv("DB_HOST")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	deckRows, err := pool.Query(ctx, `
		SELECT id, name, created_at FROM decks
		WHERE id NOT IN (
			SELECT deck_id FROM deck_combos
		)
	`)
	if err != nil {
		panic(err)
	}
	defer deckRows.Close()

	for deckRows.Next() {
		var deck Deck
		if err := deckRows.Scan(&deck.ID, &deck.Name, &deck.CreatedAt); err != nil {
			fmt.Printf("Failed to scan deck: %v\n", err)
			continue
		}

		fmt.Printf("Fetching combos for deck: %s %s\n", deck.Name, deck.ID)

		cardRows, err := pool.Query(ctx, `SELECT c.name, dc.board_type FROM deck_cards dc JOIN cards c ON c.id = dc.card_id WHERE dc.deck_id = $1`, deck.ID)
		if err != nil {
			fmt.Printf("Failed to fetch cards for deck %s: %v\n", deck.ID, err)
			continue
		}

		var commanders []map[string]interface{}
		var mainboard []map[string]interface{}

		for cardRows.Next() {
			var card Card
			if err := cardRows.Scan(&card.Name, &card.BoardType); err != nil {
				continue
			}
			entry := map[string]interface{}{"card": card.Name, "quantity": 1}
			if card.BoardType == "commander" {
				commanders = append(commanders, entry)
			} else {
				mainboard = append(mainboard, entry)
			}
		}
		cardRows.Close()

		payload := map[string]interface{}{
			"commanders": commanders,
			"main":       mainboard,
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post("https://backend.commanderspellbook.com/api/v1/find-my-combos/", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("HTTP request failed for deck %s: %v\n", deck.ID, err)
			continue
		}

		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var spellResp SpellbookResponse
		if err := json.Unmarshal(responseBody, &spellResp); err != nil {
			fmt.Printf("JSON unmarshal failed for deck %s: %v\n", deck.ID, err)
			continue
		}

		insertCombos := func(combos []Combo, bucket string) {
			for _, combo := range combos {
				var cards []string
				for _, use := range combo.Uses {
					cards = append(cards, use.Card.Name)
				}

				var produces []string
				for _, p := range combo.Produces {
					produces = append(produces, p.Feature.Name)
				}

				var requires []string
				for _, r := range combo.Requires {
					requires = append(requires, r.Template.Name)
				}

				_, err := pool.Exec(ctx, `
					INSERT INTO deck_combos (deck_id, combo_id, cards, description, prerequisites, requires, produces, inclusion_bucket)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
					deck.ID, combo.ID, cards, combo.Description, combo.NotablePrerequisites, requires, produces, bucket)
				if err != nil {
					fmt.Printf("Failed to insert combo for deck %s: %v\n", deck.ID, err)
				}
			}
		}

		insertCombos(spellResp.Results.Included, "included")
		insertCombos(spellResp.Results.IncludedByChangingCommanders, "includedByChangingCommanders")
		insertCombos(spellResp.Results.AlmostIncluded, "almostIncluded")
		insertCombos(spellResp.Results.AlmostIncludedByAddingColors, "almostIncludedByAddingColors")
		insertCombos(spellResp.Results.AlmostIncludedByChangingCommanders, "almostIncludedByChangingCommanders")
		insertCombos(spellResp.Results.AlmostIncludedByAddingColorsAndChangingCommanders, "almostIncludedByAddingColorsAndChangingCommanders")

		fmt.Printf("Finished deck: %s\n", deck.Name)
	}
}
