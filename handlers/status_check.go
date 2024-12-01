package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) StatusCheck(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
