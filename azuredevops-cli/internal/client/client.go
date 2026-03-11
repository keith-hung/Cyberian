// Package client implements the Azure DevOps Server REST API HTTP client.
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

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// Client talks to the Azure DevOps Server REST API using IIS Basic Auth.
type Client struct {
	baseURL    string
	collection string
	username   string
	password   string
	apiVersion string
	http       *http.Client
	repoCache  map[string]string // "project/repoName" -> repoID
}

// New creates a Client. When insecure is true, TLS certificate verification is skipped.
func New(baseURL, collection, username, password, apiVersion string, insecure bool) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    baseURL,
		collection: collection,
		username:   username,
		password:   password,
		apiVersion: apiVersion,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
				},
			},
		},
		repoCache: make(map[string]string),
	}
}

// doRequest executes an HTTP request and returns the response body.
func (c *Client) doRequest(method, path string, body io.Reader) ([]byte, int, error) {
	return c.doRequestWithContentType(method, path, "application/json", body)
}

// doRequestWithContentType executes an HTTP request with a custom Content-Type.
func (c *Client) doRequestWithContentType(method, path, contentType string, body io.Reader) ([]byte, int, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s/%s", c.baseURL, c.collection, path))
	if err != nil {
		return nil, 0, fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("api-version", c.apiVersion)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	if method == "POST" || method == "PATCH" || method == "PUT" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// checkResponse validates the HTTP status code and returns a descriptive error if needed.
func checkResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	apiMsg := extractErrorMessage(body)
	switch statusCode {
	case 401:
		return fmt.Errorf("authentication failed: check AZDO_USERNAME and AZDO_PASSWORD (HTTP 401)")
	case 403:
		return fmt.Errorf("forbidden: insufficient permissions (HTTP 403): %s", apiMsg)
	case 404:
		return fmt.Errorf("not found: verify project, repository, and PR ID (HTTP 404): %s", apiMsg)
	case 409:
		return fmt.Errorf("conflict: %s", apiMsg)
	default:
		return fmt.Errorf("API error (HTTP %d): %s", statusCode, apiMsg)
	}
}

// extractErrorMessage parses the Azure DevOps error response format.
func extractErrorMessage(body []byte) string {
	var apiErr types.APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return apiErr.Message
	}
	if len(body) > 200 {
		return string(body[:200]) + "..."
	}
	return string(body)
}

