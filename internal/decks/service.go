package decks

import "mtgmanager/internal/db"

type Service struct {
	DB *db.DB
}

func (s *Service) CreateDeck(name, owner string) (*db.Deck, error) {
	// TODO: implement
	return nil, nil
}
