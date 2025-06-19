package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/decks"
)

func main() {
	if err := decks.DescribeDecks(); err != nil {
		log.Fatalf("deck_describer failed: %v", err)
	}
}
