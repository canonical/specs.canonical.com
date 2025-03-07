package google

import (
	"fmt"
	"strings"
	"time"
)

// QueryBuilder helps construct Google Drive search queries
type QueryBuilder struct {
	conditions []string
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		conditions: make([]string, 0),
	}
}

// Common MIME types
const (
	MimeTypeFolder   = "application/vnd.google-apps.folder"
	MimeTypeDocument = "application/vnd.google-apps.document"
	MimeTypeSheet    = "application/vnd.google-apps.spreadsheet"
	MimeTypePDF      = "application/pdf"
	MimeTypeHTML     = "text/html"
	MimeTypeMarkdown = "text/markdown"
)

// Operators
const (
	OperatorContains  = "contains"
	OperatorEquals    = "="
	OperatorNotEquals = "!="
)

// Name adds a name condition
func (q *QueryBuilder) Name(operator, value string) *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("%s %s '%s'", FieldName, operator, value))
	return q
}

// MimeType adds a mimeType condition
func (q *QueryBuilder) MimeType(mimeType string) *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("%s = '%s'", FieldMimeType, mimeType))
	return q
}

// IsFolder adds a condition to find folders
func (q *QueryBuilder) IsFolder() *QueryBuilder {
	return q.MimeType(MimeTypeFolder)
}

// InParent adds a parent folder condition
func (q *QueryBuilder) InParent(parentID string) *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("'%s' in %s", parentID, FieldParents))
	return q
}

// ModifiedAfter adds a modified time condition
func (q *QueryBuilder) ModifiedAfter(t time.Time) *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("%s > '%s'", FieldModifiedTime, t.Format(time.RFC3339)))
	return q
}

// ModifiedBefore adds a modified time condition
func (q *QueryBuilder) ModifiedBefore(t time.Time) *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("%s < '%s'", FieldModifiedTime, t.Format(time.RFC3339)))
	return q
}

// NotTrashed adds condition to exclude trashed files
func (q *QueryBuilder) NotTrashed() *QueryBuilder {
	q.conditions = append(q.conditions, fmt.Sprintf("%s = false", FieldTrashed))
	return q
}

// Or combines conditions with OR
func (q *QueryBuilder) Or(conditions ...*QueryBuilder) *QueryBuilder {
	var subQueries []string
	for _, condition := range conditions {
		subQueries = append(subQueries, condition.Build())
	}
	q.conditions = append(q.conditions, fmt.Sprintf("(%s)", strings.Join(subQueries, " or ")))
	return q
}

// And combines conditions with AND
func (q *QueryBuilder) And(conditions ...*QueryBuilder) *QueryBuilder {
	var subQueries []string
	for _, condition := range conditions {
		subQueries = append(subQueries, condition.Build())
	}
	q.conditions = append(q.conditions, fmt.Sprintf("(%s)", strings.Join(subQueries, " and ")))
	return q
}

// Build returns the final query string
func (q *QueryBuilder) Build() string {
	return strings.Join(q.conditions, " and ")
}
