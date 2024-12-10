package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/canonical/specs-v2.canonical.com/config"
	"github.com/canonical/specs-v2.canonical.com/ui"
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

	e.GET("/api/specs", server.ListSpecs, server.AuthMiddleware)
	e.GET("/api/specs/authors", server.SpecAuthors, server.AuthMiddleware)
	e.GET("/api/specs/teams", server.SpecTeams, server.AuthMiddleware)

	// Serve static files from dist directory
	fsys, _ := fs.Sub(ui.UIAssets, "dist")
	staticHandler := http.FileServer(http.FS(fsys))
	e.GET("/assets/*", echo.WrapHandler(staticHandler), server.AuthMiddleware)
	// Serve index.html for all other routes to support client-side routing
	e.GET("*", func(c echo.Context) error {
		indexFile, err := fsys.Open("index.html")
		if err != nil {
			return err
		}
		defer indexFile.Close()
		return c.Stream(http.StatusOK, "text/html", indexFile)
	}, server.AuthMiddleware)

	server.Echo = e
	return server
}
