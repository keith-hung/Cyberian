// Package httpclient provides an HTTP client with cookie management,
// manual redirect following, and session persistence for TCRS.
package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// Response holds the result of an HTTP request.
type Response struct {
	Status     int
	Headers    http.Header
	Body       string
	URL        string
	Redirected bool
}

// Client is an HTTP client with cookie jar and redirect control.
type Client struct {
	baseURL    string
	httpClient *http.Client
	jar        *cookiejar.Jar
}

// New creates a new Client with the given base URL.
func New(baseURL string) (*Client, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	client := &Client{
		baseURL: baseURL,
		jar:     jar,
	}

	client.httpClient = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		// Disable automatic redirect following — we handle it manually
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client, nil
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) resolveURL(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	return c.baseURL + pathOrURL
}

// Get performs an HTTP GET request. Follows redirects manually.
func (c *Client) Get(pathOrURL string) (*Response, error) {
	reqURL := c.resolveURL(pathOrURL)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	// Follow redirects manually
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" {
			return c.followRedirects(resp, reqURL, 10)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		Status:     resp.StatusCode,
		Headers:    resp.Header,
		Body:       string(body),
		URL:        reqURL,
		Redirected: false,
	}, nil
}

// Post performs an HTTP POST with form-encoded body.
// Does NOT follow redirects — returns the raw 302 so callers can inspect Location.
func (c *Client) Post(pathOrURL string, formData map[string]string) (*Response, error) {
	reqURL := c.resolveURL(pathOrURL)

	values := url.Values{}
	for k, v := range formData {
		values.Set(k, v)
	}

	resp, err := c.httpClient.Post(reqURL, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		Status:     resp.StatusCode,
		Headers:    resp.Header,
		Body:       string(body),
		URL:        reqURL,
		Redirected: false,
	}, nil
}

func (c *Client) followRedirects(initialResp *http.Response, initialURL string, maxRedirects int) (*Response, error) {
	currentURL := initialURL
	resp := initialResp
	redirectCount := 0

	for resp.StatusCode >= 300 && resp.StatusCode < 400 && redirectCount < maxRedirects {
		location := resp.Header.Get("Location")
		if location == "" {
			break
		}

		// Resolve relative URLs
		currentURL = resolveRedirectURL(currentURL, location)

		// Close previous body before making new request
		resp.Body.Close()

		newResp, err := c.httpClient.Get(currentURL)
		if err != nil {
			return nil, fmt.Errorf("following redirect to %s: %w", currentURL, err)
		}
		resp = newResp
		redirectCount++
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading redirected response body: %w", err)
	}

	return &Response{
		Status:     resp.StatusCode,
		Headers:    resp.Header,
		Body:       string(body),
		URL:        currentURL,
		Redirected: redirectCount > 0,
	}, nil
}

func resolveRedirectURL(currentURL, location string) string {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		return location
	}
	if strings.HasPrefix(location, "/") {
		parsed, err := url.Parse(currentURL)
		if err != nil {
			return location
		}
		return fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, location)
	}
	// Relative path
	idx := strings.LastIndex(currentURL, "/")
	if idx >= 0 {
		return currentURL[:idx+1] + location
	}
	return location
}

// --- Session persistence ---

type sessionData struct {
	Cookies     map[string]string      `json:"cookies"`
	SessionInfo map[string]interface{} `json:"sessionInfo"`
}

// SessionCookie returns the JSESSIONID cookie value, if any.
func (c *Client) SessionCookie() string {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return ""
	}
	for _, ck := range c.jar.Cookies(parsed) {
		if ck.Name == "JSESSIONID" {
			return ck.Value
		}
	}
	return ""
}

// SaveSession persists cookies and session info to a JSON file.
func (c *Client) SaveSession(filePath string, sessionInfo map[string]interface{}) error {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}

	cookies := make(map[string]string)
	for _, ck := range c.jar.Cookies(parsed) {
		cookies[ck.Name] = ck.Value
	}

	data := sessionData{Cookies: cookies, SessionInfo: sessionInfo}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, raw, 0600)
}

// LoadSession restores cookies and session info from a JSON file.
func (c *Client) LoadSession(filePath string) (map[string]interface{}, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var data sessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	// Restore cookies into the jar
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	var httpCookies []*http.Cookie
	for k, v := range data.Cookies {
		httpCookies = append(httpCookies, &http.Cookie{Name: k, Value: v})
	}
	c.jar.SetCookies(parsed, httpCookies)

	return data.SessionInfo, nil
}

// ClearCookies removes all cookies by creating a fresh jar.
func (c *Client) ClearCookies() {
	jar, _ := cookiejar.New(nil)
	c.jar = jar
	c.httpClient.Jar = jar
}
