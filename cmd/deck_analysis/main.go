package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/analysis"
)

func main() {
	if err := analysis.AnalyzeDecks(); err != nil {
		log.Fatalf("deck_analysis failed: %v", err)
	}
}
