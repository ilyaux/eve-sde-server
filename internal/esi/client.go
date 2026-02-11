package esi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-retry"
)

const (
	ESIBaseURL = "https://esi.evetech.net/latest"
	UserAgent  = "EVE-SDE-Server/1.0 (https://github.com/yourusername/eve-sde-server)"
)

// Client handles ESI API requests
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new ESI client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: ESIBaseURL,
	}
}

// Get performs a GET request to ESI with retry logic
func (c *Client) Get(endpoint string) ([]byte, error) {
	url := c.baseURL + endpoint
	var body []byte

	// Retry configuration: exponential backoff with max 3 retries
	ctx := context.Background()
	backoff := retry.NewExponential(1 * time.Second)
	backoff = retry.WithMaxRetries(3, backoff)
	backoff = retry.WithMaxDuration(10*time.Second, backoff)

	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/json")

		log.Debug().Str("url", url).Msg("ESI request")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Network errors are retryable
			log.Warn().Err(err).Str("url", url).Msg("ESI request failed - retrying")
			return retry.RetryableError(fmt.Errorf("request failed: %w", err))
		}
		defer resp.Body.Close()

		// 5xx errors are retryable
		if resp.StatusCode >= 500 {
			log.Warn().
				Int("status", resp.StatusCode).
				Str("url", url).
				Msg("ESI server error - retrying")
			return retry.RetryableError(fmt.Errorf("ESI returned status %d", resp.StatusCode))
		}

		// 4xx errors are not retryable (client errors)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			respBody, _ := io.ReadAll(resp.Body)
			log.Warn().
				Int("status", resp.StatusCode).
				Str("url", url).
				Str("body", string(respBody)).
				Msg("ESI client error - not retrying")
			return fmt.Errorf("ESI returned status %d", resp.StatusCode)
		}

		// Success case
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return body, nil
}

// GetJSON performs a GET request and decodes JSON
func (c *Client) GetJSON(endpoint string, v interface{}) error {
	data, err := c.Get(endpoint)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// GetTypeInfo fetches type information from ESI
func (c *Client) GetTypeInfo(typeID int) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/universe/types/%d/", typeID)
	var result map[string]interface{}
	err := c.GetJSON(endpoint, &result)
	return result, err
}

// GetMarketPrices fetches market prices from ESI
func (c *Client) GetMarketPrices() ([]map[string]interface{}, error) {
	endpoint := "/markets/prices/"
	var result []map[string]interface{}
	err := c.GetJSON(endpoint, &result)
	return result, err
}

// GetMarketHistory fetches market history for a type in a region
func (c *Client) GetMarketHistory(regionID, typeID int) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/markets/%d/history/?type_id=%d", regionID, typeID)
	var result []map[string]interface{}
	err := c.GetJSON(endpoint, &result)
	return result, err
}
