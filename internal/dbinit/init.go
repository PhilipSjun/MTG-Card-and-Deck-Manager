package dbinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"mtgmanager/internal/config"

	"github.com/jackc/pgx/v5"
)

func getProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Abs(projectRoot)
}

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

func Run(forceReset bool) error {
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("missing required DATABASE_URL environment variable")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	exists, err := checkTableExists(ctx, conn, "cards")
	if err != nil {
		return fmt.Errorf("error checking if tables exist: %w", err)
	}

	if exists {
		if !forceReset {
			return fmt.Errorf("tables already exist. Use --force flag to drop and recreate them")
		}
		if err := dropTables(ctx, conn); err != nil {
			return fmt.Errorf("error dropping tables: %w", err)
		}
	}

	schemaPath := filepath.Join(projectRoot, "app", "drizzle", "0000_initial.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("unable to read schema file: %w", err)
	}

	_, err = conn.Exec(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("unable to execute schema: %w", err)
	}

	fmt.Println("Database schema initialized successfully!")
	return nil
}
