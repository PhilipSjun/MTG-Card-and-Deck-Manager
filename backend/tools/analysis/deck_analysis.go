package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"

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
		SELECT d.id, d.name
		FROM decks d
		LEFT JOIN deck_analysis a ON d.id = a.deck_id
		WHERE a.deck_id IS NULL
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var deckID, deckName string
		if err := rows.Scan(&deckID, &deckName); err != nil {
			log.Println("Error scanning deck:", err)
			continue
		}
		fmt.Printf("Analyzing deck: %s (%s)\n", deckName, deckID)
		analyzeDeck(dbCtx, db, deckID)
	}
}

func containsWord(text string, pattern string) bool {
	r := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(pattern) + `\b`)
	return r.MatchString(text)
}

func containsAnyWord(text string, words ...string) bool {
	for _, w := range words {
		if containsWord(text, w) {
			return true
		}
	}
	return false
}

func containsAnyPhrase(text string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func isDrawEffect(text string) bool {
	return containsAnyPhrase(text,
		"draw a card", "you may draw", "then draw", "draw two", "draw x", "investigate")
}

func isRampEffect(text string) bool {
	return containsAnyPhrase(text,
		"add {", "add one mana", "add two mana", "add three mana", "add an amount of mana",
		"search your library for a land", "create a treasure", "mana pool", "untap target land",
		"put a land card")
}

func isSingleTargetRemoval(text string) bool {
	return containsAnyPhrase(text,
		"destroy target", "exile target", "damage to target", "fight target creature",
		"choose one or both")
}

func isMassRemoval(text string) bool {
	return containsAnyPhrase(text,
		"each creature", "all creatures", "all permanents", "destroy all", "exile all",
		"sacrifice all", "each opponent sacrifices")
}

func isCounterspell(text string) bool {
	return containsAnyPhrase(text,
		"counter target", "unless its controller pays")
}

func isTokenGenerator(text string) bool {
	return containsAnyPhrase(text,
		"create a", "create a copy of", "token")
}

func isRecursionEffect(text string) bool {
	return containsAnyPhrase(text,
		"return target", "from your graveyard", "escape", "retrace", "unearth", "eternalize",
		"disturb", "embalm", "delve", "undying", "persist")
}

func countManaPips(manaCost string) (int, map[string]int) {
	symbolCounts := map[string]int{}
	total := 0
	tokens := regexp.MustCompile(`\{(.*?)\}`).FindAllStringSubmatch(manaCost, -1)
	for _, token := range tokens {
		contents := strings.ToUpper(token[1])
		parts := strings.Split(contents, "/")
		for _, part := range parts {
			if part != "" {
				symbolCounts[part]++
				total++
			}
		}
	}
	return total, symbolCounts
}

func analyzeDeck(ctx context.Context, db *sql.DB, deckID string) {
	drawCount := 0
	singleTargetRemoval := 0
	massRemoval := 0
	counterSpellCount := 0
	rampCount := 0
	tokenCount := 0
	recursionCount := 0

	totalCMC := 0.0
	totalNonLand := 0
	manaCurve := map[int]int{}
	colorPips := map[string]int{"W": 0, "U": 0, "B": 0, "R": 0, "G": 0, "C": 0}

	basicLandCount := 0
	nonBasicLandCount := 0
	totalLandCount := 0
	typeSet := map[string]bool{}

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

	for rows.Next() {
		var name, typeLine, manaCost, oracle string
		var cmc float64
		var isLand, isBasic bool
		var quantity int
		if err := rows.Scan(&name, &cmc, &typeLine, &manaCost, &oracle, &isLand, &isBasic, &quantity); err != nil {
			log.Println("Scan error:", err)
			continue
		}

		oracle = strings.ToLower(oracle)

		if isDrawEffect(oracle) {
			drawCount += quantity
		}
		if isRampEffect(oracle) && !isLand {
			rampCount += quantity
		}
		if isSingleTargetRemoval(oracle) {
			singleTargetRemoval += quantity
		}
		if isMassRemoval(oracle) {
			massRemoval += quantity
		}
		if isCounterspell(oracle) {
			counterSpellCount += quantity
		}
		if isTokenGenerator(oracle) {
			tokenCount += quantity
		}
		if isRecursionEffect(oracle) {
			recursionCount += quantity
		}

		if isLand {
			totalLandCount += quantity
			if isBasic {
				basicLandCount += quantity
			} else {
				nonBasicLandCount += quantity
			}
			continue
		}

		totalCMC += cmc * float64(quantity)
		totalNonLand += quantity
		manaCurve[int(cmc)] += quantity

		tokens := regexp.MustCompile(`\{(.*?)\}`).FindAllStringSubmatch(manaCost, -1)
		for _, token := range tokens {
			contents := strings.ToUpper(token[1])
			parts := strings.Split(contents, "/")
			for _, part := range parts {
				switch part {
				case "W", "U", "B", "R", "G", "C":
					colorPips[part] += quantity
				}
			}
		}

		for _, t := range strings.Split(typeLine, " ") {
			if len(t) > 0 && unicode.IsUpper(rune(t[0])) {
				typeSet[t] = true
			}
		}
	}

	avgMana := 0.0
	if totalNonLand > 0 {
		avgMana = totalCMC / float64(totalNonLand)
	}

	types := make([]string, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}

	typesJSON, _ := json.Marshal(types)
	manaCurveJSON, _ := json.Marshal(manaCurve)
	colorPipsJSON, _ := json.Marshal(colorPips)

	_, err = db.ExecContext(ctx, `
		INSERT INTO deck_analysis (
			deck_id, draw_count, ramp_count, single_target_removal_count, mass_removal_count,
			counterspell_count, token_count, recursion_count,
			average_mana_value, mana_curve, highest_mana_value,
			card_types, basic_land_count, nonbasic_land_count, land_count, color_symbols
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (deck_id) DO NOTHING
	`, deckID, drawCount, rampCount, singleTargetRemoval, massRemoval, counterSpellCount, tokenCount, recursionCount,
		avgMana, string(manaCurveJSON), highestMana(manaCurve), string(typesJSON), basicLandCount, nonBasicLandCount, totalLandCount, string(colorPipsJSON))
	if err != nil {
		log.Println("Insert failed:", err)
	}

	fmt.Printf("Draw: %d, Ramp: %d, Removal: %d/%d, Counterspells: %d, Tokens: %d, Recursion: %d\n",
		drawCount, rampCount, singleTargetRemoval, massRemoval, counterSpellCount, tokenCount, recursionCount)
}

func highestMana(curve map[int]int) int {
	high := 0
	for k := range curve {
		if k > high {
			high = k
		}
	}
	return high
}
