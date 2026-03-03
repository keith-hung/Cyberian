// Package session manages authentication, page fetching, and save operations
// against the TimeCard server.
package session

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/keith-hung/timecard-cli/internal/httpclient"
	"github.com/keith-hung/timecard-cli/internal/parser"
	"github.com/keith-hung/timecard-cli/internal/types"
)

const sessionMaxAge = 25 * time.Minute

// Config holds the connection parameters for a session.
type Config struct {
	BaseURL     string
	Username    string
	Password    string
	SessionFile string
}

// Session manages the TimeCard session lifecycle.
type Session struct {
	client        *httpclient.Client
	config        Config
	authenticated bool
	username      string
	startTime     time.Time
}

// New creates a new Session with the given config and attempts to restore a previous session.
func New(cfg Config) (*Session, error) {
	client, err := httpclient.New(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	s := &Session{
		client: client,
		config: cfg,
	}

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
	if startStr == "" {
		return
	}
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return
	}
	if time.Since(startTime) > sessionMaxAge {
		s.client.ClearCookies()
		return
	}

	authed, _ := info["authenticated"].(bool)
	if authed {
		s.authenticated = true
		s.username, _ = info["username"].(string)
		s.startTime = startTime
		log.Println("[Session] Restored session from saved state")
	}
}

func (s *Session) saveState() {
	if !s.authenticated || s.config.SessionFile == "" {
		return
	}
	info := map[string]interface{}{
		"authenticated":    s.authenticated,
		"username":         s.username,
		"sessionStartTime": s.startTime.Format(time.RFC3339),
	}
	if err := s.client.SaveSession(s.config.SessionFile, info); err != nil {
		log.Printf("[Session] Failed to save session state: %v", err)
	}
}

// Login authenticates with the TimeCard server.
func (s *Session) Login() error {
	resp, err := s.client.Post("servlet/VerifController", map[string]string{
		"method": "login",
		"name":   s.config.Username,
		"pw":     s.config.Password,
	})
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}

	if resp.Status == 302 {
		location := resp.Headers.Get("Location")

		if strings.Contains(location, "ini_mainframe") || strings.Contains(location, "project") {
			// Success — follow redirect to establish full session
			if _, err := s.client.Get(location); err != nil {
				return fmt.Errorf("following login redirect: %w", err)
			}
			s.authenticated = true
			s.username = s.config.Username
			s.startTime = time.Now()
			s.saveState()
			return nil
		}

		// Detect specific failure reasons
		switch {
		case strings.Contains(location, "login_namefail"):
			return fmt.Errorf("username not found")
		case strings.Contains(location, "login_pwfail"):
			return fmt.Errorf("wrong password")
		case strings.Contains(location, "login_notavailable"):
			return fmt.Errorf("account not available")
		case strings.Contains(location, "noseats"):
			return fmt.Errorf("no available seats")
		case strings.Contains(location, "error"):
			return fmt.Errorf("server error")
		}
	}

	return fmt.Errorf("unexpected response status %d", resp.Status)
}

// EnsureAuth ensures the session is authenticated, logging in if needed.
func (s *Session) EnsureAuth() error {
	// Check session age
	if s.authenticated && !s.startTime.IsZero() {
		if time.Since(s.startTime) > sessionMaxAge {
			s.authenticated = false
			s.client.ClearCookies()
		}
	}

	// Verify with a lightweight request
	if s.authenticated {
		resp, err := s.client.Get("Timecard/timecard_week/daychoose.jsp")
		if err == nil && !parser.IsLoginPage(resp.Body, resp.URL) {
			return nil
		}
		s.authenticated = false
	}

	return s.Login()
}

// FetchTimesheetPage fetches the timesheet HTML for the given date.
// If date is empty, fetches the current week.
func (s *Session) FetchTimesheetPage(date string) (string, error) {
	path := "Timecard/timecard_week/daychoose.jsp"
	if date != "" {
		path += "?cho_date=" + date
	}

	resp, err := s.client.Get(path)
	if err != nil {
		return "", err
	}

	// Session expired — re-auth and retry
	if parser.IsLoginPage(resp.Body, resp.URL) {
		if err := s.EnsureAuth(); err != nil {
			return "", fmt.Errorf("re-authentication failed: %w", err)
		}
		resp, err = s.client.Get(path)
		if err != nil {
			return "", err
		}
		if parser.IsLoginPage(resp.Body, resp.URL) {
			return "", fmt.Errorf("session expired and re-authentication failed")
		}
	}

	if parser.IsErrorPage(resp.Body) {
		info := parser.ParseErrorPage(resp.Body)
		if info != nil {
			return "", fmt.Errorf("server error: %s", info.MainMessage)
		}
		return "", fmt.Errorf("server returned error page")
	}

	return resp.Body, nil
}

// GetProjects returns available projects from the timesheet page.
func (s *Session) GetProjects(date string) ([]types.ProjectOption, error) {
	html, err := s.FetchTimesheetPage(date)
	if err != nil {
		return nil, err
	}
	return parser.ParseProjectOptions(html), nil
}

// GetActivities returns activity entries from the timesheet page.
func (s *Session) GetActivities(date string) ([]types.ActivityEntry, error) {
	html, err := s.FetchTimesheetPage(date)
	if err != nil {
		return nil, err
	}
	return parser.ParseActivityList(html), nil
}

// TimesheetData holds all parsed data from a single page fetch.
type TimesheetData struct {
	HTML             string
	Activities       []types.ActivityEntry
	TimeEntries      []types.TimearrayEntry
	OvertimeEntries  []types.TimearrayEntry
	Projects         []types.ProjectOption
	WeekDate         string
}

// FetchTimesheetData fetches and parses all timesheet data in a single page load.
func (s *Session) FetchTimesheetData(date string) (*TimesheetData, error) {
	html, err := s.FetchTimesheetPage(date)
	if err != nil {
		return nil, err
	}

	return &TimesheetData{
		HTML:            html,
		Activities:      parser.ParseActivityList(html),
		TimeEntries:     parser.ParseTimearray(html),
		OvertimeEntries: parser.ParseOvertimeArray(html),
		Projects:        parser.ParseProjectOptions(html),
		WeekDate:        parser.ParseWeekDate(html),
	}, nil
}
