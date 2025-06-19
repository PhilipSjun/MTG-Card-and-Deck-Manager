package decks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/admin/mtg-card-manager/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const openAIAPIURL = "https://api.openai.com/v1/chat/completions"

var openAIModels = []string{"gpt-4o-mini", "gpt-3.5-turbo"}

func DescribeDecks() error {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("missing required DATABASE_URL environment variable")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(ctx, `SELECT id, name, commander_name FROM decks WHERE description IS NULL OR description = ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var deckID, deckName, commanderName string
		if err := rows.Scan(&deckID, &deckName, &commanderName); err != nil {
			fmt.Println("Failed to scan deck:", err)
			continue
		}

		cardNames := []string{}
		cards, err := db.Query(ctx, `
			SELECT c.name
			FROM deck_cards dc
			JOIN cards c ON c.id = dc.card_id
			WHERE dc.deck_id = $1 AND dc.board_type IN ('commander', 'mainboard')
		`, deckID)
		if err != nil {
			fmt.Println("Failed to query cards for deck:", deckName, err)
			continue
		}

		for cards.Next() {
			var name string
			if err := cards.Scan(&name); err != nil {
				fmt.Println("Failed to scan card name:", err)
				continue
			}
			cardNames = append(cardNames, name)
		}
		cards.Close()

		prompt := fmt.Sprintf("Create a short (max 3 sentences) description of the play style of the following MTG commander deck:\nCommander: %s\nDeck List: %s", commanderName, strings.Join(cardNames, ", "))
		description, modelUsed, err := callOpenAI(prompt)
		if err != nil {
			fmt.Println("OpenAI API error for deck:", deckName, err)
			continue
		}

		_, err = db.Exec(ctx, `UPDATE decks SET description = $1, description_gpt_model = $2 WHERE id = $3`, description, modelUsed, deckID)
		fmt.Printf("Updated description for deck '%s' using model: %s\n%s\n", deckName, modelUsed, description)
		if err != nil {
			fmt.Println("Failed to update deck description:", err)
		}
	}
	return nil
}

func callOpenAI(prompt string) (string, string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	for _, model := range openAIModels {
		body := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": "You are a helpful assistant who summarizes Magic: The Gathering decks."},
				{"role": "user", "content": prompt},
			},
			"temperature": 0.7,
		}

		encoded, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", openAIAPIURL, bytes.NewBuffer(encoded))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Model %s failed, retrying next if available...", model)
			time.Sleep(2 * time.Second)
			continue // try next model
		}
		defer resp.Body.Close()

		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			continue
		}

		if len(result.Choices) == 0 {
			continue
		}

		return strings.TrimSpace(result.Choices[0].Message.Content), model, nil
	}

	return "", "", fmt.Errorf("all model attempts failed")
}
