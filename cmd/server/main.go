package main

import (
	"log"
	"net/http"

	"github.com/admin/mtg-card-manager/internal/api"
	"github.com/admin/mtg-card-manager/internal/config"
	"github.com/admin/mtg-card-manager/internal/db"
)

func main() {
	cfg := config.Load()
	database := db.Connect(cfg.DatabaseURL)
	router := api.NewRouter(database)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, router))
}
