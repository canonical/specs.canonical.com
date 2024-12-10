package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/googledrive"
	"github.com/canonical/specs-v2.canonical.com/specs"
)

func main() {

	c := config.MustLoadConfig()
	logger := config.SetupLogger()

	db, err := db.NewDB(logger, c)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}

	googleDrive, err := googledrive.NewGoogleDrive(googledrive.Config{
		ClientID:          "112404606310881291739",
		ClientEmail:       "specs-reader@roadmap-270011.iam.gserviceaccount.com",
		ClientX509CertURL: "https://www.googleapis.com/robot/v1/metadata/x509/specs-reader%40roadmap-270011.iam.gserviceaccount.com",
		PrivateKey:        c.GooglePrivateKey,
		PrivateKeyID:      c.GooglePrivateKeyID,
		ProjectID:         "roadmap-270011",
	})

	if err != nil {
		logger.Error("failed to create google drive client", "error", err.Error())
		os.Exit(1)
	}

	// signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncService := specs.NewSyncService(
		logger,
		googleDrive,
		db,
		specs.SyncConfig{
			RootFolderID:  "19jxxVn_3n6ZAmFl3DReEVgZjxZnlky4X",
			MaxGoroutines: 15,
		},
	)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()
	// Setup ticker for periodic sync
	ticker := time.NewTicker(c.GetSyncInterval())
	defer ticker.Stop()

	firstTick := make(chan time.Time, 1)
	firstTick <- time.Now()

	logger.Info("starting sync job",
		"interval", c.GetSyncInterval().String(),
		"pid", os.Getpid())

	// Run initial sync
	syncService.Config.ForceSync = true
	if err := syncService.SyncSpecs(ctx); err != nil {
		logger.Error("initial sync failed", "error", err)
	}
	syncService.Config.ForceSync = false

	// Wait for either context cancellation or ticker
	for {
		select {
		case <-ctx.Done():
			logger.Info("sync job stopped")
			return
		case <-ticker.C:
			if err := syncService.SyncSpecs(ctx); err != nil {
				logger.Error("sync failed", "error", err)
			}
		}
	}

}
