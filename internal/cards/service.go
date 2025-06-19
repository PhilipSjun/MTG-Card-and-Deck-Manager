package cards

import (
	"database/sql"

	"github.com/admin/mtg-card-manager/internal/db"
)

type Service struct {
	DB *sql.DB
}

func (s *Service) GetCard(id int) (*db.Card, error) {
	// TODO: implement
	return nil, nil
}
