package specs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RejectService handles the rejection of old specification documents
type RejectService struct {
	Logger       *slog.Logger
	GoogleClient *google.Google
	DB           *gorm.DB
	Config       RejectConfig

	FailedCount   int
	RejectedCount int
}

type RejectConfig struct {
	DryRun        bool // If true, will log what would be done without making changes
	MaxGoroutines int
}

// NewRejectService creates a new specification rejection service
func NewRejectService(logger *slog.Logger, googleClient *google.Google, db *gorm.DB, config RejectConfig) *RejectService {
	return &RejectService{
		Logger:       logger.With("component", "specs_reject"),
		GoogleClient: googleClient,
		DB:           db,
		Config:       config,
	}
}

// worker processes specs from a channel in a goroutine
func (r *RejectService) worker(ctx context.Context, id int, specs <-chan *db.Spec, cleanupID string) {
	logger := r.Logger.With("worker_id", id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped due to cancellation")
			return
		case spec, ok := <-specs:
			if !ok {
				return
			}
			logger := logger.With("spec_id", spec.ID, "doc_id", spec.GoogleDocID)
			err := r.RejectSpec(ctx, spec, cleanupID)
			if err != nil {
				logger.Error("failed to reject spec", "error", err.Error())
				r.FailedCount++
			} else {
				r.RejectedCount++
			}
		}
	}
}

// findStaleSpecs identifies specifications that have "Drafting" or "Braindump" status
// and have not been updated in the last 6 months
func (r *RejectService) findStaleSpecs() ([]*db.Spec, error) {
	var specs []*db.Spec
	err := r.DB.
		Where("status IN ?", []string{"Drafting", "Braindump"}).
		Where("google_doc_updated_at < ?", time.Now().AddDate(0, -6, 0)).
		Find(&specs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query stale specs: %w", err)
	}

	return specs, nil
}

// RejectAllStaleSpecs finds and rejects all stale specifications
func (r *RejectService) RejectAllStaleSpecs(ctx context.Context) error {
	r.Logger.Info("starting stale spec rejection job",
		"dry_run", r.Config.DryRun,
		"max_goroutines", r.Config.MaxGoroutines,
	)
	r.FailedCount = 0
	r.RejectedCount = 0
	cleanupID := uuid.New().String()

	specs, err := r.findStaleSpecs()
	if err != nil {
		return fmt.Errorf("failed to find stale specs: %w", err)
	}
	if len(specs) == 0 {
		r.Logger.Info("no stale specs to reject")
		return nil
	}

	// Create channel for work distribution
	workerItems := make(chan *db.Spec, r.Config.MaxGoroutines)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < r.Config.MaxGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.worker(ctx, id, workerItems, cleanupID)
		}(i)
	}

	// Send specs to workers
	go func() {
		defer close(workerItems)
		for _, spec := range specs {
			select {
			case workerItems <- spec:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all workers to finish
	wg.Wait()

	r.Logger.Info("completed stale spec rejection",
		"total", len(specs),
		"rejected", r.RejectedCount,
		"failed", r.FailedCount)

	return nil
}

// RejectSpec updates a specification document to rejected status
func (r *RejectService) RejectSpec(
	ctx context.Context,
	spec *db.Spec,
	cleanupID string,
) error {
	logger := r.Logger.With("spec_id", spec.ID, "doc_id", spec.GoogleDocID)

	if r.Config.DryRun {
		logger.Info("would reject spec (dry run)")
		return nil
	}

	// Find the status cell coordinates
	coords, err := r.findStatusCell(ctx, spec.GoogleDocID)
	if err != nil {
		return fmt.Errorf("failed to find status cell: %w", err)
	}
	if coords == nil {
		return fmt.Errorf("document is not a draft/braindump")
	}

	// Update the Google Doc
	if err := r.updateDocumentStatus(ctx, spec.GoogleDocID, coords, "Rejected"); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	updateData := map[string]any{
		"status":    "Rejected",
		"synced_at": time.Now(),
	}
	if err := r.DB.Model(spec).Where("id = ?", spec.ID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update spec status in database: %w", err)
	}

	logger.Info("successfully rejected spec")

	// Add rejection notice to the document
	if err = r.addRejectionNotice(ctx, spec.GoogleDocID, cleanupID); err != nil {
		if err = r.addFallbackRejectionNotice(ctx, spec.GoogleDocID, cleanupID); err != nil {
			logger.Error("failed to add fallback rejection notice", "error", err.Error())
		}
	}

	return nil
}

// findStatusCell locates the position of a status cell with "Drafting" or "Braindump" in a Google Doc
func (r *RejectService) findStatusCell(
	ctx context.Context,
	docID string,
) (*CellCoordinates, error) {
	// Get the first table from the document using the same approach as parser.go
	table, err := r.GoogleClient.DocumentFirstTable(ctx, docID)
	if err != nil || len(table) == 0 {
		return nil, fmt.Errorf("metadata not found or malformed: %w", err)
	}

	// Find the status cell coordinates using table format detection
	var coords *CellCoordinates
	if isColumnFormat(table) {
		coords = findStatusInColumnFormat(table)
	} else {
		coords = findStatusInRowFormat(table)
	}

	return coords, nil
}

// findStatusInColumnFormat searches for status in column-based table format
func findStatusInColumnFormat(table [][]string) *CellCoordinates {
	if len(table) < 4 || len(table[3]) < 3 {
		return nil
	}

	if status := strings.ToLower(strings.TrimSpace(table[3][2])); status == "drafting" || status == "braindump" {
		return &CellCoordinates{Row: 3, Col: 2}
	}

	return nil
}

// findStatusInRowFormat searches for status in row-based table format
func findStatusInRowFormat(table [][]string) *CellCoordinates {
	if len(table) < 3 || len(table[2]) < 2 {
		return nil
	}

	if status := strings.ToLower(strings.TrimSpace(table[2][1])); status == "drafting" || status == "braindump" {
		return &CellCoordinates{Row: 2, Col: 1}
	}

	return nil
}
