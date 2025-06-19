package main

import (
	"log"

	"github.com/admin/mtg-card-manager/internal/analysis"
)

func main() {
	if err := analysis.EstimateBrackets(); err != nil {
		log.Fatalf("bracket_estimator failed: %v", err)
	}
}
