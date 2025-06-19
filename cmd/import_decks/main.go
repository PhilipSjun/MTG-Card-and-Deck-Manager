package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/decks"
)

func main() {
	if err := decks.ImportDecks(); err != nil {
		log.Fatalf("import_decks failed: %v", err)
	}
}
