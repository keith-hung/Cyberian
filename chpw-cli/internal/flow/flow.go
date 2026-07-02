// Package flow drives the portal's two-step change-password sequence:
// login (post credentials → server sends SMS OTP) then submit (post new
// password + OTP). Session state (cookies + Submit-page token) is persisted
// between the two so an agent can relay the OTP across turns.
package flow

import (
	"fmt"
	"net/url"
	"time"

	"github.com/keith-hung/chpw-cli/internal/httpclient"
	"github.com/keith-hung/chpw-cli/internal/parser"
)

// Server-advertised limits, surfaced to the user.
const (
	OtpTTL     = 120
	SessionTTL = 180
)

// submitGrace is a client-side guard: refuse to submit once the persisted
// session is almost certainly dead, rather than burning the one-time OTP.
const submitGrace = 150 * time.Second

// Config holds connection parameters.
type Config struct {
	BaseURL     string
	Username    string
	SessionFile string
	Insecure    bool
}

// Client drives the change-password flow.
type Client struct {
	http *httpclient.Client
	cfg  Config
}

// LoginResult reports the outcome of a successful login step.
type LoginResult struct {
	Message    string
	OtpTTL     int
	SessionTTL int
}

// New creates a flow Client.
func New(cfg Config) (*Client, error) {
	hc, err := httpclient.New(cfg.BaseURL, cfg.Insecure)
	if err != nil {
		return nil, err
	}
	return &Client{http: hc, cfg: cfg}, nil
}

// resultBody returns the body of a POST, following the POST's redirect (and any
// further hops via Get) so the caller always inspects the rendered result page.
func (c *Client) resultBody(post *httpclient.Response) (string, error) {
	if post.Status >= 300 && post.Status < 400 {
		loc := post.Headers.Get("Location")
		if loc == "" {
			return "", fmt.Errorf("redirect with no Location")
		}
		r2, err := c.http.Get(loc)
		if err != nil {
			return "", err
		}
		return r2.Body, nil
	}
	return post.Body, nil
}

// Login posts credentials. On success the server sends an SMS OTP and returns
// the OTP form; we persist cookies + that form's token for the submit step.
func (c *Client) Login(password, method string) (LoginResult, error) {
	page, err := c.http.Get("")
	if err != nil {
		return LoginResult{}, err
	}
	token := parser.ParseAntiForgeryToken(page.Body)
	if token == "" {
		return LoginResult{}, fmt.Errorf("could not find antiforgery token on login page")
	}

	form := url.Values{}
	form.Set("Username", c.cfg.Username)
	form.Set("Password", password)
	form.Set("Method", method)
	form.Set("__RequestVerificationToken", token)

	post, err := c.http.PostForm("", form)
	if err != nil {
		return LoginResult{}, err
	}
	body, err := c.resultBody(post)
	if err != nil {
		return LoginResult{}, err
	}

	if parser.IsOtpPage(body) {
		submitToken := parser.ParseAntiForgeryToken(body)
		if submitToken == "" {
			return LoginResult{}, fmt.Errorf("could not find antiforgery token on OTP page")
		}
		if err := c.http.SaveSession(c.cfg.SessionFile, map[string]interface{}{
			"submitToken": submitToken,
			"username":    c.cfg.Username,
			"savedAt":     time.Now().Format(time.RFC3339),
		}); err != nil {
			return LoginResult{}, fmt.Errorf("persisting session: %w", err)
		}
		return LoginResult{
			Message:    "OTP sent to your registered phone",
			OtpTTL:     OtpTTL,
			SessionTTL: SessionTTL,
		}, nil
	}

	if parser.IsLoginPage(body) {
		if msg := parser.ParseErrorMessage(body); msg != "" {
			return LoginResult{}, fmt.Errorf("authentication failed: %s", msg)
		}
		return LoginResult{}, fmt.Errorf("authentication failed: invalid username or password")
	}
	return LoginResult{}, fmt.Errorf("unexpected response after login (status %d)", post.Status)
}

// Submit loads the persisted session and posts the new password + OTP.
func (c *Client) Submit(newPassword, otp string) error {
	info, err := c.http.LoadSession(c.cfg.SessionFile)
	if err != nil {
		return fmt.Errorf("no pending change-password session (run login first): %w", err)
	}
	submitToken, _ := info["submitToken"].(string)
	if submitToken == "" {
		return fmt.Errorf("session is missing the submit token; run login again")
	}
	if savedStr, ok := info["savedAt"].(string); ok {
		if savedAt, perr := time.Parse(time.RFC3339, savedStr); perr == nil {
			if time.Since(savedAt) > submitGrace {
				return fmt.Errorf("validation: session likely expired (>%ds); run login again", int(submitGrace.Seconds()))
			}
		}
	}

	form := url.Values{}
	form.Set("NewPassword", newPassword)
	form.Set("ConfirmPassword", newPassword)
	form.Set("Otp", otp)
	form.Set("__RequestVerificationToken", submitToken)

	post, err := c.http.PostForm("ChangePassword/Submit", form)
	if err != nil {
		return err
	}
	body, err := c.resultBody(post)
	if err != nil {
		return err
	}

	if parser.IsCompletePage(body) {
		return nil
	}
	if msg := parser.ParseErrorMessage(body); msg != "" {
		return fmt.Errorf("validation: %s", msg)
	}
	return fmt.Errorf("validation: password change failed (check OTP and password policy)")
}
