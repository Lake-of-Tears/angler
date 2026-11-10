package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SystemSettings encapsulates cluster-wide administrative configuration.
type SystemSettings struct {
	CatalogSoftDeleteDays            int     `json:"catalog_soft_delete_days"`
	WorkspaceInactiveGracePeriodDays int     `json:"workspace_inactive_grace_period_days"`
	SessionInactivityTimeoutMinutes  int     `json:"session_inactivity_timeout_minutes"`
	SessionMaxHours                  int     `json:"session_max_hours"`
	GeminiAPIKeyConfigured           bool    `json:"gemini_api_key_configured"`
	GeminiAPIKeyMasked               *string `json:"gemini_api_key_masked,omitempty"`
	GeminiDefaultModel               *string `json:"gemini_default_model,omitempty"`
}

// UpdateSystemSettingsRequest defines configuration options to update.
type UpdateSystemSettingsRequest struct {
	CatalogSoftDeleteDays            *int    `json:"catalog_soft_delete_days,omitempty"`
	WorkspaceInactiveGracePeriodDays *int    `json:"workspace_inactive_grace_period_days,omitempty"`
	SessionInactivityTimeoutMinutes  *int    `json:"session_inactivity_timeout_minutes,omitempty"`
	SessionMaxHours                  *int    `json:"session_max_hours,omitempty"`
	GeminiAPIKey                     *string `json:"gemini_api_key,omitempty"`
	ClearGeminiAPIKey                bool    `json:"clear_gemini_api_key,omitempty"`
	GeminiDefaultModel               *string `json:"gemini_default_model,omitempty"`
}

// UserSession represents an active session in the cluster.
type UserSession struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Email        *string    `json:"email,omitempty"`
	DisplayName  *string    `json:"display_name,omitempty"`
	IPAddress    *string    `json:"ip_address,omitempty"`
	UserAgent    *string    `json:"user_agent,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	IsRevoked    bool       `json:"is_revoked"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	IsCurrent    bool       `json:"is_current"`
}

// GetSystemSettings retrieves system-wide administrative configuration.
func (c *Client) GetSystemSettings() (*SystemSettings, error) {
	req, err := c.newRequest(http.MethodGet, "/api/admin/settings", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get settings request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get settings request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var settings SystemSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to parse system settings: %w", err)
	}
	return &settings, nil
}

// UpdateSystemSettings updates system configuration values.
func (c *Client) UpdateSystemSettings(reqPayload UpdateSystemSettingsRequest) (*SystemSettings, error) {
	req, err := c.newRequest(http.MethodPatch, "/api/admin/settings", reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create update settings request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update settings request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var settings SystemSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to parse updated system settings: %w", err)
	}
	return &settings, nil
}

// ListSessions lists active user and service sessions across the cluster.
func (c *Client) ListSessions() ([]UserSession, error) {
	req, err := c.newRequest(http.MethodGet, "/api/admin/sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list sessions request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list sessions request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var sessions []UserSession
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to parse sessions: %w", err)
	}
	return sessions, nil
}

// RevokeSession terminates an active session.
func (c *Client) RevokeSession(sessionID string) error {
	path := fmt.Sprintf("/api/admin/sessions/%s", url.PathEscape(sessionID))
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke session request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute revoke session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── Multi-Provider Enterprise AI Administration (`/api/admin/ai-provider`) ──

type AIProviderDescriptor struct {
	Name        string           `json:"name"`
	Label       string           `json:"label"`
	Description string           `json:"description,omitempty"`
	Active      bool             `json:"active"`
	Configured  bool             `json:"configured"`
	Values      map[string]any   `json:"values,omitempty"`
	Fields      []map[string]any `json:"fields,omitempty"`
}

type AIProviderSettingsResponse struct {
	ActiveProvider     string                 `json:"active_provider"`
	Providers          []AIProviderDescriptor `json:"providers"`
	DeprecationWarning map[string]any         `json:"deprecation_warning,omitempty"`
}

type SaveAIProviderRequest struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
	Activate bool           `json:"activate"`
}

type TestAIProviderRequest struct {
	Provider string         `json:"provider,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type DiscoverAIModelsRequest struct {
	Provider string         `json:"provider,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type DiscoverAIModelsResponse struct {
	OK       bool                `json:"ok"`
	Provider string              `json:"provider"`
	Models   map[string][]string `json:"models"`
	Message  string              `json:"message"`
}

type AIDeprecationWarningResponse struct {
	Warning map[string]any `json:"warning"`
}

func (c *Client) GetAIProviderSettings() (*AIProviderSettingsResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/admin/ai-provider", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AI provider settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get AI provider settings error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res AIProviderSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode AI provider settings: %w", err)
	}
	return &res, nil
}

func (c *Client) SaveAIProviderSettings(reqPayload SaveAIProviderRequest) (*AIProviderSettingsResponse, error) {
	req, err := c.newRequest(http.MethodPut, "/api/admin/ai-provider", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to save AI provider settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("save AI provider settings error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res AIProviderSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode updated AI provider settings: %w", err)
	}
	return &res, nil
}

func (c *Client) TestAIProvider(reqPayload TestAIProviderRequest) (map[string]any, error) {
	req, err := c.newRequest(http.MethodPost, "/api/admin/ai-provider/test", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to test AI provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("test AI provider error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode AI provider test response: %w", err)
	}
	return res, nil
}

