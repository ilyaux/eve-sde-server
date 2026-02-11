// Package evesde provides a Go client for the EVE SDE Server API
package evesde

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the EVE SDE Server API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new EVE SDE Server API client
func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Item represents an EVE item
type Item struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Volume      float64 `json:"volume"`
	GroupID     int     `json:"group_id,omitempty"`
	CategoryID  int     `json:"category_id,omitempty"`
}

// SearchResult represents search results
type SearchResult struct {
	Data []Item `json:"data"`
	Meta struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, query url.Values) ([]byte, error) {
	url := c.BaseURL + path
	if query != nil {
		url += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "eve-sde-go-sdk/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetItem retrieves an item by its type ID
func (c *Client) GetItem(typeID int) (*Item, error) {
	path := fmt.Sprintf("/api/v1/items/%d", typeID)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &item, nil
}

// ListItems retrieves a list of items with pagination
func (c *Client) ListItems(limit, offset int) ([]Item, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	data, err := c.doRequest("GET", "/api/v1/items", query)
	if err != nil {
		return nil, err
	}

	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return items, nil
}

// Search searches for items by name or description
func (c *Client) Search(query string, limit int) (*SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	data, err := c.doRequest("GET", "/api/v1/search", params)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Health checks the server health
func (c *Client) Health() (bool, error) {
	data, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, err
	}

	return result["status"] == "OK", nil
}
