package specs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"gorm.io/gorm"
)

// RejectService handles the rejection of old specification documents
type RejectService struct {
	Logger       *slog.Logger
	GoogleClient *google.Google
	DB           *gorm.DB
	Config       RejectConfig
}

type RejectConfig struct {
	DryRun    bool // If true, will log what would be done without making changes
	TimeStamp string
	CleanupID string
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

// FindStaleSpecs identifies specifications that have "Drafting" or "Braindump" status
func (r *RejectService) FindStaleSpecs(ctx context.Context) ([]*db.Spec, error) {
	// TODO: add a time condition to only select specs that haven't been updated in a while
	var specs []*db.Spec
	err := r.DB.Where("status IN ?", []string{"Drafting", "Braindump"}).Find(&specs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query stale specs: %w", err)
	}

	return specs, nil
}

// RejectAllStaleSpecs finds and rejects all stale specifications
func (r *RejectService) RejectAllStaleSpecs(ctx context.Context) error {
	specs, err := r.FindStaleSpecs(ctx)
	if err != nil {
		return fmt.Errorf("failed to find stale specs: %w", err)
	} else if len(specs) == 0 {
		return nil
	}

	rejectedCount, failedCount := 0, 0
	for _, spec := range specs {
		err := r.RejectSpec(ctx, spec)
		if err != nil {
			r.Logger.Error("failed to reject spec",
				"spec_id", spec.ID,
				"error", err.Error())
			failedCount++
			continue
		}
		rejectedCount++
	}

	r.Logger.Info("completed stale spec rejection",
		"total", len(specs),
		"rejected", rejectedCount,
		"failed", failedCount)

	return nil
}

// RejectSpec updates a specification document to rejected status
func (r *RejectService) RejectSpec(ctx context.Context, spec *db.Spec) error {
	logger := r.Logger.With("spec_id", spec.ID, "doc_id", spec.GoogleDocID)
	rejectedStatus := "Rejected"

	if r.Config.DryRun {
		logger.Info("would reject spec (dry run)")
		return nil
	}

	// Find the status cell coordinates
	coords, err := r.findStatusCell(ctx, spec.GoogleDocID)
	if err != nil {
		return fmt.Errorf("failed to find status cell: %w", err)
	} else if coords == nil {
		return fmt.Errorf("document is not a draft/braindump")
	}

	// Update the Google Doc
	if err := r.updateDocumentStatus(ctx, spec.GoogleDocID, coords, rejectedStatus); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	updateData := map[string]any{
		"status":    rejectedStatus,
		"synced_at": time.Now(),
	}
	if err := r.DB.Model(spec).Where("id = ?", spec.ID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update spec status in database: %w", err)
	}

	logger.Info("successfully rejected spec")

	// Add rejection notice to the document
	if err = r.addRejectionNotice(ctx, spec.GoogleDocID); err != nil {
		if err = r.addFallbackRejectionNotice(ctx, spec.GoogleDocID); err != nil {
			logger.Error("failed to add fallback rejection notice", "error", err.Error())
		}
	}

	return nil
}

// findStatusCell locates the position of a status cell with "Drafting" or "Braindump" in a Google Doc
func (r *RejectService) findStatusCell(ctx context.Context, docID string) (*CellCoordinates, error) {
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
