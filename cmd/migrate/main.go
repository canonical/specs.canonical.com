package main

import (
	"log"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
)

func main() {

	c := config.MustLoadConfig()
	logger := config.SetupLogger()
	dbConn, err := db.NewDB(logger, c)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		log.Fatal(err)
	}

	if err := db.Migrate(dbConn); err != nil {
		log.Fatal(err)
	}

	logger.Info("migrations completed successfully")
}
