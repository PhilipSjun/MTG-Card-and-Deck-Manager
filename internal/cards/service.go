package cards

import "mtgmanager/internal/db"

type Service struct {
	DB *db.DB
}

func (s *Service) GetCard(id int) (*db.Card, error) {
	// TODO: implement
	return nil, nil
}
