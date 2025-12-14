package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type JWSTClient interface {
	Get(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error)
	Search(ctx context.Context, query string, page, perPage int) (map[string]interface{}, error)
}

type jwstClient struct {
	host   string
	apiKey string
	email  string
	client *http.Client
}

type JWSTConfig struct {
	Host   string
	APIKey string
	Email  string
}

func NewJWSTClient(config JWSTConfig) JWSTClient {
	return &jwstClient{
		host:   config.Host,
		apiKey: config.APIKey,
		email:  config.Email,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *jwstClient) Get(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error) {
	reqURL := fmt.Sprintf("%s/%s", c.host, path)

	// Добавляем параметры запроса
	if len(params) > 0 {
		query := url.Values{}
		for key, value := range params {
			query.Add(key, value)
		}
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("User-Agent", "Cosmos-Dashboard/1.0")
	req.Header.Set("Accept", "application/json")

	if c.email != "" {
		req.Header.Set("email", c.email)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWST API returned status %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	return data, nil
}

func (c *jwstClient) Search(ctx context.Context, query string, page, perPage int) (map[string]interface{}, error) {
	params := map[string]string{
		"q":       query,
		"page":    fmt.Sprintf("%d", page),
		"perPage": fmt.Sprintf("%d", perPage),
	}

	return c.Get(ctx, "search", params)
}
