package specs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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

	failedCount   int
	rejectedCount int
}

type RejectConfig struct {
	// DryRun makes the rejection log actions without actually making changes
	DryRun bool
	// RejectThreshold defines how old a spec must be to be considered stale
	RejectThreshold time.Duration
}

// findStaleSpecs identifies specifications that:
//   - Have "Drafting" or "Braindump" status
//   - Have not been updated in the configured threshold period.
func (r *RejectService) findStaleSpecs() ([]*db.Spec, error) {
	var specs []*db.Spec
	err := r.DB.
		Where("LOWER(status) IN ?", []string{"drafting", "braindump"}).
		Where("google_doc_updated_at < ?", time.Now().Add(r.Config.RejectThreshold)).
		Find(&specs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query stale specs: %v", err)
	}

	return specs, nil
}

// RejectAllStaleSpecs finds and rejects all stale specifications
func (r *RejectService) RejectAllStaleSpecs(ctx context.Context) error {
	r.failedCount = 0
	r.rejectedCount = 0
	cleanupID := uuid.New().String()

	specs, err := r.findStaleSpecs()
	if err != nil {
		return fmt.Errorf("failed to find stale specs: %v", err)
	}
	if len(specs) == 0 {
		r.Logger.Info("no stale specs to reject")
		return nil
	}

	// Process specs sequentially
	for _, spec := range specs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger := r.Logger.With("spec_id", spec.ID, "doc_id", spec.GoogleDocID)
		err := r.RejectSpec(ctx, spec, cleanupID)
		if err != nil {
			logger.Error("failed to reject spec", "error", err.Error())
			r.failedCount++
		} else {
			r.rejectedCount++
		}
	}

	r.Logger.Info("completed stale spec rejection",
		"total", len(specs),
		"rejected", r.rejectedCount,
		"failed", r.failedCount)

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
		return fmt.Errorf("failed to find status cell: %v", err)
	}
	if coords == nil {
		return fmt.Errorf("document is not a draft/braindump")
	}

	// Update the Google Doc
	if err := r.updateDocumentStatus(ctx, spec.GoogleDocID, coords, "Rejected"); err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	updateData := map[string]any{
		"status":    "Rejected",
		"synced_at": time.Now(),
	}
	if err := r.DB.Model(spec).Where("id = ?", spec.ID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update spec status in database: %v", err)
	}

	logger.Info("successfully rejected spec")

	// Add rejection notice to the document
	// Rejection notice is not critical, so log error but do not fail
	if err = r.addRejectionNotice(ctx, spec.GoogleDocID, cleanupID); err != nil {
		if err = r.addFallbackRejectionNotice(ctx, spec.GoogleDocID, cleanupID); err != nil {
			logger.Error("failed to add rejection notice", "error", err.Error())
		}
	}

	return nil
}

// findStatusCell locates the position of a spec status cell in a Google Doc
func (r *RejectService) findStatusCell(
	ctx context.Context,
	docID string,
) (*cellCoordinates, error) {
	table, err := r.GoogleClient.DocumentFirstTable(ctx, docID)
	if err != nil || len(table) == 0 {
		return nil, fmt.Errorf("metadata not found or malformed: %v", err)
	}

	// Find the status cell coordinates using table format detection
	var coords *cellCoordinates
	if isColumnFormat(table) {
		coords = findStatusInColumnFormat(table)
	} else {
		coords = findStatusInRowFormat(table)
	}

	return coords, nil
}

// findStatusInColumnFormat searches for status in column-based table format
func findStatusInColumnFormat(table [][]string) *cellCoordinates {
	if len(table) < 4 || len(table[3]) < 3 {
		return nil
	}

	if status := strings.ToLower(strings.TrimSpace(table[3][2])); status == "drafting" || status == "braindump" {
		return &cellCoordinates{Row: 3, Col: 2}
	}

	return nil
}

// findStatusInRowFormat searches for status in row-based table format
func findStatusInRowFormat(table [][]string) *cellCoordinates {
	if len(table) < 3 || len(table[2]) < 2 {
		return nil
	}

	if status := strings.ToLower(strings.TrimSpace(table[2][1])); status == "drafting" || status == "braindump" {
		return &cellCoordinates{Row: 2, Col: 1}
	}

	return nil
}

// RejectSpecByGoogleDocID finds a spec by its Google Doc ID and rejects it
// This is useful for testing with a single file
func (r *RejectService) RejectSpecByGoogleDocID(ctx context.Context, googleDocID string) error {
	logger := r.Logger.With("doc_id", googleDocID)

	// Find the spec in the database
	var spec *db.Spec
	if err := r.DB.Where("google_doc_id = ?", googleDocID).First(&spec).Error; err != nil {
		return fmt.Errorf("failed to find spec with google_doc_id %s: %v", googleDocID, err)
	}

	logger = logger.With("spec_id", spec.ID)
	logger.Info("found spec, attempting rejection")

	cleanupID := uuid.New().String()
	return r.RejectSpec(ctx, spec, cleanupID)
}
