package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() (string, error) {
	// Get the directory of the current file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	// Navigate up three levels from backend/tools/dbinit to project root
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Abs(projectRoot)
}

// checkTableExists checks if a table exists in the database
func checkTableExists(ctx context.Context, conn *pgx.Conn, tableName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`, tableName).Scan(&exists)
	return exists, err
}

// dropTables drops all the tables in our schema
func dropTables(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		DROP TABLE IF EXISTS missing_cards CASCADE;
		DROP TABLE IF EXISTS deck_cards CASCADE;
		DROP TABLE IF EXISTS decks CASCADE;
		DROP TABLE IF EXISTS owned_cards CASCADE;
		DROP TABLE IF EXISTS cards CASCADE;
	`)
	return err
}

func main() {
	// Parse command line flags
	forceReset := flag.Bool("force", false, "Force reset: drop existing tables before creating new ones")
	flag.Parse()

	// Get project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatalf("Failed to determine project root: %v", err)
	}

	// Load environment variables from .env.local file
	envPath := filepath.Join(projectRoot, ".env.local")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env.local file not found at %s: %v", envPath, err)
	}

	// Get database connection parameters
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Validate required environment variables
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Missing required database environment variables. Please ensure DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, and DB_NAME are set")
	}

	// Construct connection string
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	// Connect to the database
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Check if tables exist
	exists, err := checkTableExists(ctx, conn, "cards")
	if err != nil {
		log.Fatalf("Error checking if tables exist: %v", err)
	}

	if exists {
		if !*forceReset {
			log.Fatal("Tables already exist. Use --force flag to drop and recreate them")
		}
		log.Println("Dropping existing tables...")
		if err := dropTables(ctx, conn); err != nil {
			log.Fatalf("Error dropping tables: %v", err)
		}
	}

	// Read the schema file
	schemaPath := filepath.Join(projectRoot, "app", "drizzle", "0000_initial.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Unable to read schema file: %v", err)
	}

	// Execute the schema
	_, err = conn.Exec(ctx, string(schema))
	if err != nil {
		log.Fatalf("Unable to execute schema: %v", err)
	}

	fmt.Println("Database schema initialized successfully!")
}
