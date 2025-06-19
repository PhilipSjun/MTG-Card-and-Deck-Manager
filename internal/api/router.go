package api

import (
	"database/sql"
	"net/http"
)

func NewRouter(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/decks", createDeckHandler(db))
	// ... other routes
	return mux
}
