package api

import (
	"database/sql"
	"net/http"
)

func createDeckHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement
		w.Write([]byte("create deck"))
	}
}
