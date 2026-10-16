package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

type Client struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
}

func NewClient(ctx config.Context) *Client {
	timeout := time.Duration(ctx.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		Endpoint: ctx.Endpoint,
		Token:    ctx.Token,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(data)
	}

	url := fmt.Sprintf("%s%s", c.Endpoint, path)
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "angler-cli")
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	return req, nil
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Services  map[string]string `json:"services,omitempty"`
	Timestamp string            `json:"timestamp,omitempty"`
}

func (c *Client) Health() (*HealthResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// Fallback check to root or /health
		reqRoot, err2 := c.newRequest(http.MethodGet, "/", nil)
		if err2 == nil {
			respRoot, err3 := c.HTTPClient.Do(reqRoot)
			if err3 == nil {
				defer respRoot.Body.Close()
				return &HealthResponse{
					Status:  "online",
					Version: "reachable",
				}, nil
			}
		}
		return nil, fmt.Errorf("failed to reach server at %s: %w", c.Endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &HealthResponse{
			Status:  fmt.Sprintf("HTTP %d", resp.StatusCode),
			Version: resp.Status,
		}, nil
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return &HealthResponse{
			Status:  "online",
			Version: "active",
		}, nil
	}

	return &health, nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/auth/login", LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	return &loginResp, nil
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Columns  []string        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int             `json:"row_count"`
	Duration string          `json:"duration,omitempty"`
}

func (c *Client) Query(sql string) (*QueryResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/query", QueryRequest{
		Query: sql,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse query results: %w", err)
	}

	return &queryResp, nil
}

type AIQueryRequest struct {
	Prompt string `json:"prompt"`
}

type AIQueryResponse struct {
	Answer   string   `json:"answer"`
	SQLQuery string   `json:"sql_query,omitempty"`
	Sources  []string `json:"sources,omitempty"`
}

func (c *Client) AIQuery(prompt string) (*AIQueryResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/ai/query", AIQueryRequest{
		Prompt: prompt,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI query error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var aiResp AIQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &aiResp, nil
}

type CatalogItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Rows        int64  `json:"rows,omitempty"`
	Size        string `json:"size,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
}

func (c *Client) ListCatalog() ([]CatalogItem, error) {
	req, err := c.newRequest(http.MethodGet, "/api/catalog", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("catalog error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var items []CatalogItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to parse catalog response: %w", err)
	}

	return items, nil
}
