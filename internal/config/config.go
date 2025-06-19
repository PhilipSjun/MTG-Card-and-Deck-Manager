package config

import (
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ServerAddress string
}

var loadOnce sync.Once

func Load() Config {
	loadOnce.Do(func() {
		_ = godotenv.Load(".env.local")
	})
	return Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		ServerAddress: os.Getenv("SERVER_ADDRESS"),
	}
}
