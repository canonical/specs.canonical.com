package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"github.com/canonical/specs-v2.canonical.com/specs"
	"google.golang.org/api/drive/v3"
)

func main() {
	var (
		dryRun      bool
		googleDocID string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "perform a dry run without making actual changes")
	flag.StringVar(&googleDocID, "google-doc-id", "", "reject a single doc (optional)")
	flag.Parse()

	c := config.MustLoadConfig()
	logger := config.SetupLogger()

	dbConn, err := db.NewDB(logger, c)
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
		ClientID:          "112404606310881291739",
		ClientEmail:       "specs-reader@roadmap-270011.iam.gserviceaccount.com",
		ClientX509CertURL: "https://www.googleapis.com/robot/v1/metadata/x509/specs-reader%40roadmap-270011.iam.gserviceaccount.com",
		PrivateKey:        c.GooglePrivateKey,
		PrivateKeyID:      c.GooglePrivateKeyID,
		ProjectID:         "roadmap-270011",
		Scopes:            []string{drive.DriveScope},
	})

	if err != nil {
		logger.Error("failed to create google drive client", "error", err.Error())
		os.Exit(1)
	}

	// Signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	rejectService := specs.NewRejectService(
		logger,
		googleDrive,
		dbConn,
		specs.RejectConfig{
			DryRun: dryRun,
		},
	)

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
	ticker := time.NewTicker(c.GetRejectInterval())
	defer ticker.Stop()

	logger.Info("starting rejection job",
		"interval", c.GetRejectInterval(),
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
			if err := rejectService.RejectAllStaleSpecs(ctx); err != nil {
				logger.Error("spec rejection failed", "error", err)
			}
		}
	}
}
