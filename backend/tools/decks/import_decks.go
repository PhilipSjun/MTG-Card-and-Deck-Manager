package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbConnString = "postgres://postgres:postgres@localhost:5432/mtgcards"
	deckDir      = "./data/decks"
)

var sectionHeaders = map[string]string{
	"commander":  "commander",
	"mainboard":  "mainboard",
	"sideboard":  "sideboard",
	"maybeboard": "maybeboard",
}

type DeckEntry struct {
	CardName string
	Quantity int
	Section  string
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbConnString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	files, err := filepath.Glob(filepath.Join(deckDir, "*.txt"))
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fmt.Println("Importing deck:", file)
		if err := importDeck(ctx, db, file); err != nil {
			fmt.Println("Error importing deck:", err)
		}
	}
}

func importDeck(ctx context.Context, db *pgxpool.Pool, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	deckName := strings.TrimSuffix(filepath.Base(filePath), ".txt")
	deckID := uuid.New()
	sections := make([]DeckEntry, 0)

	scanner := bufio.NewScanner(f)
	currentSection := ""
	quantityPattern := regexp.MustCompile(`(?i)(\d+)x?\s+(.*)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		section := strings.ToLower(line)
		if mapped, ok := sectionHeaders[section]; ok {
			currentSection = mapped
			continue
		}

		matches := quantityPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		qty, _ := strconv.Atoi(matches[1])
		cardName := matches[2]
		sections = append(sections, DeckEntry{CardName: cardName, Quantity: qty, Section: currentSection})
	}

	// Check if deck already exists
	var existingDeckID string
	err = db.QueryRow(ctx, `SELECT id FROM decks WHERE name = $1 LIMIT 1`, deckName).Scan(&existingDeckID)
	if err == nil {
		// Deck exists, clear old data
		_, _ = db.Exec(ctx, `DELETE FROM missing_cards WHERE deck_id = $1`, existingDeckID)
		_, _ = db.Exec(ctx, `DELETE FROM deck_cards WHERE deck_id = $1`, existingDeckID)
		deckID = uuid.MustParse(existingDeckID)
	} else {
		// Insert new deck
		_, err = db.Exec(ctx, `INSERT INTO decks (id, name, created_at) VALUES ($1, $2, $3)`, deckID, deckName, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create deck: %w", err)
		}
	}

	for _, entry := range sections {
		var cardID string
		err := db.QueryRow(ctx, `SELECT id FROM cards WHERE lower(name) = lower($1) LIMIT 1`, entry.CardName).Scan(&cardID)
		if err != nil {
			fmt.Println("Card not found in database:", entry.CardName)
			continue
		}

		_, err = db.Exec(ctx, `
			INSERT INTO deck_cards (deck_id, card_id, quantity, board_type)
			VALUES ($1, $2, $3, $4)
		`, deckID, cardID, entry.Quantity, entry.Section)
		if err != nil {
			return fmt.Errorf("failed to insert deck card: %w", err)
		}

		var owned, inUse int
		db.QueryRow(ctx, `SELECT COALESCE(SUM(quantity), 0) FROM owned_cards WHERE card_id = $1`, cardID).Scan(&owned)
		db.QueryRow(ctx, `SELECT COALESCE(SUM(quantity), 0) FROM deck_cards WHERE card_id = $1 AND deck_id != $2`, cardID, deckID).Scan(&inUse)

		if owned == 0 {
			_, _ = db.Exec(ctx, `INSERT INTO missing_cards (deck_id, card_id, reason) VALUES ($1, $2, 'not_owned')`, deckID, cardID)
		} else if owned-inUse < entry.Quantity {
			_, _ = db.Exec(ctx, `INSERT INTO missing_cards (deck_id, card_id, reason) VALUES ($1, $2, 'in_use_elsewhere')`, deckID, cardID)
		}
	}

	return nil
}
