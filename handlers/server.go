package handlers

import (
	"log/slog"
	"net/http"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Server struct {
	Logger *slog.Logger
	Config *config.Config
	DB     *gorm.DB
	Echo   *echo.Echo
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func NewServer(logger *slog.Logger, config *config.Config, db *gorm.DB) *Server {
	server := &Server{
		Logger: logger,
		Config: config,
		DB:     db,
	}

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.GET("_status/check", server.StatusCheck)

	e.GET("/auth/google/login", server.HandleGoogleLogin)
	e.GET("/auth/google/callback", server.HandleGoogleCallback)

	e.GET("/specs", server.ListSpecs, server.AuthMiddleware)

	server.Echo = e
	return server
}
