package specs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
)

func (s *SyncService) Parse(ctx context.Context, logger *slog.Logger, workerItem *WorkerItem) error {
	file := workerItem.File
	parentFolder := workerItem.ParentFolder

	logger.Debug("processing file")

	// google doc title: format {id} - {title}
	parts := strings.SplitN(file.File.Name, "-", 2)
	var specId string
	var specTitle string
	if len(parts) == 2 {
		specId, specTitle = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	parsedTime, err := time.Parse(time.RFC3339, file.File.ModifiedTime)
	if err != nil {
		return fmt.Errorf("failed to parse google doc updated time: %w", err)
	}
	googleDocUpdatedAt := parsedTime

	parsedTime, err = time.Parse(time.RFC3339, file.File.CreatedTime)
	if err != nil {
		return fmt.Errorf("failed to parse google doc created time: %w", err)
	}
	googleDocCreatedAt := parsedTime

	// check if spec hasn't changed since last sync
	if !s.Config.ForceSync {
		var updatedAt time.Time
		s.DB.Raw("SELECT google_doc_updated_at FROM specs WHERE id = ? LIMIT 1", specId).Scan(&updatedAt)
		if !updatedAt.IsZero() && updatedAt == googleDocUpdatedAt {
			logger.Debug("spec hasn't changed since last sync")
			s.DB.Model(&db.Spec{}).Where("id = ?", specId).Update("synced_at", time.Now())
			s.SkippedCount++
			return nil
		}
	}

	newSpec := db.Spec{
		ID:                 specId,
		Title:              &specTitle,
		Team:               parentFolder.File.Name,
		GoogleDocID:        file.File.Id,
		GoogleDocName:      file.File.Name,
		GoogleDocURL:       file.File.WebViewLink,
		GoogleDocCreatedAt: googleDocCreatedAt,
		GoogleDocUpdatedAt: googleDocUpdatedAt,
	}

	specsMetadatabTable, err := s.DriveClient.DocumentFirstTable(ctx, file.File.Id)
	if err != nil {
		return fmt.Errorf("failed to get first table: %w", err)
	}
	logger.Debug("metadata table", "table", specsMetadatabTable)
	for key, values := range specsMetadatabTable {
		switch key {
		case "title":
			if specTitle == "" {
				specTitle = values[0]
			}
		case "index":
			if specId == "" {
				specId = values[0]
			}
		case "status":
			newSpec.Status = &values[0]
		case "authors":
			newSpec.Authors = []string{}
			for _, author := range values {
				formattedAuthor := strings.TrimSpace(author)
				authorValid := len(formattedAuthor) > 3
				if authorValid {
					newSpec.Authors = append(newSpec.Authors, formattedAuthor)
				}
			}
		case "type":
			newSpec.SpecType = &values[0]
		}
	}
	logger.Debug("creating spec", "specs", newSpec)
	if err := s.DB.Where(db.Spec{ID: newSpec.ID}).Assign(newSpec).FirstOrCreate(&newSpec).Error; err != nil {
		return fmt.Errorf("failed to upsert spec: %w", err)
	}

	return nil
}
