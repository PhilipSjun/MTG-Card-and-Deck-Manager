package decks

import (
	"database/sql"

	"github.com/admin/mtg-card-manager/internal/db"
)

type Service struct {
	DB *sql.DB
}

func (s *Service) CreateDeck(name, owner string) (*db.Deck, error) {
	// TODO: implement
	return nil, nil
}
