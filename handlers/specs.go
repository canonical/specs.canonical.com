package handlers

import (
	"net/http"
	"strings"

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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":  total,
		"specs":  specs,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}
