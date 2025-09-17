package specs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/canonical/specs-v2.canonical.com/google"
	"google.golang.org/api/docs/v1"
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

// StatusCellCoordinates represents the position of a status cell in a Google Doc table
type StatusCellCoordinates struct {
	RowIndex    int
	ColumnIndex int
}

// ChangelogColumnMapping represents the column indices for changelog table
type ChangelogColumnMapping struct {
	AuthorIndex  int
	StatusIndex  int
	DateIndex    int
	CommentIndex int
}

// FindStaleSpecs identifies specifications that have "Drafting" or "Braindump" status
func (r *RejectService) FindStaleSpecs(ctx context.Context) ([]*db.Spec, error) {
	var specs []*db.Spec

	// Find specs with Drafting or Braindump status
	err := r.DB.Where("status IN ?", []string{"Drafting", "Braindump"}).Find(&specs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query stale specs: %w", err)
	}

	r.Logger.Info("found stale specs", "count", len(specs))
	return specs, nil
}

// FindStatusCellCoordinates locates the position of a status cell with "Drafting" or "Braindump" in a Google Doc
func (r *RejectService) FindStatusCellCoordinates(ctx context.Context, docID string) (*StatusCellCoordinates, error) {
	// Get the first table from the document using the same approach as parser.go
	table, err := r.GoogleClient.DocumentFirstTable(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get first table: %w", err)
	}

	if len(table) == 0 {
		return nil, fmt.Errorf("metadata table is empty")
	}

	// Find the status cell coordinates using table format detection
	coords := r.findStatusInTable(table)
	if coords == nil {
		return nil, fmt.Errorf("no drafting or braindump status found in document")
	}

	return coords, nil
}

// findStatusInTable searches for "Drafting" or "Braindump" status in a table
func (r *RejectService) findStatusInTable(table [][]string) *StatusCellCoordinates {
	if r.isColumnFormat(table) {
		// Handle column-based format
		return r.findStatusInColumnFormat(table)
	} else {
		// Handle row-based format
		return r.findStatusInRowFormat(table)
	}
}

// findStatusInColumnFormat searches for status in column-based table format
func (r *RejectService) findStatusInColumnFormat(table [][]string) *StatusCellCoordinates {
	if len(table) < 4 {
		return nil
	}

	keysRow := table[2]
	valuesRow := table[3]
	if len(keysRow) != len(valuesRow) {
		return nil
	}

	for i, key := range keysRow {
		if strings.EqualFold(key, "status") {
			cellText := strings.ToLower(strings.TrimSpace(valuesRow[i]))
			if cellText == "drafting" || cellText == "braindump" {
				r.Logger.Debug("found stale status in column format",
					"status", cellText,
					"row", 3,
					"column", i)

				return &StatusCellCoordinates{
					RowIndex:    3,
					ColumnIndex: i,
				}
			}
		}
	}
	return nil
}

