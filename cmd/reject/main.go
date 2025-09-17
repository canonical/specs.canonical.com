package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"github.com/canonical/specs-v2.canonical.com/specs"
	"github.com/google/uuid"
	"google.golang.org/api/drive/v3"
)

func main() {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "perform a dry run without making actual changes")
	flag.Parse()

	c := config.MustLoadConfig()
	logger := config.SetupLogger()

	dbConn, err := db.NewDB(logger, c)
	if err != nil {
		logger.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	rejectService := specs.NewRejectService(
		logger,
		googleDrive,
		dbConn,
		specs.RejectConfig{
			DryRun:    dryRun,
			TimeStamp: time.Now().Format("2006-01-02"),
			CleanupID: uuid.New().String(),
		},
	)

	logger.Info("starting spec rejection job",
		"dry_run", dryRun,
		"pid", os.Getpid())

	if dryRun {
		logger.Info("DRY RUN MODE: No actual changes will be made")
	}

	if err := rejectService.RejectAllStaleSpecs(ctx); err != nil {
		logger.Error("spec rejection failed", "error", err)
		os.Exit(1)
	}

	logger.Info("spec rejection job completed successfully")
}
