package googledrive

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
func (g *GoogleDrive) ListFilesChannel(ctx context.Context, opts QueryOptions) <-chan FileResult {
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
func (g *GoogleDrive) GetSubFoldersChannel(ctx context.Context, folderID string) <-chan FileResult {
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
func (g *GoogleDrive) GetFilesInFolderChannel(ctx context.Context, folderID string) <-chan FileResult {
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
func (g *GoogleDrive) ExportFile(ctx context.Context, fileID string, format string) (string, error) {
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

// DocumentFirstTable gets the first table in a Google Doc
func (g *GoogleDrive) DocumentFirstTable(ctx context.Context, fileID string) (map[string][]string, error) {
	content, err := g.ExportFile(ctx, fileID, MimeTypeHTML)
	if err != nil {
		return nil, err
	}
	tableMap := make(map[string][]string)

	document, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}

	document.Find("table").First().Find("tr").Each(func(i int, s *goquery.Selection) {
		key := strings.ToLower(strings.TrimSpace(s.Find("td").First().Text()))
		valuesNodes := s.Find("td").Last().Find("p > span")
		values := make([]string, 0, valuesNodes.Length())
		valuesNodes.Each(func(i int, s *goquery.Selection) {
			trimmedValue := strings.TrimSpace(s.Text())
			if trimmedValue != "" {
				values = append(values, trimmedValue)
			}
		})

		if key != "" && len(values) > 0 {
			tableMap[key] = values
		}
	})

	return tableMap, nil

}
