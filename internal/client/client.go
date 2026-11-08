package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

type Client struct {
	Endpoint    string
	Token       string
	Workspace   string
	WorkspaceID string
	HTTPClient  *http.Client
}

func NewClient(ctx config.Context) *Client {
	timeout := time.Duration(ctx.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		Endpoint:    ctx.Endpoint,
		Token:       ctx.Token,
		Workspace:   ctx.Workspace,
		WorkspaceID: ctx.WorkspaceID,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
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

	wsSlug := c.Workspace
	wsID := c.WorkspaceID

	if wsID == "" && isUUID(wsSlug) {
		wsID = wsSlug
	}

	if wsSlug != "" {
		req.Header.Set("X-Workspace-Slug", wsSlug)
	} else {
		req.Header.Set("X-Workspace-Slug", "default")
	}

	if wsID != "" {
		req.Header.Set("X-Workspace-Id", wsID)
	}

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

// -----------------------------------------------------------------------------
// Lake Asset Bundles (LABs) API Client Models & Methods
// -----------------------------------------------------------------------------

type BundleValidateRequest struct {
	ManifestContent   string                 `json:"manifest_content"`
	ManifestFormat    string                 `json:"manifest_format"`
	Target            string                 `json:"target,omitempty"`
	VariableOverrides map[string]interface{} `json:"variable_overrides,omitempty"`
	WorkspaceSlug     string                 `json:"workspace_slug,omitempty"`
}

type BundleValidationDiff struct {
	ToCreate  []interface{} `json:"to_create"`
	ToUpdate  []interface{} `json:"to_update"`
	ToDelete  []interface{} `json:"to_delete"`
	Unchanged []interface{} `json:"unchanged"`
}

type BundleValidationResponse struct {
	Valid          bool                  `json:"valid"`
	BundleName     string                `json:"bundle_name"`
	Target         string                `json:"target"`
	Diff           *BundleValidationDiff `json:"diff,omitempty"`
	Errors         []string              `json:"errors,omitempty"`
	Warnings       []string              `json:"warnings,omitempty"`
	DagValidated   bool                  `json:"dag_validated"`
	ResourcesCount map[string]int        `json:"resources_count,omitempty"`
}

type BundleDeployRequest struct {
	ManifestContent   string                 `json:"manifest_content"`
	ManifestFormat    string                 `json:"manifest_format"`
	Target            string                 `json:"target,omitempty"`
	VariableOverrides map[string]interface{} `json:"variable_overrides,omitempty"`
	CommitHash        string                 `json:"commit_hash,omitempty"`
	Files             map[string]string      `json:"files,omitempty"`
	DryRun            bool                   `json:"dry_run"`
	WorkspaceSlug     string                 `json:"workspace_slug,omitempty"`
}

type BundleDeployResponse struct {
	ID                  string                `json:"id"`
	BundleID            string                `json:"bundle_id"`
	BundleName          string                `json:"bundle_name"`
	Target              string                `json:"target"`
	Status              string                `json:"status"`
	Version             string                `json:"version"`
	Message             string                `json:"message"`
	DryRun              bool                  `json:"dry_run"`
	ResourcesMap        map[string][]string   `json:"resources_map,omitempty"`
	ProvisionedCounts   map[string]int        `json:"provisioned_counts,omitempty"`
	Diff                *BundleValidationDiff `json:"diff,omitempty"`
	ExecutionDurationMs float64               `json:"execution_duration_ms,omitempty"`
}

type BundleListItem struct {
	ID              string                 `json:"id"`
	BundleName      string                 `json:"bundle_name"`
	Target          string                 `json:"target"`
	Status          string                 `json:"status"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description,omitempty"`
	GitCommitSHA    string                 `json:"git_commit_sha,omitempty"`
	LastDeployedAt  string                 `json:"last_deployed_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	ResourcesMap    map[string]interface{} `json:"resources_map,omitempty"`
	CatalogsCount   int                    `json:"catalogs_count"`
	TablesCount     int                    `json:"tables_count"`
	PipelinesCount  int                    `json:"pipelines_count"`
	DashboardsCount int                    `json:"dashboards_count"`
}

type BundleListResponse struct {
	Bundles []BundleListItem `json:"bundles"`
	Total   int              `json:"total"`
}

type BundleDetailResponse struct {
	ID             string                 `json:"id"`
	WorkspaceID    string                 `json:"workspace_id"`
	BundleName     string                 `json:"bundle_name"`
	Target         string                 `json:"target"`
	Status         string                 `json:"status"`
	Version        string                 `json:"version"`
	Description    string                 `json:"description,omitempty"`
	GitCommitSHA   string                 `json:"git_commit_sha,omitempty"`
	ManifestRaw    string                 `json:"manifest_raw"`
	ResourcesMap   map[string]interface{} `json:"resources_map,omitempty"`
	Drift          map[string]interface{} `json:"drift,omitempty"`
	LastDeployedAt string                 `json:"last_deployed_at,omitempty"`
	CreatedAt      string                 `json:"created_at"`
}

type BundleRunRequest struct {
	PipelineName string `json:"pipeline_name,omitempty"`
}

type BundleRunResponse struct {
	BundleID     string `json:"bundle_id"`
	BundleName   string `json:"bundle_name"`
	PipelineName string `json:"pipeline_name"`
	DAGID        string `json:"dag_id"`
	DAGRunID     string `json:"dag_run_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

type BundleDestroyResponse struct {
	BundleID string `json:"bundle_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

func (c *Client) ValidateBundle(req *BundleValidateRequest) (*BundleValidationResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/bundles/validate", req)
	if err != nil {
		return nil, err
	}
	if req.WorkspaceSlug != "" {
		httpReq.Header.Set("X-Workspace-Slug", req.WorkspaceSlug)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bundle validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bundle validation failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res BundleValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse validation response: %w", err)
	}
	return &res, nil
}

func (c *Client) DeployBundle(req *BundleDeployRequest) (*BundleDeployResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/bundles/deploy", req)
	if err != nil {
		return nil, err
	}
	if req.WorkspaceSlug != "" {
		httpReq.Header.Set("X-Workspace-Slug", req.WorkspaceSlug)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bundle deploy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bundle deploy failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res BundleDeployResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse deploy response: %w", err)
	}
	return &res, nil
}

func (c *Client) ListBundles(target, status string) (*BundleListResponse, error) {
	params := url.Values{}
	if target != "" && target != "all" {
		params.Set("target", target)
	}
	if status != "" && status != "all" {
		params.Set("status", status)
	}

	path := "/api/bundles"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("list bundles request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list bundles (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res BundleListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse bundle list response: %w", err)
	}
	return &res, nil
}

func (c *Client) GetBundle(id string) (*BundleDetailResponse, error) {
	httpReq, err := c.newRequest(http.MethodGet, fmt.Sprintf("/api/bundles/%s?include_drift=true", url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("get bundle request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get bundle %s (HTTP %d): %s", id, resp.StatusCode, string(body))
	}

	var res BundleDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse bundle detail response: %w", err)
	}
	return &res, nil
}

func (c *Client) RunBundle(id string, pipelineName string) (*BundleRunResponse, error) {
	path := fmt.Sprintf("/api/bundles/%s/run", url.PathEscape(id))
	if pipelineName != "" {
		path += fmt.Sprintf("?pipeline_name=%s", url.QueryEscape(pipelineName))
	}

	httpReq, err := c.newRequest(http.MethodPost, path, BundleRunRequest{
		PipelineName: pipelineName,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("run bundle pipeline failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("run bundle pipeline failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res BundleRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse bundle run response: %w", err)
	}
	return &res, nil
}

func (c *Client) DestroyBundle(id string, cascade, purgeStorage bool) (*BundleDestroyResponse, error) {
	path := fmt.Sprintf("/api/bundles/%s?cascade=%t&purge_storage=%t", url.PathEscape(id), cascade, purgeStorage)

	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("destroy bundle failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("destroy bundle failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res BundleDestroyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse destroy bundle response: %w", err)
	}
	return &res, nil
}

// -----------------------------------------------------------------------------
// Cluster Status & Topology API Client Models & Methods
// -----------------------------------------------------------------------------

type ServiceHealthDetail struct {
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"display_name"`
	Status            string                 `json:"status"`
	Category          string                 `json:"category"`
	Port              *int                   `json:"port,omitempty"`
	PodName           *string                `json:"pod_name,omitempty"`
	NodeName          *string                `json:"node_name,omitempty"`
	Restarts          int                    `json:"restarts"`
	Uptime            *string                `json:"uptime,omitempty"`
	CreatedAt         *string                `json:"created_at,omitempty"`
	CpuUsage          *string                `json:"cpu_usage,omitempty"`
	MemoryUsage       *string                `json:"memory_usage,omitempty"`
	CpuLimit          *string                `json:"cpu_limit,omitempty"`
	MemoryLimit       *string                `json:"memory_limit,omitempty"`
	ActiveConnections *int                   `json:"active_connections,omitempty"`
	LatencyMs         *float64               `json:"latency_ms,omitempty"`
	Error             *string                `json:"error,omitempty"`
	ExtraDetails      map[string]interface{} `json:"extra_details,omitempty"`
}

type ClusterStatusResponse struct {
	Status           string                         `json:"status"`
	ClusterConnected bool                           `json:"cluster_connected"`
	Mode             string                         `json:"mode"`
	Namespace        string                         `json:"namespace"`
	Timestamp        string                         `json:"timestamp"`
	TotalServices    int                            `json:"total_services"`
	HealthyCount     int                            `json:"healthy_count"`
	Services         map[string]ServiceHealthDetail `json:"services"`
}

type IngressRoute struct {
	Service    string `json:"service"`
	URL        string `json:"url"`
	Host       string `json:"host"`
	Path       string `json:"path"`
	TLSEnabled bool   `json:"tls_enabled"`
}

type StorageVolume struct {
	Name         string  `json:"name"`
	PVCName      *string `json:"pvc_name,omitempty"`
	MountPath    string  `json:"mount_path"`
	Size         *string `json:"size,omitempty"`
	StorageClass *string `json:"storage_class,omitempty"`
	Status       *string `json:"status,omitempty"`
}

type ClusterTopologyResponse struct {
	Namespace      string                 `json:"namespace"`
	Mode           string                 `json:"mode"`
	IngressRoutes  []IngressRoute         `json:"ingress_routes"`
	StorageVolumes []StorageVolume        `json:"storage_volumes"`
	RawConfig      map[string]interface{} `json:"raw_config,omitempty"`
}

func (c *Client) ClusterStatus() (*ClusterStatusResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/cluster/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cluster status error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var status ClusterStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to parse cluster status response: %w", err)
	}

	return &status, nil
}

func (c *Client) ClusterTopology() (*ClusterTopologyResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/cluster/topology", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster topology: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cluster topology error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var topology ClusterTopologyResponse
	if err := json.NewDecoder(resp.Body).Decode(&topology); err != nil {
		return nil, fmt.Errorf("failed to parse cluster topology response: %w", err)
	}

	return &topology, nil
}
