package main

import (
	"flag"
	"log"

	"github.com/admin/mtg-card-manager/internal/dbinit"
)

func main() {
	forceReset := flag.Bool("force", false, "Force reset: drop existing tables before creating new ones")
	flag.Parse()

	if err := dbinit.Run(*forceReset); err != nil {
		log.Fatalf("dbinit failed: %v", err)
	}
}
