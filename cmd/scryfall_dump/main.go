package main

import (
	"log"
	"mtgmanager/internal/scryfall"
)

func main() {
	if err := scryfall.DumpBulkCards(); err != nil {
		log.Fatalf("scryfall dump failed: %v", err)
	}
}
