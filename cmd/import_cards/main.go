package main

import (
	"log"
	"mtgmanager/internal/scryfall"
)

func main() {
	if err := scryfall.ImportCards(); err != nil {
		log.Fatalf("import_cards failed: %v", err)
	}
}
