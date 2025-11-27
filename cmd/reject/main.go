package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"github.com/canonical/specs-v2.canonical.com/specs"
)

// main implements a command that runs as a daemon to auto-reject specs based
// on how long they have been open. In addition to the daemon mode, it also
// supports rejecting an individual spec, or performing a dry-run on all specs.
func main() {
	var (
		dryRun      bool
		googleDocID string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "perform a dry run without making actual changes")
	flag.StringVar(&googleDocID, "google-doc-id", "", "reject a single doc (optional)")
	flag.Parse()

	cfg := config.MustLoadConfig()
	logger := config.SetupLogger()

	// Signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	rejectService := initRejectService(cfg, logger)
	rejectService.Config = specs.RejectConfig{
		DryRun: dryRun,
	}

	// Handle single file testing
	if googleDocID != "" {
		logger.Info("testing with single file", "file_id", googleDocID)
		if err := rejectService.RejectSpecByGoogleDocID(ctx, googleDocID); err != nil {
			logger.Error("failed to reject spec", "error", err)
			os.Exit(1)
		}
		logger.Info("single file test completed")
		return
	}

	// Set up ticker for periodic runs
	ticker := time.NewTicker(cfg.GetRejectInterval())
	defer ticker.Stop()

	logger.Info("starting rejection job",
		"interval", cfg.GetRejectInterval(),
		"dry_run", dryRun,
		"pid", os.Getpid())

	// Run initial rejection
	if err := rejectService.RejectAllStaleSpecs(ctx); err != nil {
		logger.Error("spec rejection failed", "error", err)
	}
	// Exit after initial dry run: no need to continue running on loop
	if dryRun {
		return
	}

	// Wait for either context cancellation or ticker
	for {
		select {
		case <-ctx.Done():
			logger.Info("rejection job stopped")
			return
		case <-ticker.C:
			logger.Info("starting periodic rejection iteration")
			if err := rejectService.RejectAllStaleSpecs(ctx); err != nil {
				logger.Error("spec rejection failed", "error", err)
			}
		}
	}
}

// initRejectService initializes the Reject service with all its dependencies
func initRejectService(cfg *config.Config, logger *slog.Logger) *specs.RejectService {
	dbConn, err := db.NewDB(logger, cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}

	if err := db.Migrate(dbConn); err != nil {
		log.Fatal(err)
	}

	logger.Info("migrations completed successfully")

	// Create Google client with write access for document updates
	googleDrive, err := google.NewGoogleDrive(google.Config{
		ClientID:     cfg.GoogleClientID,
		ClientEmail:  cfg.GoogleClientEmail,
		PrivateKey:   cfg.GooglePrivateKey,
		PrivateKeyID: cfg.GooglePrivateKeyID,
		ProjectID:    cfg.GoogleProjectID,
		Scopes:       cfg.GetRejectGoogleDriveScopes(),
	})

	if err != nil {
		logger.Error("failed to create google drive client", "error", err.Error())
		os.Exit(1)
	}

	return &specs.RejectService{
		Logger:       logger.With("component", "specs_reject"),
		GoogleClient: googleDrive,
		DB:           dbConn,
	}
}
