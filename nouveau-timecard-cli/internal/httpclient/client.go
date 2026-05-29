// Package httpclient provides an HTTP client with cookie management,
// manual redirect control, and session persistence for the Nouveau Timecard server.
package httpclient

import (
	"crypto/tls"
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
	Status  int
	Headers http.Header
	Body    string
	URL     string
}

// Client is an HTTP client with cookie jar and redirect control.
type Client struct {
	baseURL    string
	httpClient *http.Client
	jar        *cookiejar.Jar
}

// New creates a new Client with the given base URL. When insecure is true,
// TLS certificate verification is skipped (useful for local dev servers).
func New(baseURL string, insecure bool) (*Client, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &Client{
		baseURL: baseURL,
		jar:     jar,
	}

	client.httpClient = &http.Client{
		Jar:       jar,
		Timeout:   30 * time.Second,
		Transport: transport,
		// Disable automatic redirect following — callers inspect 302 Location.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client, nil
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) resolveURL(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	return c.baseURL + strings.TrimPrefix(pathOrURL, "/")
}

// Get performs a GET and follows redirects manually (up to 10 hops).
func (c *Client) Get(pathOrURL string) (*Response, error) {
	currentURL := c.resolveURL(pathOrURL)

	for hops := 0; hops < 10; hops++ {
		resp, err := c.httpClient.Get(currentURL)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", currentURL, err)
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			resp.Body.Close()
			if loc == "" {
				return nil, fmt.Errorf("redirect with no Location from %s", currentURL)
			}
			currentURL = resolveRedirectURL(currentURL, loc)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}
		return &Response{Status: resp.StatusCode, Headers: resp.Header, Body: string(body), URL: currentURL}, nil
	}
	return nil, fmt.Errorf("too many redirects starting from %s", c.resolveURL(pathOrURL))
}

// PostForm performs a POST with url-encoded body. Does NOT follow redirects —
// returns the raw response so callers can inspect a 302 Location.
func (c *Client) PostForm(pathOrURL string, values url.Values) (*Response, error) {
	reqURL := c.resolveURL(pathOrURL)

	resp, err := c.httpClient.Post(reqURL, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return &Response{Status: resp.StatusCode, Headers: resp.Header, Body: string(body), URL: reqURL}, nil
}

func resolveRedirectURL(currentURL, location string) string {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		return location
	}
	parsed, err := url.Parse(currentURL)
	if err != nil {
		return location
	}
	if strings.HasPrefix(location, "/") {
		return fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, location)
	}
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

// SaveSession persists cookies and session info to a JSON file (0600).
func (c *Client) SaveSession(filePath string, sessionInfo map[string]interface{}) error {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	cookies := make(map[string]string)
	for _, ck := range c.jar.Cookies(parsed) {
		cookies[ck.Name] = ck.Value
	}
	raw, err := json.MarshalIndent(sessionData{Cookies: cookies, SessionInfo: sessionInfo}, "", "  ")
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
