package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/decks"
)

func main() {
	if err := decks.ImportCombos(); err != nil {
		log.Fatalf("import_combos failed: %v", err)
	}
}
