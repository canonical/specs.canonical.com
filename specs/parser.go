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

	parts := strings.SplitN(file.File.Name, "-", 2)
	var specId, specTitle string
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

	if !s.Config.ForceSync {
		var updatedAt time.Time
		s.DB.Model(&db.Spec{}).Where("id = ?", specId).Pluck("google_doc_updated_at", &updatedAt)
		if !updatedAt.IsZero() && updatedAt.Equal(googleDocUpdatedAt) {
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
		SyncedAt:           time.Now(),
	}

	specsMetadataTable, err := s.GoogleClient.DocumentFirstTable(ctx, file.File.Id)
	if err != nil {
		return fmt.Errorf("failed to get first table: %w", err)
	}
	logger.Debug("metadata table", "table", specsMetadataTable)

	if len(specsMetadataTable) == 0 {
		return fmt.Errorf("metadata table is empty")
	}

	if isColumnFormat(specsMetadataTable) {
		parseColumnBasedMetadata(specsMetadataTable, &newSpec)
	} else {
		parseRowBasedMetadata(specsMetadataTable, &newSpec)
	}

	logger.Debug("creating spec", "specs", newSpec)
	if err := s.DB.Where(db.Spec{ID: newSpec.ID}).Assign(newSpec).FirstOrCreate(&newSpec).Error; err != nil {
		return fmt.Errorf("failed to upsert spec: %w", err)
	}

	return nil
}

// isColumnFormat checks if the given table has the old specification design or the new one.
// Old design is row-based, where each row contains a key-value pair. Where the table will look like:
/*
[
 	["authors", "user1@canonical,user2@canonical.com,user3@canonical"],
	["created", "2021-09-13"],
	["index", "SN114"]
]
*/
// And the new design is column-based, where the table will look like:
/*
[
	["Index", "PR001", "", ""],
	["Title", "Specifications - Purpose and Guidance"],
	["Type", "Author(s)", "Status", "Created"],
	[
		"Process",
		"user1@canonical.com,user2@canonical.com,user3@canonical.com",
		"Approved",
		"Apr 22, 2021"
	],
	["", "Reviewer(s)", "Status", "Date"],
	["", "user4@canonical.com", "Approved", "Aug 11, 2023"],
	["", "user3@canonical.com", "Approved", "Aug 11, 2023"],
	["", "user2@canonical.com", "Approved", "Aug 11, 2023"],
	["", "user1@canonical.com", "Approved", "Jul 11, 2023"]
]
*/
// The function expects that the table's third row (index 2) should contain the column headers.
// The expected column headers are "type", "author(s)", "status", and "created".
// The function returns true if column headers are found in the, and false otherwise.
func isColumnFormat(table [][]string) bool {
	expectedKeys := []string{"type", "author(s)", "status", "created"}
	if len(table) < 4 {
		return false
	}

	foundKeys := 0
	for _, cell := range table[2] {
		for _, expected := range expectedKeys {
			if strings.EqualFold(cell, expected) {
				foundKeys++
				break
			}
		}
	}
	return foundKeys == len(expectedKeys)
}

func parseRowBasedMetadata(table [][]string, spec *db.Spec) {
	for _, row := range table {
		if len(row) < 2 {
			continue
		}
		key := strings.ToLower(row[0])
		value := row[1]

		switch key {
		case "title":
			if spec.Title == nil || *spec.Title == "" {
				spec.Title = &value
			}
		case "index":
			if spec.ID == "" {
				spec.ID = value
			}
		case "status":
			spec.Status = &value
		case "authors":
			spec.Authors = parseAuthors([]string{value})
		case "type":
			spec.SpecType = &value
		}
	}
}

func parseColumnBasedMetadata(table [][]string, spec *db.Spec) {
	if len(table) < 4 {
		return
	}

	keysRow := table[2]
	valuesRow := table[3]
	if len(keysRow) != len(valuesRow) {
		return
	}

	for i, key := range keysRow {
		key = strings.ToLower(key)
		value := valuesRow[i]

		switch key {
		case "title":
			if spec.Title == nil || *spec.Title == "" {
				spec.Title = &value
			}
		case "index":
			if spec.ID == "" {
				spec.ID = value
			}
		case "status":
			spec.Status = &value
		case "author(s)":
			spec.Authors = parseAuthors([]string{value})
		case "type":
			spec.SpecType = &value
		}
	}
}

func parseAuthors(values []string) []string {
	authors := []string{}
	for _, value := range values {
		for _, author := range strings.FieldsFunc(value, AuthorsSplit) {
			author = strings.TrimSpace(strings.Split(author, "<")[0])
			if len(author) > 4 {
				authors = append(authors, author)
			}
		}
	}
	return authors
}

func AuthorsSplit(r rune) bool {
	return r == ',' || r == ';' || r == '/' || r == '|'
}
