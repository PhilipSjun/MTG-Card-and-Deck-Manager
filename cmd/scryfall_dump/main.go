package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/scryfall"
)

func main() {
	if err := scryfall.DumpBulkCards(); err != nil {
		log.Fatalf("scryfall dump failed: %v", err)
	}
}
