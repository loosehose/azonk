// Package graph provides a client for Microsoft Graph API operations.
// It handles HTTP requests, pagination, and rate limiting for all Graph endpoints.
package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/loosehose/azonk/internal/config"
)

// Client is the Microsoft Graph API client.
// It manages authentication headers, request formatting, and response parsing.
type Client struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

// NewClient creates a new Graph API client with the given access token.
func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		baseURL:     config.GraphBaseURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// =============================================================================
// Core HTTP Methods
// =============================================================================

// Get performs a GET request to the specified Graph API endpoint.
// The endpoint should be a path like "/users" (base URL is prepended).
func (c *Client) Get(endpoint string) ([]byte, error) {
	return c.request("GET", endpoint, nil)
}

// Post performs a POST request with a JSON payload.
func (c *Client) Post(endpoint string, payload []byte) ([]byte, error) {
	return c.request("POST", endpoint, payload)
}

// request is the internal method that handles all HTTP requests.
func (c *Client) request(method, endpoint string, body []byte) ([]byte, error) {
	url := c.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, truncateError(respBody))
	}

	return respBody, nil
}

// =============================================================================
// Pagination Support
// =============================================================================

// GetAllPages retrieves all pages of results from a paginated endpoint.
// Set maxResults to 0 for unlimited results.
func (c *Client) GetAllPages(endpoint string, maxResults int) ([]json.RawMessage, error) {
	var allResults []json.RawMessage
	nextLink := c.baseURL + endpoint

	for nextLink != "" {
		// Check if we've reached the limit
		if maxResults > 0 && len(allResults) >= maxResults {
			break
		}

		results, next, err := c.getPage(nextLink)
		if err != nil {
			return allResults, err
		}

		allResults = append(allResults, results...)
		nextLink = next

		// Rate limiting to avoid throttling
		if nextLink != "" {
			time.Sleep(config.RateLimitDelay)
		}
	}

	return allResults, nil
}

// getPage retrieves a single page of results.
func (c *Client) getPage(url string) ([]json.RawMessage, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, truncateError(body))
	}

	var result struct {
		Value    []json.RawMessage `json:"value"`
		NextLink string            `json:"@odata.nextLink"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("parse response: %w", err)
	}

	return result.Value, result.NextLink, nil
}

// =============================================================================
// Helpers
// =============================================================================

// truncateError truncates error messages for cleaner output.
func truncateError(body []byte) string {
	const maxLen = 200
	s := string(body)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
