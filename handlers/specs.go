package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/canonical/specs-v2.canonical.com/db"
	"github.com/labstack/echo/v4"
)

type ListSpecsRequest struct {
	Limit    int32  `query:"limit" validate:"min=1,max=100"`
	Offset   int32  `query:"offset" validate:"min=0"`
	OrderBy  string `query:"order_by" validate:"oneof=created_at updated_at title team id"`
	OrderDir string `query:"order_dir" validate:"oneof=asc desc"`
	Title    string `query:"title"`
	Team     string `query:"team"`
	Type     string `query:"type"`
	Author   string `query:"author"`
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
		r.OrderDir = "desc"
	}
}

func (s *Server) ListSpecs(c echo.Context) error {
	req := new(ListSpecsRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid query parameters")
	}
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
	if req.Type != "" {
		query = query.Where("spec_type = ?", req.Type)
	}
	if req.Author != "" {
		query = query.Where("ARRAY_TO_STRING(authors, ' ') ILIKE ?", "%"+strings.TrimSpace(req.Author)+"%")
	}
	if req.OrderBy == "created_at" {
		req.OrderBy = "google_doc_created_at"
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
