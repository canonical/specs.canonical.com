package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/canonical/specs-v2.canonical.com/config"
)

type GoogleUser struct {
	Email string `json:"email"`
	HD    string `json:"hd"`
}

func (s *Server) initGoogleOAuth() *oauth2.Config {
	callbackURL, _ := url.JoinPath(s.Config.BaseURL, "/auth/google/callback")
	return &oauth2.Config{
		ClientID:     s.Config.GoogleOAuthClientID,
		ClientSecret: s.Config.GoogleOAuthClientSecret,
		RedirectURL:  callbackURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := c.Cookie("session_token")
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/auth/google/login")
		}

		if !isValidSession(s.Config, session.Value) {
			return c.Redirect(http.StatusTemporaryRedirect, "/auth/google/login")
		}

		c.Set("email", emailFromSession(session.Value))

		return next(c)
	}
}

func (s *Server) HandleGoogleLogin(c echo.Context) error {
	oauth := s.initGoogleOAuth()
	url := oauth.AuthCodeURL("state")
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Server) HandleGoogleCallback(c echo.Context) error {
	oauth := s.initGoogleOAuth()

	code := c.QueryParam("code")
	token, err := oauth.Exchange(c.Request().Context(), code)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to exchange token")
	}

	client := oauth.Client(c.Request().Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user info")
	}
	defer resp.Body.Close()

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to decode user info")
	}

	sessionToken := createSession(s.Config, user.Email)

	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Value = sessionToken
	cookie.Path = "/"
	c.SetCookie(cookie)

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func createSession(config *config.Config, email string) string {
	timestamp := time.Now().Unix()
	payload := email + ":" + string(timestamp)

	h := hmac.New(sha256.New, []byte(config.AppSecretKey))
	h.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Combine payload and signature
	token := base64.URLEncoding.EncodeToString([]byte(payload)) + "." + signature
	return token
}

func isValidSession(config *config.Config, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}

	payload, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	// Recreate signature
	h := hmac.New(sha256.New, []byte(config.AppSecretKey))
	h.Write(payload)
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return parts[1] == expectedSig
}

func emailFromSession(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ""
	}

	payload, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return ""
	}

	return strings.Split(string(payload), ":")[0]
}
