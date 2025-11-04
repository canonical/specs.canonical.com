package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/labstack/echo/v4"
)

type ListSpecsRequest struct {
	Limit       int32    `query:"limit" validate:"min=1,max=100"`
	Offset      int32    `query:"offset" validate:"min=0"`
	OrderBy     string   `query:"orderBy" validate:"oneof=created_at updated_at title team id"`
	OrderDir    string   `query:"orderDir" validate:"oneof=asc desc"`
	Title       string   `query:"title"`
	Team        string   `query:"team"`
	Type        []string `query:"type"`
	Status      []string `query:"status"`
	Author      string   `query:"author"`
	SearchQuery string   `query:"searchQuery"`
}

type Spec struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	Status             string    `json:"status"`
	Authors            []string  `json:"authors"`
	SpecType           string    `json:"spec_type"`
	Team               string    `json:"team"`
	GoogleDocID        string    `json:"google_doc_id"`
	GoogleDocName      string    `json:"google_doc_name"`
	GoogleDocURL       string    `json:"google_doc_url"`
	GoogleDocCreatedAt time.Time `json:"google_doc_created_at"`
	GoogleDocUpdatedAt time.Time `json:"google_doc_updated_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	SyncedAt           time.Time `json:"synced_at"`
}

type ListSpecsResponse struct {
	Total  int64  `json:"total"`
	Specs  []Spec `json:"specs"`
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
}

func (r *ListSpecsRequest) setDefaults() {
	if r.Limit == 0 {
		r.Limit = 10
	}
	if r.OrderBy == "" {
		r.OrderBy = "created_at"
	}
	if r.OrderDir == "" {
		if r.OrderBy == "updated_at" || r.OrderBy == "created_at" {
			r.OrderDir = "desc"
		} else {
			r.OrderDir = "asc"
		}
	}
}

func (s *Server) ListSpecs(c echo.Context) error {
	req := new(ListSpecsRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid query parameters")
	}
	q := c.Request().URL.Query()
	req.Type = q["type"]
	req.Status = q["status"]

	req.setDefaults()
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var specs []db.Spec
	query := s.DB.Model(&db.Spec{})

	if req.Title != "" {
		query = query.Where("title ILIKE ?", "%"+req.Title+"%")
	}
	if req.Team != "" {
		query = query.Where("team ILIKE ?", "%"+req.Team+"%")
	}
	if len(req.Type) > 0 {
		query = query.Where("spec_type IN (?)", req.Type)
	}
	if len(req.Status) > 0 {
		lowercaseStatus := make([]string, len(req.Status))

		for i, status := range req.Status {
			lowercaseStatus[i] = strings.ToLower(status)
		}
		query = query.Where("lower(status) IN ?", lowercaseStatus)
	}

	if req.Author != "" {
		query = query.Where("ARRAY_TO_STRING(authors, ' ') ILIKE ?", "%"+strings.TrimSpace(req.Author)+"%")
	}

	if req.OrderBy == "created_at" {
		req.OrderBy = "google_doc_created_at"
	}
	if req.OrderBy == "updated_at" {
		req.OrderBy = "google_doc_updated_at"
	}

	if req.SearchQuery != "" {
		searchConfig := "english" // or any other language configuration
		searchFields := []string{"id", "title", "team", "google_doc_name"}

		// Create concatenated text search vector
		vectorExpr := fmt.Sprintf(
			"to_tsvector('%s', %s)",
			searchConfig,
			strings.Join(searchFields, " || ' ' || "),
		)

		// Create plain text query (converts search terms to tsquery)
		queryExpr := fmt.Sprintf(
			"plainto_tsquery('%s', ?)",
			searchConfig,
		)

		textFilter := s.DB.Where(
			fmt.Sprintf("%s @@ %s", vectorExpr, queryExpr), req.SearchQuery,
		)

		// also add ilike query for the same search fields
		for _, field := range searchFields {
			textFilter = textFilter.Or(fmt.Sprintf("%s ILIKE ?", field), "%"+req.SearchQuery+"%")
		}

		query = query.Where(textFilter)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count specs")
	}

	result := query.
		Order(req.OrderBy + " " + req.OrderDir).
		Limit(int(req.Limit)).
		Offset(int(req.Offset)).
		Find(&specs)

	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch specs")
	}

	specsList := ListSpecsResponse{
		Total:  total,
		Specs:  make([]Spec, len(specs)),
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	for i, spec := range specs {
		specsList.Specs[i].ID = spec.ID

		if spec.Title != nil {
			specsList.Specs[i].Title = *spec.Title
		}
		specsList.Specs[i].Team = spec.Team
		if spec.SpecType != nil {
			specsList.Specs[i].SpecType = *spec.SpecType
		}
		specsList.Specs[i].Authors = spec.Authors
		if specsList.Specs[i].Authors == nil {
			specsList.Specs[i].Authors = []string{}
		}
		if spec.Status != nil {
			specsList.Specs[i].Status = *spec.Status
		}

		specsList.Specs[i].GoogleDocID = spec.GoogleDocID
		specsList.Specs[i].GoogleDocName = spec.GoogleDocName
		specsList.Specs[i].GoogleDocURL = spec.GoogleDocURL
		specsList.Specs[i].GoogleDocCreatedAt = spec.GoogleDocCreatedAt
		specsList.Specs[i].GoogleDocUpdatedAt = spec.GoogleDocUpdatedAt
		specsList.Specs[i].CreatedAt = spec.CreatedAt
		specsList.Specs[i].UpdatedAt = spec.UpdatedAt
		specsList.Specs[i].SyncedAt = spec.SyncedAt
	}
	return c.JSON(http.StatusOK, specsList)
}

func (s *Server) SpecAuthors(c echo.Context) error {
	var uniqueAuthors []string
	if err := s.DB.Model(&db.Spec{}).
		Select("DISTINCT UNNEST(authors) as author").
		Order("author").
		Pluck("author", &uniqueAuthors).
		Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch authors: "+err.Error())
	}
	return c.JSON(http.StatusOK, uniqueAuthors)
}

func (s *Server) SpecTeams(c echo.Context) error {
	var uniqueTeams []string
	if err := s.DB.Model(&db.Spec{}).
		Select("DISTINCT team").
		Order("team").
		Pluck("team", &uniqueTeams).
		Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch teams: "+err.Error())
	}
	return c.JSON(http.StatusOK, uniqueTeams)
}
