package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/scryfall"
)

func main() {
	if err := scryfall.ImportCards(); err != nil {
		log.Fatalf("import_cards failed: %v", err)
	}
}