// findStatusInRowFormat searches for status in row-based table format
func (r *RejectService) findStatusInRowFormat(table [][]string) *StatusCellCoordinates {
	for rowIndex, row := range table {
		if len(row) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(row[0]))
		value := strings.ToLower(strings.TrimSpace(row[1]))

		if key == "status" && (value == "drafting" || value == "braindump") {
			r.Logger.Debug("found stale status in row format",
				"status", value,
				"row", rowIndex,
				"column", 1)

			return &StatusCellCoordinates{
				RowIndex:    rowIndex,
				ColumnIndex: 1,
			}
		}
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
// The function will then check the fifth row (index 4) and following for the reviewers information.
func (r *RejectService) isColumnFormat(table [][]string) bool {
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

// RejectSpec updates a specification document to rejected status
func (r *RejectService) RejectSpec(ctx context.Context, spec *db.Spec) error {
	logger := r.Logger.With("spec_id", spec.ID, "doc_id", spec.GoogleDocID)

	if r.Config.DryRun {
		logger.Info("would reject spec (dry run)")
		return nil
	}

	logger.Info("rejecting spec")

	// Find the status cell coordinates
	coords, err := r.FindStatusCellCoordinates(ctx, spec.GoogleDocID)
	if err != nil {
		return fmt.Errorf("failed to find status cell: %w", err)
	}
	logger.Info("Found status cell coordinates")

	// Update the Google Doc
	if err := r.updateDocumentStatus(ctx, spec.GoogleDocID, coords, "Rejected"); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	logger.Info("Updated Google Doc status to 'Rejected'")

	// TODO: Check if this db update is done correctly (focus: "Update the local spec object to reflect the change")
	// Update the spec status to "Rejected" and update the synced_at timestamp
	rejectedStatus := "Rejected"
	updateData := map[string]interface{}{
		"status":    &rejectedStatus,
		"synced_at": time.Now(),
	}

	if err := r.DB.Model(spec).Where("id = ?", spec.ID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update spec status in database: %w", err)
	}

	// Update the local spec object to reflect the change
	spec.Status = &rejectedStatus
	spec.SyncedAt = time.Now()

	logger.Info("successfully rejected spec")

	// Add rejection notice to the document
	err = r.addRejectionNotice(ctx, spec.GoogleDocID)
	if err != nil && r.addFallbackRejectionNotice(ctx, spec.GoogleDocID) != nil {
		logger.Error("failed to add fallback rejection notice", "error", err.Error())
		return fmt.Errorf("failed to add fallback rejection notice: %w", err)
	}

	return nil
}

// updateDocumentStatus updates the Google Doc to change status
func (r *RejectService) updateDocumentStatus(ctx context.Context, docID string, coords *StatusCellCoordinates, newStatus string) error {
	// STEP 1: Get the document
	doc, err := r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch document: %w", err)
	}

	// STEP 2: Locate the metadata table
	var table *docs.Table
	for _, elem := range doc.Body.Content {
		if elem.Table != nil {
			table = elem.Table
			break
		}
	}
	if table == nil {
		return fmt.Errorf("no table found in document")
	}

	// STEP 3: Locate the specific cell (for validation purposes)
	if coords.RowIndex >= len(table.TableRows) {
		return fmt.Errorf("row index %d out of bounds", coords.RowIndex)
	}
	row := table.TableRows[coords.RowIndex]

	if coords.ColumnIndex >= len(row.TableCells) {
		return fmt.Errorf("column index %d out of bounds", coords.ColumnIndex)
	}
	cell := row.TableCells[coords.ColumnIndex]

	if len(cell.Content) == 0 || cell.Content[0].Paragraph == nil || len(cell.Content[0].Paragraph.Elements) == 0 {
		return fmt.Errorf("target cell content is empty or malformed")
	}

	// STEP 4: Get the text content and indices for the cell
	// We need to get the actual text content of the cell to determine proper ranges
	var cellStartIndex, cellEndIndex int64
	var hasValidContent = false
	for _, content := range cell.Content {
		if content.Paragraph != nil && len(content.Paragraph.Elements) > 0 {
			// Get the start of the first element and end of the last element
			// The cellEndIndex includes the paragraph marker, so we exclude the last char
			cellStartIndex = content.Paragraph.Elements[0].StartIndex
			cellEndIndex = content.Paragraph.Elements[len(content.Paragraph.Elements)-1].EndIndex - 1
			hasValidContent = true
			break
		}
	}

	if !hasValidContent {
		return fmt.Errorf("cell does not contain valid text content")
	}

	// STEP 5: Update requests to update spec Status
	updateRequests := []*docs.Request{
		{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{
					StartIndex: cellStartIndex,
					EndIndex:   cellEndIndex,
				},
			},
		},
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: cellStartIndex,
				},
				Text: newStatus,
			},
		},
	}

	// STEP 6: Send update requests
	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: updateRequests,
	}).Context(ctx).Do()
	if err != nil {
		r.Logger.Error("failed to update status cell", "error", err.Error(), "requests", updateRequests)
		return fmt.Errorf("failed to update status cell: %w", err)
	}

	return nil
}

