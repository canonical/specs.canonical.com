package db

import (
	"log/slog"

	"github.com/canonical/specs-v2.canonical.com/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(logger *slog.Logger, config *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.GetDBDSN()), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	return db, nil
}
