package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/admin/mtg-card-manager/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CardPayload struct {
	Card     string `json:"card"`
	Quantity int    `json:"quantity"`
}

type BracketRequest struct {
	Main       []CardPayload `json:"main"`
	Commanders []CardPayload `json:"commanders"`
}

type BracketEstimation struct {
	DeckID                    uuid.UUID       `json:"deck_id"`
	BracketTag                string          `json:"bracketTag"`
	GameChangerCards          json.RawMessage `json:"gameChangerCards"`
	MassLandDenialCards       json.RawMessage `json:"massLandDenialCards"`
	MassLandDenialTemplates   json.RawMessage `json:"massLandDenialTemplates"`
	MassLandDenialCombos      json.RawMessage `json:"massLandDenialCombos"`
	ExtraTurnCards            json.RawMessage `json:"extraTurnCards"`
	ExtraTurnTemplates        json.RawMessage `json:"extraTurnTemplates"`
	ExtraTurnsCombos          json.RawMessage `json:"extraTurnsCombos"`
	TutorCards                json.RawMessage `json:"tutorCards"`
	TutorTemplates            json.RawMessage `json:"tutorTemplates"`
	LockCombos                json.RawMessage `json:"lockCombos"`
	SkipTurnsCombos           json.RawMessage `json:"skipTurnsCombos"`
	DefinitelyEarlyGameCombos json.RawMessage `json:"definitelyEarlyGameTwoCardCombos"`
	ArguablyEarlyGameCombos   json.RawMessage `json:"arguablyEarlyGameTwoCardCombos"`
	DefinitelyLateGameCombos  json.RawMessage `json:"definitelyLateGameTwoCardCombos"`
	BorderlineLateGameCombos  json.RawMessage `json:"borderlineLateGameTwoCardCombos"`
}

func EstimateBrackets() error {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("missing required DATABASE_URL environment variable")
	}

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(ctx, `
		SELECT d.id, 
    ARRAY_REMOVE(ARRAY_AGG(CASE WHEN board_type = 'commander' THEN c.name END), NULL) AS commanders,
    ARRAY_REMOVE(ARRAY_AGG(CASE WHEN board_type = 'mainboard' THEN c.name END), NULL) AS mainboard
FROM decks d
JOIN deck_cards dc ON dc.deck_id = d.id
JOIN cards c ON c.id = dc.card_id
GROUP BY d.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var deckID uuid.UUID
		var commanders, mainboard []string

		err := rows.Scan(&deckID, &commanders, &mainboard)
		if err != nil {
			log.Printf("Failed to scan deck: %v", err)
			continue
		}

		fmt.Printf("Processing deck: %s\n", deckID.String())

		payload := BracketRequest{
			Commanders: buildCardPayload(commanders),
			Main:       buildCardPayload(mainboard),
		}

		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Failed to marshal JSON: %v", err)
			continue
		}

		apiURL := "https://backend.commanderspellbook.com/estimate-bracket"

		resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("API call failed for deck %s: %v", deckID.String(), err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			log.Printf("API call failed for deck %s: status %d, body: %s", deckID.String(), resp.StatusCode, string(body))
			continue
		}

		var result BracketEstimation
		err = json.Unmarshal(body, &result)
		if err != nil {
			log.Printf("Unmarshal failed for deck %s: %v\nBody: %s", deckID.String(), err, string(body))
			continue
		}

		// Add deck ID for reference
		result.DeckID = deckID

		_, err = conn.Exec(ctx, `
			INSERT INTO bracket_estimation (
				deck_id, bracket_tag, game_changer_cards, mass_land_denial_cards,
				mass_land_denial_templates, mass_land_denial_combos, extra_turn_cards,
				extra_turn_templates, extra_turns_combos, tutor_cards, tutor_templates,
				lock_combos, skip_turns_combos, definitely_early_game_two_card_combos,
				arguably_early_game_two_card_combos, definitely_late_game_two_card_combos,
				borderline_late_game_two_card_combos
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16, $17
			)
			ON CONFLICT (deck_id) DO UPDATE SET
				bracket_tag = EXCLUDED.bracket_tag,
				game_changer_cards = EXCLUDED.game_changer_cards,
				mass_land_denial_cards = EXCLUDED.mass_land_denial_cards,
				mass_land_denial_templates = EXCLUDED.mass_land_denial_templates,
				mass_land_denial_combos = EXCLUDED.mass_land_denial_combos,
				extra_turn_cards = EXCLUDED.extra_turn_cards,
				extra_turn_templates = EXCLUDED.extra_turn_templates,
				extra_turns_combos = EXCLUDED.extra_turns_combos,
				tutor_cards = EXCLUDED.tutor_cards,
				tutor_templates = EXCLUDED.tutor_templates,
				lock_combos = EXCLUDED.lock_combos,
				skip_turns_combos = EXCLUDED.skip_turns_combos,
				definitely_early_game_two_card_combos = EXCLUDED.definitely_early_game_two_card_combos,
				arguably_early_game_two_card_combos = EXCLUDED.arguably_early_game_two_card_combos,
				definitely_late_game_two_card_combos = EXCLUDED.definitely_late_game_two_card_combos,
				borderline_late_game_two_card_combos = EXCLUDED.borderline_late_game_two_card_combos
		`,
			result.DeckID, result.BracketTag,
			result.GameChangerCards, result.MassLandDenialCards, result.MassLandDenialTemplates,
			result.MassLandDenialCombos, result.ExtraTurnCards, result.ExtraTurnTemplates,
			result.ExtraTurnsCombos, result.TutorCards, result.TutorTemplates, result.LockCombos,
			result.SkipTurnsCombos, result.DefinitelyEarlyGameCombos,
			result.ArguablyEarlyGameCombos, result.DefinitelyLateGameCombos,
			result.BorderlineLateGameCombos,
		)
		if err != nil {
			log.Printf("Failed to insert estimation for deck %s: %v", deckID.String(), err)
		} else {
			fmt.Printf("Finished deck: %s\n", deckID.String())
		}
	}
	return nil
}

func buildCardPayload(cards []string) []CardPayload {
	var payload []CardPayload
	for _, c := range cards {
		if c != "" {
			payload = append(payload, CardPayload{Card: c, Quantity: 1})
		}
	}
	return payload
}
