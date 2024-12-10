package googledrive

import (
	"fmt"
	"strings"
)

// FieldBuilder helps construct Google Drive fields query parameter
type FieldBuilder struct {
	fields []string
}

const (
	// Basic file fields
	FieldID          = "id"
	FieldName        = "name"
	FieldMimeType    = "mimeType"
	FieldSize        = "size"
	FieldWebViewLink = "webViewLink"
	FieldWebContent  = "webContentLink"

	// Time-related fields
	FieldCreatedTime    = "createdTime"
	FieldModifiedTime   = "modifiedTime"
	FieldModifiedByMe   = "modifiedByMe"
	FieldViewedByMe     = "viewedByMe"
	FieldViewedByMeTime = "viewedByMeTime"

	// Permission-related fields
	FieldOwners       = "owners"
	FieldPermissions  = "permissions"
	FieldShared       = "shared"
	FieldSharedWithMe = "sharedWithMe"
	FieldSharingUser  = "sharingUser"

	// Special fields
	FieldNextPageToken = "nextPageToken"
	FieldFiles         = "files"
	FieldParents       = "parents"
	FieldTrashed       = "trashed"
	FieldOwner         = "owner"
	FieldFullText      = "fullText"
)

func NewFieldBuilder() *FieldBuilder {
	return &FieldBuilder{
		fields: make([]string, 0),
	}
}

// AddField adds a single field
func (f *FieldBuilder) AddField(field string) *FieldBuilder {
	f.fields = append(f.fields, field)
	return f
}

// AddFields adds multiple fields
func (f *FieldBuilder) AddFields(fields ...string) *FieldBuilder {
	f.fields = append(f.fields, fields...)
	return f
}

// Pagination adds nextPageToken and files fields for paginated results
func (f *FieldBuilder) Pagination() *FieldBuilder {
	return f.AddFields(FieldNextPageToken)
}

// Basic adds commonly used basic fields
func (f *FieldBuilder) Basic() *FieldBuilder {
	return f.AddFields(FieldID, FieldName, FieldMimeType)
}

// WithTimes adds time-related fields
func (f *FieldBuilder) WithTimes() *FieldBuilder {
	return f.AddFields(FieldCreatedTime, FieldModifiedTime)
}

// WithPermissions adds permission-related fields
func (f *FieldBuilder) WithPermissions() *FieldBuilder {
	return f.AddFields(FieldOwners, FieldPermissions, FieldShared)
}

// WithLinks adds web view and content links
func (f *FieldBuilder) WithLinks() *FieldBuilder {
	return f.AddFields(FieldWebViewLink, FieldWebContent)
}

// SubFields creates a field specification for nested fields
func (f *FieldBuilder) SubFields(field string, subFields ...string) *FieldBuilder {
	if len(subFields) > 0 {
		f.fields = append(f.fields, fmt.Sprintf("%s(%s)", field, strings.Join(subFields, ", ")))
	} else {
		f.fields = append(f.fields, field)
	}
	return f
}

// Build returns the final fields string
func (f *FieldBuilder) Build() string {
	return strings.Join(f.fields, ", ")
}

// CommonFieldSets provides predefined field combinations
var CommonFieldSets = struct {
	Minimal  func() *FieldBuilder
	Standard func() *FieldBuilder
	Full     func() *FieldBuilder
}{
	Minimal: func() *FieldBuilder {
		return NewFieldBuilder().
			Pagination().
			SubFields(FieldFiles, FieldID, FieldName)
	},
	Standard: func() *FieldBuilder {
		return NewFieldBuilder().
			Pagination().
			SubFields(FieldFiles, FieldID, FieldName, FieldMimeType, FieldModifiedTime, FieldSize)
	},
	Full: func() *FieldBuilder {
		return NewFieldBuilder().
			Pagination().
			SubFields(FieldFiles,
				FieldID,
				FieldName,
				FieldMimeType,
				FieldSize,
				FieldCreatedTime,
				FieldModifiedTime,
				FieldWebViewLink,
				FieldWebContent,
				FieldParents,
				FieldPermissions)
	},
}
