package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type LakesetStatusResponse struct {
	OK                  bool    `json:"ok"`
	SupersetOnline      bool    `json:"superset_online"`
	DatabaseName        string  `json:"database_name"`
	Engine              string  `json:"engine"`
	WorkspaceID         string  `json:"workspace_id"`
	WorkspaceSlug       string  `json:"workspace_slug"`
	WorkspaceName       string  `json:"workspace_name"`
	Role                string  `json:"role"`
	UserEmail           string  `json:"user_email"`
	DatasetsCount       int     `json:"datasets_count"`
	SyncedDatasetsCount int     `json:"synced_datasets_count"`
	DashboardsCount     int     `json:"dashboards_count"`
	LastSyncedAt        *string `json:"last_synced_at,omitempty"`
}

type LakesetColumn struct {
	ColumnName  string  `json:"column_name"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
}

type LakesetDataset struct {
	ID              string          `json:"id"`
	CatalogName     string          `json:"catalog_name"`
	CatalogSlug     string          `json:"catalog_slug"`
	SchemaName      string          `json:"schema_name"`
	SchemaSlug      string          `json:"schema_slug"`
	Tier            string          `json:"tier,omitempty"`
	TableName       string          `json:"table_name"`
	FullName        string          `json:"full_name"`
	DisplayName     string          `json:"display_name"`
	SQLView         string          `json:"sql_view"`
	Description     *string         `json:"description,omitempty"`
	Format          string          `json:"format,omitempty"`
	S3PathPattern   string          `json:"s3_path_pattern,omitempty"`
	ColumnCount     int             `json:"column_count"`
	Columns         []LakesetColumn `json:"columns,omitempty"`
	Synced          bool            `json:"synced"`
	SupersetDataset *int            `json:"superset_dataset_id,omitempty"`
}

type LakesetDatasetsResponse struct {
	OK            bool             `json:"ok"`
	WorkspaceID   string           `json:"workspace_id"`
	WorkspaceSlug string           `json:"workspace_slug"`
	Role          string           `json:"role"`
	DatabaseName  string           `json:"database_name"`
	Total         int              `json:"total"`
	Datasets      []LakesetDataset `json:"datasets"`
}

type LakesetDatasetsFilter struct {
	Catalog string
	Schema  string
	Tier    string
	Search  string
}

type LakesetSyncRequest struct {
	CatalogSlug string `json:"catalog_slug,omitempty"`
	SchemaSlug  string `json:"schema_slug,omitempty"`
	Force       bool   `json:"force"`
}

type LakesetSyncResponse struct {
	OK                  bool             `json:"ok"`
	WorkspaceID         string           `json:"workspace_id"`
	WorkspaceSlug       string           `json:"workspace_slug"`
	DatabaseID          int              `json:"database_id"`
	DatabaseName        string           `json:"database_name"`
	Engine              string           `json:"engine"`
	SyncedDatasetsCount int              `json:"synced_datasets_count"`
	SyncedAt            string           `json:"synced_at"`
	Datasets            []LakesetDataset `json:"datasets"`
}

type LakesetGuestTokenRequest struct {
	Dataset     string `json:"dataset,omitempty"`
	DashboardID string `json:"dashboard_id,omitempty"`
	TTLSeconds  int    `json:"ttl_seconds,omitempty"`
}

type LakesetGuestTokenResponse struct {
	OK            bool                   `json:"ok"`
	Token         string                 `json:"token"`
	ExpiresIn     int                    `json:"expires_in"`
	ExpiresAt     string                 `json:"expires_at,omitempty"`
	WorkspaceID   string                 `json:"workspace_id"`
	WorkspaceSlug string                 `json:"workspace_slug"`
	UserEmail     string                 `json:"user_email"`
	Resources     []map[string]any       `json:"resources,omitempty"`
	RLS           []map[string]any       `json:"rls,omitempty"`
	Extra         map[string]interface{} `json:"-"`
}

type LakesetExploreRequest struct {
	Dataset   string `json:"dataset,omitempty"`
	SQL       string `json:"sql,omitempty"`
	ChartType string `json:"chart_type,omitempty"`
	Dimension string `json:"dimension,omitempty"`
	Metric    string `json:"metric,omitempty"`
	MetricAgg string `json:"metric_agg,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type LakesetExploreResponse struct {
	OK        bool           `json:"ok"`
	Dataset   string         `json:"dataset"`
	ChartType string         `json:"chart_type"`
	Dimension string         `json:"dimension"`
	Metric    string         `json:"metric"`
	SQL       string         `json:"sql"`
	Data      map[string]any `json:"data"`
}

func (c *Client) GetLakesetStatus() (*LakesetStatusResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/v1/lakeset/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lakeset status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lakeset status error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res LakesetStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode lakeset status: %w", err)
	}
	return &res, nil
}

func (c *Client) ListLakesetDatasets(filter LakesetDatasetsFilter) (*LakesetDatasetsResponse, error) {
	params := url.Values{}
	if filter.Catalog != "" {
		params.Set("catalog", filter.Catalog)
	}
	if filter.Schema != "" {
		params.Set("schema", filter.Schema)
	}
	if filter.Tier != "" {
		params.Set("tier", filter.Tier)
	}
	if filter.Search != "" {
		params.Set("search", filter.Search)
	}

	path := "/api/v1/lakeset/datasets"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list lakeset datasets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list lakeset datasets error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res LakesetDatasetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode lakeset datasets: %w", err)
	}
	return &res, nil
}

func (c *Client) SyncLakeset(reqPayload LakesetSyncRequest) (*LakesetSyncResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/lakeset/sync", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to sync lakeset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lakeset sync error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res LakesetSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode lakeset sync response: %w", err)
	}
	return &res, nil
}

func (c *Client) CreateLakesetGuestToken(reqPayload LakesetGuestTokenRequest) (*LakesetGuestTokenResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/lakeset/guest-token", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to mint lakeset guest token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lakeset guest token error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res LakesetGuestTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode lakeset guest token response: %w", err)
	}
	return &res, nil
}

func (c *Client) ExploreLakesetData(reqPayload LakesetExploreRequest) (*LakesetExploreResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/lakeset/explore-data", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute lakeset explore query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lakeset explore error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res LakesetExploreResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode lakeset explore response: %w", err)
	}
	return &res, nil
}
