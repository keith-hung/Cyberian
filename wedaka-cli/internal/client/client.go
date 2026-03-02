// Package client implements the WeDaka API HTTP client.
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/keith-hung/wedaka-cli/internal/types"
)

// Client talks to the WeDaka API.
type Client struct {
	baseURL  string
	deviceID string
	http     *http.Client
}

// New creates a Client. baseURL must include the scheme (https://...).
func New(baseURL, deviceID string) *Client {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		baseURL:  baseURL,
		deviceID: deviceID,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

// GetDateType checks whether a date is a work day.
func (c *Client) GetDateType(empID, date string) (*types.DateTypeResponse, error) {
	u, err := url.Parse(c.baseURL + "worktime/GetDateType/")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("empID", empID)
	q.Set("date", date)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result types.DateTypeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(body))
	}
	return &result, nil
}

// InsertTimeLog submits a clock-in or clock-out record.
func (c *Client) InsertTimeLog(payload *types.InsertTimeLogPayload) (*types.TimeLogResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"worktime/InsertTimeLog", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result types.TimeLogResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(body))
	}
	return &result, nil
}

// SearchTimelog queries time log records for a given month/year.
func (c *Client) SearchTimelog(username string, month, year int) (*types.SearchTimelogResponse, error) {
	u, err := url.Parse(c.baseURL + "worktime/SearchTimelog/")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("username", username)
	q.Set("month", fmt.Sprintf("%d", month))
	q.Set("year", fmt.Sprintf("%d", year))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result types.SearchTimelogResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(body))
	}
	return &result, nil
}

func (c *Client) setHeaders(req *http.Request) {
	if c.deviceID != "" {
		req.Header.Set("X-UUID", c.deviceID)
	}
}