// ListProjects returns all projects in the collection.
func (c *Client) ListProjects() (*types.ProjectListResponse, error) {
	body, status, err := c.doRequest("GET", "_apis/projects", nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.ProjectListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ListRepos returns all Git repositories in a project.
func (c *Client) ListRepos(project string) (*types.RepoListResponse, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories", url.PathEscape(project))
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.RepoListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ResolveRepoID resolves a repo name (or ID) to a repo GUID.
func (c *Client) ResolveRepoID(project, nameOrID string) (string, error) {
	cacheKey := strings.ToLower(project + "/" + nameOrID)
	if id, ok := c.repoCache[cacheKey]; ok {
		return id, nil
	}

	repos, err := c.ListRepos(project)
	if err != nil {
		return "", fmt.Errorf("list repos for resolution: %w", err)
	}

	var names []string
	for _, r := range repos.Value {
		c.repoCache[strings.ToLower(project+"/"+r.Name)] = r.ID
		c.repoCache[strings.ToLower(project+"/"+r.ID)] = r.ID
		names = append(names, r.Name)
		if strings.EqualFold(r.Name, nameOrID) || r.ID == nameOrID {
			return r.ID, nil
		}
	}

	return "", fmt.Errorf("repository %q not found in project %q; available: %s", nameOrID, project, strings.Join(names, ", "))
}

// ListPullRequests returns pull requests for a repository.
func (c *Client) ListPullRequests(project, repoID, status string) (*types.PRListResponse, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests",
		url.PathEscape(project), url.PathEscape(repoID))
	if status != "" {
		path += "?searchCriteria.status=" + url.QueryEscape(status)
	}

	body, statusCode, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(statusCode, body); err != nil {
		return nil, err
	}

	var result types.PRListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// GetPullRequest returns a single pull request.
func (c *Client) GetPullRequest(project, repoID string, prID int) (*types.APIPR, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d",
		url.PathEscape(project), url.PathEscape(repoID), prID)
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.APIPR
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(project, repoID string, pr *types.PRCreateBody) (*types.APIPR, error) {
	data, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}

	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests",
		url.PathEscape(project), url.PathEscape(repoID))
	body, status, err := c.doRequest("POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.APIPR
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// UpdatePullRequest updates an existing pull request.
func (c *Client) UpdatePullRequest(project, repoID string, prID int, update *types.PRUpdateBody) (*types.APIPR, error) {
	data, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}

	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d",
		url.PathEscape(project), url.PathEscape(repoID), prID)
	body, status, err := c.doRequest("PATCH", path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.APIPR
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// VotePullRequest casts a vote on a pull request.
func (c *Client) VotePullRequest(project, repoID string, prID int, reviewerID string, vote int) error {
	data, err := json.Marshal(types.VoteBody{Vote: vote})
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/reviewers/%s",
		url.PathEscape(project), url.PathEscape(repoID), prID, url.PathEscape(reviewerID))
	body, status, err := c.doRequest("PUT", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return checkResponse(status, body)
}

// CreateThread adds a comment thread to a pull request.
func (c *Client) CreateThread(project, repoID string, prID int, comment string) error {
	thread := types.ThreadBody{
		Comments: []types.ThreadComment{
			{
				ParentCommentID: 0,
				Content:         comment,
				CommentType:     "text",
			},
		},
		Status: 1,
	}
	data, err := json.Marshal(thread)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/threads",
		url.PathEscape(project), url.PathEscape(repoID), prID)
	body, status, err := c.doRequest("POST", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return checkResponse(status, body)
}

// ListThreads returns all comment threads for a pull request.
func (c *Client) ListThreads(project, repoID string, prID int) (*types.ThreadListResponse, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/threads",
		url.PathEscape(project), url.PathEscape(repoID), prID)
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.ThreadListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ListReviewers returns the reviewers of a pull request.
func (c *Client) ListReviewers(project, repoID string, prID int) (*types.ReviewerListResponse, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/reviewers",
		url.PathEscape(project), url.PathEscape(repoID), prID)
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.ReviewerListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// AddReviewer adds a reviewer to a pull request.
func (c *Client) AddReviewer(project, repoID string, prID int, reviewerID string) error {
	data, err := json.Marshal(types.ReviewerAddBody{ID: reviewerID})
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}

	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/reviewers/%s",
		url.PathEscape(project), url.PathEscape(repoID), prID, url.PathEscape(reviewerID))
	body, status, err := c.doRequest("PUT", path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return checkResponse(status, body)
}

// GetCurrentUserID returns the authenticated user's ID.
func (c *Client) GetCurrentUserID() (string, error) {
	body, status, err := c.doRequest("GET", "_apis/connectionData", nil)
	if err != nil {
		return "", err
	}
	if err := checkResponse(status, body); err != nil {
		return "", err
	}

	var result types.APIConnectionData
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode connectionData: %w", err)
	}
	if result.AuthenticatedUser.ID == "" {
		return "", fmt.Errorf("could not determine authenticated user ID")
	}
	return result.AuthenticatedUser.ID, nil
}

// ListMyPullRequests returns pull requests across all projects filtered by a search criterion.
// searchParam should be "searchCriteria.creatorId" or "searchCriteria.reviewerId".
func (c *Client) ListMyPullRequests(status, searchParam, userID string) (*types.PRListResponse, error) {
	path := fmt.Sprintf("_apis/git/pullrequests?searchCriteria.status=%s&%s=%s",
		url.QueryEscape(status), searchParam, url.QueryEscape(userID))

	body, statusCode, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(statusCode, body); err != nil {
		return nil, err
	}

	var result types.PRListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// UploadAttachment uploads a file as a PR attachment and returns the attachment metadata.
func (c *Client) UploadAttachment(project, repoID string, prID int, filename string, data io.Reader) (*types.APIAttachment, error) {
	path := fmt.Sprintf("%s/_apis/git/repositories/%s/pullrequests/%d/attachments/%s",
		url.PathEscape(project), url.PathEscape(repoID), prID, url.PathEscape(filename))
	body, status, err := c.doRequestWithContentType("POST", path, "application/octet-stream", data)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(status, body); err != nil {
		return nil, err
	}

	var result types.APIAttachment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ResolveIdentityID resolves a display name or domain\username to a GUID via the Identity API.
func (c *Client) ResolveIdentityID(searchValue string) (string, error) {
	path := "_apis/identities?searchFilter=General&filterValue=" + url.QueryEscape(searchValue)
	body, status, err := c.doRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	if err := checkResponse(status, body); err != nil {
		return "", fmt.Errorf("identity search: %w", err)
	}

	var result types.APIIdentitySearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode identity response: %w", err)
	}
	if result.Count == 0 || len(result.Value) == 0 {
		return "", fmt.Errorf("no identity found for %q", searchValue)
	}
	return result.Value[0].ID, nil
}

