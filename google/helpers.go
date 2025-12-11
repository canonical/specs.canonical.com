package google

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// FileResult represents a single file result with potential error
type FileResult struct {
	File *drive.File
	Err  error
}

// ListFilesChannel streams files matching the provided query options through a channel
func (g *Google) ListFilesChannel(ctx context.Context, opts QueryOptions) <-chan FileResult {
	resultChan := make(chan FileResult)

	go func() {
		defer close(resultChan)

		pageToken := ""
		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				resultChan <- FileResult{Err: ctx.Err()}
				return
			default:
			}

			call := g.DriveService.Files.List().
				Context(ctx).
				Q(opts.Query).
				Fields(googleapi.Field(opts.Fields))

			if opts.SupportsAllDrives {
				call = call.SupportsAllDrives(true)
			}
			if opts.IncludeItemsFromAllDrives {
				call = call.IncludeItemsFromAllDrives(true)
			}
			if pageToken != "" {
				call = call.PageToken(pageToken)
			}

			fileList, err := call.Do()
			if err != nil {

				resultChan <- FileResult{Err: err}
				return
			}

			// Send each file through the channel
			for _, file := range fileList.Files {
				select {
				case resultChan <- FileResult{File: file}:
				case <-ctx.Done():
					resultChan <- FileResult{Err: ctx.Err()}
					return
				}
			}

			pageToken = fileList.NextPageToken
			if pageToken == "" {
				break
			}
		}
	}()

	return resultChan
}

// GetSubFoldersChannel streams subfolders of the provided folder ID through a channel
func (g *Google) GetSubFoldersChannel(ctx context.Context, folderID string) <-chan FileResult {
	qb := NewQueryBuilder()
	query := qb.IsFolder().
		InParent(folderID).
		Build()

	fb := NewFieldBuilder()
	fields := fb.Pagination().
		SubFields(FieldFiles, FieldID, FieldName).
		Build()

	opts := QueryOptions{
		Query:                     query,
		Fields:                    fields,
		SupportsAllDrives:         true,
		IncludeItemsFromAllDrives: true,
	}

	return g.ListFilesChannel(ctx, opts)
}

// GetFilesInFolderChannel streams files in the provided folder ID through a channel
func (g *Google) GetFilesInFolderChannel(ctx context.Context, folderID string) <-chan FileResult {
	qb := NewQueryBuilder()
	query := qb.NotTrashed().
		InParent(folderID).
		MimeType(MimeTypeDocument).
		Build()

	fb := NewFieldBuilder()
	fields := fb.Pagination().
		SubFields(FieldFiles,
			FieldID,
			FieldName,
			FieldModifiedTime,
			FieldCreatedTime,
			FieldWebViewLink).
		Build()

	opts := QueryOptions{
		Query:                     query,
		Fields:                    fields,
		SupportsAllDrives:         true,
		IncludeItemsFromAllDrives: true,
	}

	return g.ListFilesChannel(ctx, opts)
}

// ExportFile exports a Google Doc to markdown format
func (g *Google) ExportFile(ctx context.Context, fileID string, format string) (string, error) {
	resp, err := g.DriveService.Files.Export(fileID, format).Context(ctx).Download()
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	markdown := string(content)

	return markdown, nil
}

// DocumentFirstTable extracts the first table from a Google Document and converts it into a 2D string array.
// It takes a context and a file ID as input parameters and returns a 2D slice of strings representing
// the table's content.
//
// The function performs the following steps:
// 1. Exports the Google Document as HTML using the provided file ID.
// 2. Parses the HTML content to find the first table element.
// 3. Iterates over each row ("tr") in the table.
// 4. For each row, extracts the text content from each cell ("th" and "td").
// 5. If a cell contains mailto links, it extracts the email addresses and joins them with commas.
// 6. Appends the extracted row data to the result slice.
//
// If no table is found or no data could be extracted, it returns an error.
//
// Parameters:
// - ctx: The context for the request.
// - fileID: The ID of the Google Document file.
//
// Returns:
// - A 2D slice of strings representing the table's content.
// - An error if any issues occur during the extraction process.
func (g *Google) DocumentFirstTable(ctx context.Context, fileID string) ([][]string, error) {
	content, err := g.ExportFile(ctx, fileID, MimeTypeHTML)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	table := doc.Find("table").First()

	if table.Length() == 0 {
		return nil, fmt.Errorf("no table found in the document")
	}

	var result [][]string

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		var rowData []string

		// Extract text from each cell (both th and td) in the row
		row.Find("th, td").Each(func(j int, cell *goquery.Selection) {
			// Check if the cell contains any mailto links (user badges)
			mailtoLinks := cell.Find("a[href^='mailto:']")
			if mailtoLinks.Length() > 0 {
				var authors []string
				mailtoLinks.Each(func(k int, link *goquery.Selection) {
					authorName := strings.TrimSpace(link.Text())
					authors = append(authors, authorName)
				})
				if len(authors) > 0 {
					rowData = append(rowData, strings.Join(authors, ","))
				} else {
					rowData = append(rowData, strings.TrimSpace(cell.Text()))
				}
			} else {
				// default to text content otherwise
				rowData = append(rowData, strings.TrimSpace(cell.Text()))
			}
		})

		if len(rowData) > 0 {
			result = append(result, rowData)
		}
	})

	if len(result) == 0 {
		return nil, fmt.Errorf("table found but no data could be extracted")
	}

	return result, nil
}

// AddDocComment adds a comment to the specified Google Doc.
//
// Parameters:
//   - ctx: The context for the request.
//   - fileID: The ID of the Google Doc to which the comment will be added.
//   - content: The text content of the comment to add.
//
// This function requires the drive.DriveScope permission to create comments
// (drive.DriveReadonlyScope is not sufficient).
//
// Returns an error if the comment cannot be created, for example due to
// permission issues, an invalid fileID, or network problems.
func (g *Google) AddDocComment(ctx context.Context, fileID, content string) error {
	_, err := g.DriveService.Comments.
		Create(fileID, &drive.Comment{Content: content}).
		Context(ctx).
		Fields("id").
		Do()
	return err
}
