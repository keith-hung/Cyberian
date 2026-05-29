// Package session manages authentication, page fetching, and save operations
// against the new Nouveau Timecard (Razor Pages) timesheet server.
package session

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/keith-hung/nouveau-timecard-cli/internal/httpclient"
	"github.com/keith-hung/nouveau-timecard-cli/internal/parser"
	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
)

// sessionMaxAge is conservative relative to the server's 30-min sliding cookie.
const sessionMaxAge = 25 * time.Minute

// Config holds the connection parameters for a session.
type Config struct {
	BaseURL     string
	Username    string
	Password    string
	SessionFile string
	Insecure    bool
}

// Session manages the Nouveau Timecard session lifecycle.
type Session struct {
	client        *httpclient.Client
	config        Config
	authenticated bool
	startTime     time.Time
}

// New creates a Session and attempts to restore a previous one from disk.
func New(cfg Config) (*Session, error) {
	client, err := httpclient.New(cfg.BaseURL, cfg.Insecure)
	if err != nil {
		return nil, err
	}
	s := &Session{client: client, config: cfg}
	s.tryRestore()
	return s, nil
}

func (s *Session) tryRestore() {
	if s.config.SessionFile == "" {
		return
	}
	info, err := s.client.LoadSession(s.config.SessionFile)
	if err != nil {
		return
	}
	startStr, _ := info["sessionStartTime"].(string)
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return
	}
	if time.Since(startTime) > sessionMaxAge {
		s.client.ClearCookies()
		return
	}
	if authed, _ := info["authenticated"].(bool); authed {
		s.authenticated = true
		s.startTime = startTime
	}
}

func (s *Session) saveState() {
	if !s.authenticated || s.config.SessionFile == "" {
		return
	}
	_ = s.client.SaveSession(s.config.SessionFile, map[string]interface{}{
		"authenticated":    true,
		"username":         s.config.Username,
		"sessionStartTime": s.startTime.Format(time.RFC3339),
	})
}

// Login authenticates via the LDAP-backed login form, carrying the antiforgery
// token and cookie issued on the GET of the login page.
func (s *Session) Login() error {
	// 1. GET login page → antiforgery cookie + token.
	resp, err := s.client.Get("")
	if err != nil {
		return fmt.Errorf("fetching login page: %w", err)
	}
	token := parser.ParseAntiForgeryToken(resp.Body)
	if token == "" {
		return fmt.Errorf("could not find antiforgery token on login page")
	}

	// 2. POST credentials.
	form := url.Values{}
	form.Set("username", s.config.Username)
	form.Set("password", s.config.Password)
	form.Set("__RequestVerificationToken", token)

	post, err := s.client.PostForm("", form)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}

	// Success: LocalRedirect("/Home") → 302 with Location containing Home.
	// A bare "/" or empty Location is NOT accepted — it could mask an auth
	// failure that bounced back to the login page.
	if post.Status >= 300 && post.Status < 400 {
		loc := post.Headers.Get("Location")
		if strings.Contains(loc, "Home") || strings.Contains(loc, "Timesheet") {
			s.authenticated = true
			s.startTime = time.Now()
			s.saveState()
			return nil
		}
		return fmt.Errorf("login redirected unexpectedly to %q", loc)
	}

	// 200 means the login page was re-rendered with an error message.
	if strings.Contains(post.Body, "Invalid username or password") ||
		strings.Contains(post.Body, "輸入資訊有誤") {
		return fmt.Errorf("invalid username or password")
	}
	if strings.Contains(post.Body, "User not found") || strings.Contains(post.Body, "無法取得使用者資訊") {
		return fmt.Errorf("user not found")
	}
	return fmt.Errorf("login failed (status %d)", post.Status)
}

// EnsureAuth ensures the session is authenticated, logging in if needed.
func (s *Session) EnsureAuth() error {
	if s.authenticated && !s.startTime.IsZero() && time.Since(s.startTime) > sessionMaxAge {
		s.authenticated = false
		s.client.ClearCookies()
	}
	if s.authenticated {
		resp, err := s.client.Get("Home")
		if err == nil && !parser.IsLoginPage(resp.Body) {
			return nil
		}
		s.authenticated = false
	}
	return s.Login()
}

// timesheetPath builds the Timesheet GET path for a year/month.
func timesheetPath(year, month int) string {
	return fmt.Sprintf("Timesheet?year=%d&month=%d", year, month)
}

// FetchTimesheetPage fetches the Timesheet HTML for the given year/month,
// re-authenticating once if the session has expired.
func (s *Session) FetchTimesheetPage(year, month int) (string, error) {
	if err := s.EnsureAuth(); err != nil {
		return "", err
	}
	resp, err := s.client.Get(timesheetPath(year, month))
	if err != nil {
		return "", err
	}
	if parser.IsLoginPage(resp.Body) {
		if err := s.Login(); err != nil {
			return "", fmt.Errorf("re-authentication failed: %w", err)
		}
		resp, err = s.client.Get(timesheetPath(year, month))
		if err != nil {
			return "", err
		}
		if parser.IsLoginPage(resp.Body) {
			return "", fmt.Errorf("session expired and re-authentication failed")
		}
	}
	return resp.Body, nil
}

// GetProjects returns the authorized project options for the given month.
func (s *Session) GetProjects(year, month int) ([]types.ProjectOption, error) {
	html, err := s.FetchTimesheetPage(year, month)
	if err != nil {
		return nil, err
	}
	return parser.ParseProjectOptions(html), nil
}

// GetActivities returns the selectable activities for one project.
func (s *Session) GetActivities(year, month int, projectID string) ([]types.Activity, error) {
	html, err := s.FetchTimesheetPage(year, month)
	if err != nil {
		return nil, err
	}
	cache, err := parser.ParseActivityCache(html)
	if err != nil {
		return nil, err
	}
	return cache[projectID], nil
}

// PageData bundles everything parsed from one timesheet page load.
type PageData struct {
	HTML        string
	Token       string
	TimeRows    []types.ServerRow
	OvertimeRows []types.ServerRow
}

// FetchPageData fetches and parses the token + existing rows for a month.
func (s *Session) FetchPageData(year, month int) (*PageData, error) {
	html, err := s.FetchTimesheetPage(year, month)
	if err != nil {
		return nil, err
	}
	token := parser.ParseAntiForgeryToken(html)
	if token == "" {
		return nil, fmt.Errorf("could not find antiforgery token on timesheet page")
	}
	timeRows, err := parser.ParseServerRows(html, "serverTimeRows")
	if err != nil {
		return nil, fmt.Errorf("parsing serverTimeRows: %w", err)
	}
	overtimeRows, err := parser.ParseServerRows(html, "serverOvertimeRows")
	if err != nil {
		return nil, fmt.Errorf("parsing serverOvertimeRows: %w", err)
	}
	return &PageData{HTML: html, Token: token, TimeRows: timeRows, OvertimeRows: overtimeRows}, nil
}

// Client exposes the underlying HTTP client for save/leave operations.
func (s *Session) Client() *httpclient.Client { return s.client }
