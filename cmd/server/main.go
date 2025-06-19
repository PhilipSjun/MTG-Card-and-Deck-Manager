package main

import (
	"log"
	"mtgmanager/internal/api"
	"mtgmanager/internal/config"
	"mtgmanager/internal/db"
)

func main() {
	cfg := config.Load()
	database := db.Connect(cfg.DatabaseURL)
	router := api.NewRouter(database)
	log.Fatal(router.Run(cfg.ServerAddress))
}
