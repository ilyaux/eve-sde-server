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

// Category represents an EVE inventory category.
type Category struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

// Group represents an EVE inventory group.
type Group struct {
	GroupID    int    `json:"group_id"`
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

// SearchResult represents search results
type SearchResult struct {
	Data []Item `json:"data"`
	Meta struct {
		Count  int `json:"count"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// ListResult represents a paginated item list response.
type ListResult struct {
	Data []Item `json:"data"`
	Meta struct {
		Count  int `json:"count"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// CategoryResult represents a paginated category list response.
type CategoryResult struct {
	Data []Category `json:"data"`
	Meta struct {
		Count  int `json:"count"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// GroupResult represents a paginated group list response.
type GroupResult struct {
	Data []Group `json:"data"`
	Meta struct {
		Count  int `json:"count"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// HealthStatus represents the public liveness response.
type HealthStatus struct {
	Status        string `json:"status"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Timestamp     string `json:"timestamp"`
}

// ServerVersion represents server build metadata.
type ServerVersion struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// ReadinessStatus represents the public readiness response.
type ReadinessStatus struct {
	Status              string            `json:"status"`
	Checks              map[string]string `json:"checks"`
	ItemsCount          int               `json:"items_count"`
	LatestSDEVersion    *string           `json:"latest_sde_version"`
	LatestSDEImportedAt *string           `json:"latest_sde_imported_at"`
	Version             ServerVersion     `json:"version"`
}

// ItemChange represents a change between two SDE versions.
type ItemChange struct {
	TypeID       int    `json:"type_id"`
	Name         string `json:"name"`
	ChangeType   string `json:"change_type"`
	OldValue     string `json:"old_value,omitempty"`
	NewValue     string `json:"new_value,omitempty"`
	FieldChanged string `json:"field_changed,omitempty"`
}

// DiffSummary summarizes SDE version changes by type.
type DiffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
}

// DiffResponse represents a response from /api/v1/diff.
type DiffResponse struct {
	FromVersion string       `json:"from_version"`
	ToVersion   string       `json:"to_version"`
	Changes     []ItemChange `json:"changes"`
	Summary     DiffSummary  `json:"summary"`
	Note        string       `json:"note,omitempty"`
}

// SDEVersion represents one imported SDE version in the changelog.
type SDEVersion struct {
	Version    string `json:"version"`
	ImportedAt string `json:"imported_at"`
	ItemCount  int    `json:"item_count"`
}

// ChangelogResponse represents recent SDE import history.
type ChangelogResponse struct {
	Versions []SDEVersion `json:"versions"`
	Count    int          `json:"count"`
	Note     string       `json:"note,omitempty"`
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
	result, err := c.ListItemsWithMeta(limit, offset)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ListItemsWithMeta retrieves items and pagination metadata.
func (c *Client) ListItemsWithMeta(limit, offset int) (*ListResult, error) {
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

	var result ListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
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

// GetCategory retrieves a category by ID.
func (c *Client) GetCategory(categoryID int) (*Category, error) {
	path := fmt.Sprintf("/api/v1/categories/%d", categoryID)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var category Category
	if err := json.Unmarshal(data, &category); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &category, nil
}

// ListCategories retrieves categories and pagination metadata.
func (c *Client) ListCategories(limit, offset int) (*CategoryResult, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	data, err := c.doRequest("GET", "/api/v1/categories", query)
	if err != nil {
		return nil, err
	}

	var result CategoryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetGroup retrieves a group by ID.
func (c *Client) GetGroup(groupID int) (*Group, error) {
	path := fmt.Sprintf("/api/v1/groups/%d", groupID)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var group Group
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &group, nil
}

// ListGroups retrieves groups and pagination metadata.
// Pass categoryID <= 0 to list groups across all categories.
func (c *Client) ListGroups(categoryID, limit, offset int) (*GroupResult, error) {
	query := url.Values{}
	if categoryID > 0 {
		query.Set("category_id", fmt.Sprintf("%d", categoryID))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	data, err := c.doRequest("GET", "/api/v1/groups", query)
	if err != nil {
		return nil, err
	}

	var result GroupResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Diff compares two SDE version identifiers.
func (c *Client) Diff(fromVersion, toVersion string) (*DiffResponse, error) {
	query := url.Values{}
	query.Set("from", fromVersion)
	query.Set("to", toVersion)

	data, err := c.doRequest("GET", "/api/v1/diff", query)
	if err != nil {
		return nil, err
	}

	var result DiffResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Changelog returns recent SDE import history.
func (c *Client) Changelog() (*ChangelogResponse, error) {
	data, err := c.doRequest("GET", "/api/v1/changelog", nil)
	if err != nil {
		return nil, err
	}

	var result ChangelogResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ESITypeInfo returns proxied ESI type information.
func (c *Client) ESITypeInfo(typeID int) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/esi/types/%d", typeID)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ESIMarketPrices returns proxied ESI market prices.
func (c *Client) ESIMarketPrices() ([]map[string]interface{}, error) {
	data, err := c.doRequest("GET", "/api/esi/markets/prices", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ESIMarketHistory returns proxied ESI market history for a region and type.
func (c *Client) ESIMarketHistory(regionID, typeID int) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/api/esi/markets/%d/history/%d", regionID, typeID)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// HealthStatus returns liveness details from the server.
func (c *Client) HealthStatus() (*HealthStatus, error) {
	data, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	var result HealthStatus
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Health checks whether the server liveness endpoint reports OK.
func (c *Client) Health() (bool, error) {
	status, err := c.HealthStatus()
	if err != nil {
		return false, err
	}

	return status.Status == "OK", nil
}

// Ready returns readiness details, including dependency checks and SDE metadata.
func (c *Client) Ready() (*ReadinessStatus, error) {
	data, err := c.doRequest("GET", "/ready", nil)
	if err != nil {
		return nil, err
	}

	var result ReadinessStatus
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Version returns server build metadata.
func (c *Client) Version() (*ServerVersion, error) {
	data, err := c.doRequest("GET", "/version", nil)
	if err != nil {
		return nil, err
	}

	var result ServerVersion
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