func (c *Client) DiscoverAIModels(reqPayload DiscoverAIModelsRequest) (*DiscoverAIModelsResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/admin/ai-provider/discover-models", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to discover AI models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discover AI models error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res DiscoverAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode discovered AI models: %w", err)
	}
	return &res, nil
}

func (c *Client) GetAIDeprecationWarning() (*AIDeprecationWarningResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/admin/ai-provider/deprecation-warning", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI deprecation warning: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get AI deprecation warning error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res AIDeprecationWarningResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode AI deprecation warning: %w", err)
	}
	return &res, nil
}

func (c *Client) DismissAIDeprecationWarning() (map[string]any, error) {
	req, err := c.newRequest(http.MethodPost, "/api/admin/ai-provider/dismiss-deprecation-warning", map[string]any{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to dismiss AI deprecation warning: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dismiss AI deprecation warning error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode dismiss response: %w", err)
	}
	return res, nil
}

// ── Kubernetes Platform Service Lifecycle Controls (`/api/admin/services`) ──

// PlatformServiceInfo represents a Kubernetes platform deployment managed by Lake of Tears.
type PlatformServiceInfo struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	DisplayName       string         `json:"display_name"`
	Description       string         `json:"description,omitempty"`
	Category          string         `json:"category,omitempty"`
	Status            string         `json:"status"`
	DesiredReplicas   int            `json:"desired_replicas"`
	ReadyReplicas     int            `json:"ready_replicas"`
	IsCore            bool           `json:"is_core"`
	Restarts          int            `json:"restarts"`
	Uptime            *string        `json:"uptime,omitempty"`
	CreatedAt         *string        `json:"created_at,omitempty"`
	PodName           *string        `json:"pod_name,omitempty"`
	NodeName          *string        `json:"node_name,omitempty"`
	CPUUsage          *string        `json:"cpu_usage,omitempty"`
	MemoryUsage       *string        `json:"memory_usage,omitempty"`
	CPULimit          *string        `json:"cpu_limit,omitempty"`
	MemoryLimit       *string        `json:"memory_limit,omitempty"`
	LastRestart       *string        `json:"last_restart,omitempty"`
	Image             *string        `json:"image,omitempty"`
	Port              *int           `json:"port,omitempty"`
	ActiveConnections *int           `json:"active_connections,omitempty"`
	LatencyMs         *float64       `json:"latency_ms,omitempty"`
	Error             *string        `json:"error,omitempty"`
	ExtraDetails      map[string]any `json:"extra_details,omitempty"`
}

// PlatformServicesListResponse represents the response from GET /api/admin/services.
type PlatformServicesListResponse struct {
	ClusterConnected bool                  `json:"cluster_connected"`
	Mode             string                `json:"mode"`
	Namespace        string                `json:"namespace"`
	TotalServices    int                   `json:"total_services"`
	HealthyCount     int                   `json:"healthy_count"`
	Services         []PlatformServiceInfo `json:"services"`
}

// PlatformServiceActionResponse represents the response from restart, stop, or start lifecycle actions.
type PlatformServiceActionResponse struct {
	Success     bool   `json:"success"`
	Service     string `json:"service"`
	Deployment  string `json:"deployment,omitempty"`
	Action      string `json:"action"`
	Message     string `json:"message"`
	RestartedAt string `json:"restarted_at,omitempty"`
}

// ListPlatformServices fetches all Kubernetes platform deployments and their states.
func (c *Client) ListPlatformServices() (*PlatformServicesListResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/admin/services", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list platform services: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list platform services failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res PlatformServicesListResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode platform services response: %w", err)
	}
	return &res, nil
}

// RestartPlatformService triggers a rolling restart of a platform deployment.
func (c *Client) RestartPlatformService(serviceName string) (*PlatformServiceActionResponse, error) {
	path := fmt.Sprintf("/api/admin/services/%s/restart", url.PathEscape(serviceName))
	req, err := c.newRequest(http.MethodPost, path, map[string]any{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to restart platform service %q: %w", serviceName, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("restart platform service failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res PlatformServiceActionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode restart service response: %w", err)
	}
	return &res, nil
}

// StopPlatformService scales a non-core platform service deployment to 0 replicas.
func (c *Client) StopPlatformService(serviceName string) (*PlatformServiceActionResponse, error) {
	path := fmt.Sprintf("/api/admin/services/%s/stop", url.PathEscape(serviceName))
	req, err := c.newRequest(http.MethodPost, path, map[string]any{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to stop platform service %q: %w", serviceName, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("stop platform service failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res PlatformServiceActionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode stop service response: %w", err)
	}
	return &res, nil
}

// StartPlatformService scales a stopped platform service back to active replicas.
func (c *Client) StartPlatformService(serviceName string) (*PlatformServiceActionResponse, error) {
	path := fmt.Sprintf("/api/admin/services/%s/start", url.PathEscape(serviceName))
	req, err := c.newRequest(http.MethodPost, path, map[string]any{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start platform service %q: %w", serviceName, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("start platform service failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res PlatformServiceActionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode start service response: %w", err)
	}
	return &res, nil
}

