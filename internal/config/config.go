package config

import (
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ServerAddress string
	SchemaPath    string
}

var loadOnce sync.Once

func Load() Config {
	loadOnce.Do(func() {
		_ = godotenv.Load(".env.local")
	})
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/mtgcards?sslmode=disable"
	}
	serverAddr := os.Getenv("SERVER_ADDRESS")
	if serverAddr == "" {
		serverAddr = ":8080"
	}
	schemaPath := os.Getenv("SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = "./app/drizzle/0000_initial.sql"
	}
	return Config{
		DatabaseURL:   dbURL,
		ServerAddress: serverAddr,
		SchemaPath:    schemaPath,
	}
}
