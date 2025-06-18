package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const dbConnString = "postgres://postgres:postgres@localhost:5432/mtgcards?sslmode=disable"

func main() {
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	dbCtx := context.Background()

	rows, err := db.QueryContext(dbCtx, `
		SELECT d.id
		FROM decks d
		LEFT JOIN deck_analysis a ON d.id = a.deck_id
		WHERE a.deck_id IS NULL
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var deckID string
		if err := rows.Scan(&deckID); err != nil {
			log.Println("Error scanning deck id:", err)
			continue
		}
		analyzeDeck(dbCtx, db, deckID)
	}
}

type cardInfo struct {
	CMC      float64
	Types    []string
	ManaCost string
	Oracle   string
	Name     string
	IsLand   bool
	IsBasic  bool
	Quantity int
}

func analyzeDeck(ctx context.Context, db *sql.DB, deckID string) {
	colorSymbols := map[string]int{}
	drawCount := 0
	singleTargetRemoval := 0
	massRemoval := 0
	counterSpellCount := 0
	rampCount := 0
	tokenCount := 0
	recursionCount := 0
	fmt.Println("Analyzing deck:", deckID)

	query := `
		SELECT c.name, c.cmc, c.type_line, c.mana_cost, c.oracle_text,
		       (POSITION('Land' IN c.type_line) > 0) AS is_land,
		       (POSITION('Basic' IN c.type_line) > 0) AS is_basic,
		       dc.quantity
		FROM deck_cards dc
		JOIN cards c ON c.id = dc.card_id
		WHERE dc.deck_id = $1 AND dc.board_type IN ('commander', 'mainboard')
	`

	rows, err := db.QueryContext(ctx, query, deckID)
	if err != nil {
		log.Println("Query failed:", err)
		return
	}
	defer rows.Close()

	var cards []cardInfo
	for rows.Next() {
		var card cardInfo
		var typeLine string
		if err := rows.Scan(&card.Name, &card.CMC, &typeLine, &card.ManaCost, &card.Oracle, &card.IsLand, &card.IsBasic, &card.Quantity); err != nil {
			log.Println("Scan error:", err)
			continue
		}
		card.Types = parseTypes(typeLine)
		cards = append(cards, card)
	}

	if len(cards) == 0 {
		log.Println("No cards found for deck:", deckID)
		return
	}

	var cmcSum float64
	manaCurve := map[string]int{}
	cardTypes := map[string]int{}
	landCount := 0
	basicLands := 0
	highestCMC := 0.0
	cardTotal := 0

	for _, c := range cards {
		cardTotal += c.Quantity
		if !c.IsLand {
			cmcSum += c.CMC * float64(c.Quantity)
			if c.CMC > highestCMC {
				highestCMC = c.CMC
			}

			var k string
			if c.CMC >= 6 {
				k = "6+"
			} else {
				k = fmt.Sprintf("%.0f", c.CMC)
			}
			manaCurve[k] += c.Quantity
		}

		for _, t := range c.Types {
			cardTypes[t] += c.Quantity
		}

		if c.IsLand {
			landCount += c.Quantity
			if c.IsBasic {
				basicLands += c.Quantity
			}
		}

		for _, symbol := range []string{"W", "U", "B", "R", "G"} {
			count := strings.Count(c.ManaCost, "{"+symbol+"}")
			colorSymbols[symbol] += count * c.Quantity
		}

		oracle := strings.ToLower(c.Oracle)
		if strings.Contains(oracle, "draw a card") {
			drawCount += c.Quantity
		}
		if strings.Contains(oracle, "each creature") || strings.Contains(oracle, "all creatures") || strings.Contains(oracle, "all permanents") {
			massRemoval += c.Quantity
		} else if strings.Contains(oracle, "destroy") || strings.Contains(oracle, "exile") || strings.Contains(oracle, "counter target") {
			singleTargetRemoval += c.Quantity
		}
		if strings.Contains(oracle, "counter target") {
			counterSpellCount += c.Quantity
		}
		if !c.IsLand && (strings.Contains(oracle, "add {") || strings.Contains(oracle, "search your library for a land") || strings.Contains(oracle, "create a treasure")) {
			rampCount += c.Quantity
		}
		if strings.Contains(oracle, "create a") && strings.Contains(oracle, "token") {
			tokenCount += c.Quantity
		}
		if strings.Contains(oracle, "return target") && (strings.Contains(oracle, "graveyard") || strings.Contains(oracle, "hand")) {
			recursionCount += c.Quantity
		}
	}

	avgCMC := cmcSum / float64(cardTotal)
	jsonColors, _ := json.Marshal(colorSymbols)
	jsonCurve, _ := json.Marshal(manaCurve)
	jsonTypes, _ := json.Marshal(cardTypes)

	_, err = db.ExecContext(ctx, `
		INSERT INTO deck_analysis (
			deck_id, average_mana_value, highest_mana_value, mana_curve,
			card_types, land_count, basic_land_count, nonbasic_land_count,
			color_symbols, draw_count, single_target_removal_count, mass_removal_count,
			ramp_count, counterspell_count, token_count, recursion_count, analyzed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		deckID, avgCMC, int(highestCMC), jsonCurve, jsonTypes,
		landCount, basicLands, landCount-basicLands, jsonColors,
		drawCount, singleTargetRemoval,
		massRemoval, rampCount, counterSpellCount, tokenCount, recursionCount, time.Now())

	if err != nil {
		log.Println("Insert failed:", err)
	} else {
		fmt.Println("Deck analysis saved.")
	}
}

func parseTypes(typeLine string) []string {
	types := strings.Split(typeLine, " — ")[0]
	parts := strings.Fields(types)
	return parts
}