func (r *RejectService) addRejectionNotice(ctx context.Context, docID string) error {
	doc, err := r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch updated document: %w", err)
	}

	// STEP 1: Find the changelog table (last table after "Spec History and Changelog" heading)
	var changelogTable *docs.Table
	var foundChangelogHeading bool
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			// Check if this paragraph contains the changelog heading
			for _, elem := range element.Paragraph.Elements {
				if elem.TextRun != nil {
					text := strings.ToLower(strings.TrimSpace(elem.TextRun.Content))
					if strings.Contains(text, "spec history and changelog") ||
						strings.Contains(text, "changelog") ||
						strings.Contains(text, "history") {
						foundChangelogHeading = true
						r.Logger.Debug("found changelog heading", "text", text)
						break
					}
				}
			}
		} else if element.Table != nil && foundChangelogHeading {
			changelogTable = element.Table
			r.Logger.Debug("found table after changelog heading")
		}
	}
	if !foundChangelogHeading || changelogTable == nil {
		return fmt.Errorf("no 'Spec History and Changelog' table found")
	}

	// STEP 2: Validate table structure and get column mapping
	if len(changelogTable.TableRows) == 0 || len(changelogTable.TableRows[0].TableCells) == 0 {
		return fmt.Errorf("changelog table is empty")
	}

	headerRow := changelogTable.TableRows[0]
	columnMapping := &ChangelogColumnMapping{
		AuthorIndex:  -1,
		StatusIndex:  -1,
		DateIndex:    -1,
		CommentIndex: -1,
	}
	for i, cell := range headerRow.TableCells {
		if len(cell.Content) == 0 || cell.Content[0].Paragraph == nil {
			continue
		}
		for _, elem := range cell.Content[0].Paragraph.Elements {
			if elem.TextRun == nil {
				continue
			}

			header := strings.ToLower(strings.TrimSpace(elem.TextRun.Content))
			switch {
			case strings.Contains(header, "author"):
				columnMapping.AuthorIndex = i
			case strings.Contains(header, "status"):
				columnMapping.StatusIndex = i
			case strings.Contains(header, "date"):
				columnMapping.DateIndex = i
			case strings.Contains(header, "comment"):
				columnMapping.CommentIndex = i
			}
			break
		}
	}

	numColumns := len(changelogTable.TableRows[len(changelogTable.TableRows)-1].TableCells)
	cellContents := make([]string, numColumns)

	cellContents[columnMapping.AuthorIndex] = "Specs Automations"
	cellContents[columnMapping.StatusIndex] = "Rejected"
	cellContents[columnMapping.DateIndex] = time.Now().Format("Jan 02, 2006")
	cellContents[columnMapping.CommentIndex] = fmt.Sprintf(
		"This spec was rejected during the automated cleanup of stale documents (Cleanup ID: %s)",
		r.Config.CleanupID,
	)

	// STEP 3: Create insertion requests for new row
	tableIndex := changelogTable.TableRows[0].StartIndex - 1
	rejectionRequests := []*docs.Request{
		{
			InsertTableRow: &docs.InsertTableRowRequest{
				TableCellLocation: &docs.TableCellLocation{
					TableStartLocation: &docs.Location{
						Index: tableIndex, // Use the table's start index
					},
					RowIndex:    int64(len(changelogTable.TableRows) - 1), // Last row index
					ColumnIndex: int64(0),                                 // Reference the first column
				},
				InsertBelow: true,
			},
		},
	}
	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: rejectionRequests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create new changelog row: %w", err)
	}

	// STEP 4: Insert text into each cell of the new row
	// Refresh the document to get updated table structure (table index should be the same)
	doc, err = r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch updated document: %w", err)
	}

	// Look for cell indexes in the last row of the changelog table
	// Look for the updated changelog table in the refreshed document
	var updatedChangelogTable *docs.Table
	for _, element := range doc.Body.Content {
		if element.Table != nil && element.StartIndex == tableIndex {
			updatedChangelogTable = element.Table
			break
		}
	}
	if updatedChangelogTable == nil {
		return fmt.Errorf("failed to locate updated changelog table at expected position")
	}

	// Get the newly inserted last row
	newRow := updatedChangelogTable.TableRows[len(updatedChangelogTable.TableRows)-1]

	// Prepare insert text requests for each cell
	var insertTextRequests []*docs.Request
	cellStartIndex := newRow.TableCells[0].Content[0].Paragraph.Elements[0].StartIndex
	for colIdx, content := range cellContents {
		if content == "" {
			continue // skip empty cells
		}

		cell := newRow.TableCells[colIdx]
		contentSize := cell.Content[0].Paragraph.Elements[len(cell.Content[0].Paragraph.Elements)-1].EndIndex - 1 - cell.Content[0].Paragraph.Elements[0].StartIndex
		// Delete old content
		if contentSize > 0 {
			insertTextRequests = append(insertTextRequests, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						StartIndex: cellStartIndex,
						EndIndex:   cellStartIndex + contentSize,
					},
				},
			})
		}

		// Insert new content
		insertTextRequests = append(insertTextRequests, &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Text: content,
				Location: &docs.Location{
					Index: cellStartIndex,
				},
			},
		})

		// Move to the start index of the next cell
		cellStartIndex += int64(len(content) + 2) // +1 for cell division and start
	}

	r.Logger.Info("inserting rejection notice into changelog", "requests", insertTextRequests)

	// Perform the text insertions in a single batch request
	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: insertTextRequests,
	}).Context(ctx).Do()

	if err != nil {
		return fmt.Errorf("failed to insert text into new changelog row: %w", err)
	}

	return nil
}

// createFallbackRejectionMessage creates a fallback rejection message with red text when changelog table is not available
func (r *RejectService) addFallbackRejectionNotice(ctx context.Context, docID string) error {
	rejectionMessage := fmt.Sprintf(
		"This spec was rejected during the automated cleanup of stale documents on %s. Cleanup ID: %s",
		r.Config.TimeStamp,
		r.Config.CleanupID,
	)

	rejectionRequests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: 1},
				Text:     rejectionMessage + "\n\n",
			},
		},
		{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				Range: &docs.Range{
					StartIndex: 1,
					EndIndex:   int64(len(rejectionMessage)) + 1,
				},
				TextStyle: &docs.TextStyle{
					ForegroundColor: &docs.OptionalColor{
						Color: &docs.Color{
							RgbColor: &docs.RgbColor{
								Red:   0.8,
								Green: 0.2,
								Blue:  0.2,
							},
						},
					},
					Bold: true,
				},
				Fields: "foregroundColor,bold",
			},
		},
	}

	_, err := r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: rejectionRequests,
	}).Context(ctx).Do()

	return err
}

// RejectAllStaleSpecs finds and rejects all stale specifications
func (r *RejectService) RejectAllStaleSpecs(ctx context.Context) error {
	specs, err := r.FindStaleSpecs(ctx)
	if err != nil {
		return fmt.Errorf("failed to find stale specs: %w", err)
	}

	if len(specs) == 0 {
		r.Logger.Info("no stale specs found")
		return nil
	}

	rejectedCount := 0
	failedCount := 0

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
