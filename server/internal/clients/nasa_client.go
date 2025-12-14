package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type NASAClient interface {
	FetchOSDR(ctx context.Context) ([]map[string]interface{}, error)
	FetchAPOD(ctx context.Context, date string) (map[string]interface{}, error)
	FetchNEOFeed(ctx context.Context, days int) (map[string]interface{}, error)
	FetchDONKI(ctx context.Context, eventType string, days int) ([]map[string]interface{}, error)
}

type nasaClient struct {
	apiKey   string
	osdrURL  string
	apodURL  string
	neoURL   string
	donkiURL string
	client   *http.Client
}

type NASAConfig struct {
	APIKey   string
	OSDRURL  string
	APODURL  string
	NEOURL   string
	DONKIURL string
}

func NewNASAClient(config NASAConfig) NASAClient {
	return &nasaClient{
		apiKey:   config.APIKey,
		osdrURL:  config.OSDRURL,
		apodURL:  config.APODURL,
		neoURL:   config.NEOURL,
		donkiURL: "https://api.nasa.gov/DONKI",
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

func (c *nasaClient) FetchOSDR(ctx context.Context) ([]map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.osdrURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	if c.apiKey != "" {
		q := req.URL.Query()
		q.Add("api_key", c.apiKey)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	// Извлекаем items/results из ответа
	var items []map[string]interface{}
	if data, ok := result["items"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				items = append(items, itemMap)
			}
		}
	} else if data, ok := result["results"].([]interface{}); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				items = append(items, itemMap)
			}
		}
	} else {
		// Если структура другая, возвращаем весь объект как единственный элемент
		items = append(items, result)
	}

	return items, nil
}

func (c *nasaClient) FetchAPOD(ctx context.Context, date string) (map[string]interface{}, error) {
	reqURL := c.apodURL

	// Добавляем параметры
	params := url.Values{}
	params.Add("thumbs", "true")
	if date != "" {
		params.Add("date", date)
	}
	if c.apiKey != "" {
		params.Add("api_key", c.apiKey)
	}

	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("APOD API returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}

func (c *nasaClient) FetchNEOFeed(ctx context.Context, days int) (map[string]interface{}, error) {
	if days < 1 || days > 7 {
		days = 7
	}

	endDate := time.Now().UTC().Format("2006-01-02")
	startDate := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	reqURL := c.neoURL
	params := url.Values{}
	params.Add("start_date", startDate)
	params.Add("end_date", endDate)
	if c.apiKey != "" {
		params.Add("api_key", c.apiKey)
	}

	reqURL += "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NEO API returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}

func (c *nasaClient) FetchDONKI(ctx context.Context, eventType string, days int) ([]map[string]interface{}, error) {
	if days < 1 || days > 30 {
		days = 5
	}

	endDate := time.Now().UTC().Format("2006-01-02")
	startDate := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	reqURL := fmt.Sprintf("%s/%s", c.donkiURL, eventType)
	params := url.Values{}
	params.Add("startDate", startDate)
	params.Add("endDate", endDate)
	if c.apiKey != "" {
		params.Add("api_key", c.apiKey)
	}

	reqURL += "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DONKI API returned status %d", resp.StatusCode)
	}

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}
