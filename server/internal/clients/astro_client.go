package clients

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type AstroClient interface {
	GetEvents(ctx context.Context, lat, lon float64, days int) (map[string]interface{}, error)
	GetBodies(ctx context.Context) (map[string]interface{}, error)
	GetMoonPhase(ctx context.Context, date time.Time) (map[string]interface{}, error)
}

type astroClient struct {
	appID   string
	secret  string
	baseURL string
	client  *http.Client
}

type AstroConfig struct {
	AppID   string
	Secret  string
	BaseURL string
}

func NewAstroClient(config AstroConfig) AstroClient {
	return &astroClient{
		appID:   config.AppID,
		secret:  config.Secret,
		baseURL: config.BaseURL,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (c *astroClient) GetEvents(ctx context.Context, lat, lon float64, days int) (map[string]interface{}, error) {
	from := time.Now().UTC().Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")

	reqURL := fmt.Sprintf("%s/bodies/events", c.baseURL)

	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%f", lat))
	params.Add("longitude", fmt.Sprintf("%f", lon))
	params.Add("from", from)
	params.Add("to", to)

	reqURL += "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Базовая авторизация
	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.secret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("AstronomyAPI returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}

func (c *astroClient) GetBodies(ctx context.Context) (map[string]interface{}, error) {
	reqURL := fmt.Sprintf("%s/bodies", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.secret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("AstronomyAPI returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}

func (c *astroClient) GetMoonPhase(ctx context.Context, date time.Time) (map[string]interface{}, error) {
	dateStr := date.Format("2006-01-02")
	reqURL := fmt.Sprintf("%s/moon-phase?date=%s", c.baseURL, dateStr)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.appID + ":" + c.secret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("AstronomyAPI returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}
