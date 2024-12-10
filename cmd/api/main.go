package main

import (
	"os"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/handlers"
)

func main() {

	c := config.MustLoadConfig()
	logger := config.SetupLogger()

	db, err := db.NewDB(logger, c)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	server := handlers.NewServer(logger, c, db)

	err = server.Echo.Start(server.Config.GetHost())
	if err != nil {
		server.Logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}

}
