package specs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/docs/v1"
)

// cellBoundaryOffset represents the fixed boundary size between cells
// in Google Docs API.
//
// In the Google Docs API, when calculating cell positions in a table,
// each cell is separated by a boundary marker. This constant accounts for:
//   - 1 byte for the cell division character (typically '\n')
//   - 1 byte for the cell start character (typically '\v')
//
// This is used when advancing from one cell's position to the next
// cell's start index.
const cellBoundaryOffset int64 = 2

// cellCoordinates represents the position of a status cell in a
// Google Doc table.
type cellCoordinates struct {
	Row int
	Col int
}

// updateDocumentStatus updates the Google Doc to change status
func (r *RejectService) updateDocumentStatus(
	ctx context.Context,
	docID string,
	coords *cellCoordinates,
	newStatus string,
) error {
	doc, err := r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch document: %v", err)
	}

	var table *docs.Table
	for _, elem := range doc.Body.Content {
		if elem.Table != nil {
			table = elem.Table
			break
		}
	}
	if table == nil {
		return fmt.Errorf("no metadata table found in document")
	}

	if coords.Row >= len(table.TableRows) {
		return fmt.Errorf("row index %d out of bounds (max %d)", coords.Row, len(table.TableRows)-1)
	}
	row := table.TableRows[coords.Row]

	if coords.Col >= len(row.TableCells) {
		return fmt.Errorf("column index %d out of bounds (max %d)", coords.Col, len(row.TableCells)-1)
	}
	cell := row.TableCells[coords.Col]

	if len(cell.Content) == 0 || cell.Content[0].Paragraph == nil || len(cell.Content[0].Paragraph.Elements) == 0 {
		return fmt.Errorf("target cell is empty or malformed")
	}

	contentElements := cell.Content[0].Paragraph.Elements
	cellStartIndex := contentElements[0].StartIndex
	cellEndIndex := contentElements[len(contentElements)-1].EndIndex - 1
	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{StartIndex: cellStartIndex, EndIndex: cellEndIndex},
			},
		}, {
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: cellStartIndex},
				Text:     newStatus,
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to update status cell: %v", err)
	}

	return nil
}

