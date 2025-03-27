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

// NoCache middleware adds headers to prevent caching
// this is needed for the current production deployment
// with content-cache to prevent exposing private content
func NoCache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Response().Header().Set("Pragma", "no-cache")
		c.Response().Header().Set("Expires", "0")
		return next(c)
	}
}

func NewServer(logger *slog.Logger, config *config.Config, db *gorm.DB) *Server {
	server := &Server{
		Logger: logger,
		Config: config,
		DB:     db,
	}

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(NoCache)

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

	// Handle favicon.ico requests
	e.GET("/favicon.ico", func(c echo.Context) error {
		favicon, err := fsys.Open("favicon.ico")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		defer favicon.Close()
		return c.Stream(http.StatusOK, "image/x-icon", favicon)
	}, server.AuthMiddleware)

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