// addRejectionNotice appends a rejection notice to the spec's changelog table
func (r *RejectService) addRejectionNotice(
	ctx context.Context,
	docID string,
	cleanupID string,
) error {
	doc, err := r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch updated document: %v", err)
	}

	// Find the changelog table (table after "Spec History and Changelog" heading)
	var changelogTable *docs.Table
	var changelogTableElementIndex int
	var foundChangelogHeading bool
	for i, element := range doc.Body.Content {
		if element.Table != nil && foundChangelogHeading {
			changelogTable = element.Table
			changelogTableElementIndex = i
			break
		}
		if element.Paragraph == nil {
			continue
		}

		// Check if this is the changelog heading
		for _, elem := range element.Paragraph.Elements {
			if elem.TextRun == nil {
				continue
			}
			text := strings.ToLower(strings.TrimSpace(elem.TextRun.Content))
			if strings.Contains(text, "changelog") ||
				strings.Contains(text, "history") {
				foundChangelogHeading = true
				break
			}
		}
	}
	if !foundChangelogHeading || changelogTable == nil {
		return fmt.Errorf("no 'Spec History and Changelog' table found")
	}
	if len(changelogTable.TableRows[0].TableCells) == 0 {
		return fmt.Errorf("changelog table is malformed")
	}

	tableIndex := changelogTable.TableRows[0].StartIndex - 1
	rejectionRequests := []*docs.Request{{
		InsertTableRow: &docs.InsertTableRowRequest{
			TableCellLocation: &docs.TableCellLocation{
				TableStartLocation: &docs.Location{Index: tableIndex},        // Use the table's start index
				RowIndex:           int64(len(changelogTable.TableRows) - 1), // Last row index
				ColumnIndex:        int64(0),                                 // Reference the first column
			},
			InsertBelow: true,
		},
	}}
	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: rejectionRequests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create new changelog row: %v", err)
	}

	// Refresh the document to get updated table
	// Some tables have defaults for new rows which we need to overwrite
	doc, err = r.GoogleClient.DocsService.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to fetch updated document: %v", err)
	}
	if len(doc.Body.Content) <= changelogTableElementIndex || doc.Body.Content[changelogTableElementIndex].Table == nil {
		return fmt.Errorf("failed to locate updated changelog table at expected position")
	}

	changelogTable = doc.Body.Content[changelogTableElementIndex].Table
	headerColumns := map[string]int{
		"author":  -1,
		"status":  -1,
		"date":    -1,
		"comment": -1,
	}
	for i, headerCell := range changelogTable.TableRows[0].TableCells {
		header := normalizeChangelogHeader(headerCell)
		for key := range headerColumns {
			if headerColumns[key] == -1 && strings.Contains(header, key) {
				headerColumns[key] = i // update the map on 1st occurrence
				break
			}
		}
	}

	// check that all indexes are in bounds
	numColumns := len(changelogTable.TableRows[len(changelogTable.TableRows)-1].TableCells)
	for key, colIdx := range headerColumns {
		if colIdx == -1 {
			return fmt.Errorf("changelog table is missing required column: %s", key)
		}
		if colIdx >= numColumns {
			return fmt.Errorf("changelog table column index out of bounds for key: %s", key)
		}
	}

	cellContents := make([]string, numColumns)
	cellContents[headerColumns["author"]] = "Specs Automations"
	cellContents[headerColumns["status"]] = "Rejected"
	cellContents[headerColumns["date"]] = time.Now().Format("2006-01-02")
	cellContents[headerColumns["comment"]] = fmt.Sprintf(
		"This spec was rejected during the automated cleanup of stale documents (Cleanup ID: %s)",
		cleanupID,
	)

	// Prepare text requests for cells
	newRow := changelogTable.TableRows[len(changelogTable.TableRows)-1]
	var insertTextRequests []*docs.Request
	cellStartIndex := newRow.TableCells[0].Content[0].Paragraph.Elements[0].StartIndex
	for colIdx, content := range cellContents {
		cell := newRow.TableCells[colIdx]
		contentSize := cell.Content[0].Paragraph.Elements[len(cell.Content[0].Paragraph.Elements)-1].EndIndex - 1 - cell.Content[0].Paragraph.Elements[0].StartIndex

		// Delete old content
		if contentSize > 0 {
			insertTextRequests = append(insertTextRequests, &docs.Request{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{StartIndex: cellStartIndex, EndIndex: cellStartIndex + contentSize},
				},
			})
		}

		// Insert new content
		insertTextRequests = append(insertTextRequests, &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Text:     content,
				Location: &docs.Location{Index: cellStartIndex},
			},
		})

		// Move to the start index of the next cell
		cellStartIndex += int64(len(content)) + cellBoundaryOffset
	}

	_, err = r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: insertTextRequests,
	}).Context(ctx).Do()
	if err != nil {
		r.Logger.Error("failed to add rejection notice to changelog", "error", err.Error())
		return fmt.Errorf("failed to insert text into new changelog row: %v", err)
	}

	return nil
}

// addFallbackRejectionNotice creates a fallback rejection message with
// red text when changelog table is not available.
func (r *RejectService) addFallbackRejectionNotice(
	ctx context.Context,
	docID string,
	cleanupID string,
) error {
	rejectionMessage := fmt.Sprintf(
		"This spec was rejected during the automated cleanup of stale documents on %s. Cleanup ID: %s",
		time.Now().Format("2006-01-02"),
		cleanupID,
	)

	rejectionRequests := []*docs.Request{{
		InsertText: &docs.InsertTextRequest{
			Location: &docs.Location{Index: 1},
			Text:     rejectionMessage + "\n\n",
		},
	}, {
		UpdateTextStyle: &docs.UpdateTextStyleRequest{
			Range: &docs.Range{
				StartIndex: 1,
				EndIndex:   int64(len(rejectionMessage)) + 1,
			},
			TextStyle: &docs.TextStyle{
				ForegroundColor: &docs.OptionalColor{
					Color: &docs.Color{
						RgbColor: &docs.RgbColor{Red: 0.8, Green: 0.2, Blue: 0.2},
					},
				},
				Bold: true,
			},
			Fields: "foregroundColor,bold",
		},
	}}

	_, err := r.GoogleClient.DocsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: rejectionRequests,
	}).Context(ctx).Do()

	return err
}

// normalizeChangelogHeader extracts and normalizes text from a table
// cell header.
func normalizeChangelogHeader(cell *docs.TableCell) string {
	if cell == nil || len(cell.Content) == 0 || cell.Content[0].Paragraph == nil {
		return ""
	}
	for _, elem := range cell.Content[0].Paragraph.Elements {
		if elem.TextRun != nil {
			return strings.ToLower(strings.TrimSpace(elem.TextRun.Content))
		}
	}
	return ""
}
