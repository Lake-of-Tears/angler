package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Dashboard represents a Lake of Tears business intelligence / visualization dashboard.
type Dashboard struct {
	ID           string           `json:"id"`
	WorkspaceID  string           `json:"workspace_id,omitempty"`
	Title        string           `json:"title"`
	Slug         string           `json:"slug"`
	Description  string           `json:"description,omitempty"`
	IsPublished  bool             `json:"is_published"`
	LayoutConfig []map[string]any `json:"layout_config,omitempty"`
	ChartsCount  int              `json:"charts_count"`
	CreatedBy    *string          `json:"created_by,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// DashboardStats provides aggregate metrics for native workspace dashboards.
type DashboardStats struct {
	TotalDashboards     int  `json:"total_dashboards"`
	PublishedDashboards int  `json:"published_dashboards"`
	TotalCharts         int  `json:"total_charts"`
	SupersetOnline      bool `json:"superset_online"`
}

// CreateDashboardRequest defines the payload for creating a dashboard.
type CreateDashboardRequest struct {
	Title        string           `json:"title"`
	Description  string           `json:"description,omitempty"`
	IsPublished  bool             `json:"is_published"`
	Template     string           `json:"template,omitempty"`
	LayoutConfig []map[string]any `json:"layout_config,omitempty"`
}

// UpdateDashboardRequest defines the fields for updating a dashboard.
type UpdateDashboardRequest struct {
	Title        *string          `json:"title,omitempty"`
	Description  *string          `json:"description,omitempty"`
	IsPublished  *bool            `json:"is_published,omitempty"`
	LayoutConfig []map[string]any `json:"layout_config,omitempty"`
}

// WidgetDataResponse represents live query data for a specific dashboard widget.
type WidgetDataResponse struct {
	WidgetID string         `json:"widget_id"`
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Data     map[string]any `json:"data"`
}

// ListDashboards lists all dashboards in the current workspace.
func (c *Client) ListDashboards() ([]Dashboard, error) {
	req, err := c.newRequest(http.MethodGet, "/api/dashboards", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list dashboards request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list dashboards request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var dashboards []Dashboard
	if err := json.NewDecoder(resp.Body).Decode(&dashboards); err != nil {
		return nil, fmt.Errorf("failed to parse dashboards list: %w", err)
	}
	return dashboards, nil
}

// GetDashboard retrieves dashboard details and layout configuration.
func (c *Client) GetDashboard(dashboardID string) (*Dashboard, error) {
	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(dashboardID))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get dashboard request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get dashboard request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var dash Dashboard
	if err := json.NewDecoder(resp.Body).Decode(&dash); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard: %w", err)
	}
	return &dash, nil
}

// CreateDashboard creates a new dashboard in the active workspace.
func (c *Client) CreateDashboard(reqPayload CreateDashboardRequest) (*Dashboard, error) {
	req, err := c.newRequest(http.MethodPost, "/api/dashboards", reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create dashboard request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var dash Dashboard
	if err := json.NewDecoder(resp.Body).Decode(&dash); err != nil {
		return nil, fmt.Errorf("failed to parse created dashboard: %w", err)
	}
	return &dash, nil
}

// UpdateDashboard updates an existing dashboard.
func (c *Client) UpdateDashboard(dashboardID string, reqPayload UpdateDashboardRequest) (*Dashboard, error) {
	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(dashboardID))
	req, err := c.newRequest(http.MethodPatch, path, reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to create update dashboard request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update dashboard request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var dash Dashboard
	if err := json.NewDecoder(resp.Body).Decode(&dash); err != nil {
		return nil, fmt.Errorf("failed to parse updated dashboard: %w", err)
	}
	return &dash, nil
}

// DeleteDashboard removes a dashboard from the current workspace.
func (c *Client) DeleteDashboard(dashboardID string) error {
	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(dashboardID))
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete dashboard request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete dashboard request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetDashboardStats retrieves summary metrics for workspace dashboards.
func (c *Client) GetDashboardStats() (*DashboardStats, error) {
	req, err := c.newRequest(http.MethodGet, "/api/dashboards/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get dashboard stats request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get dashboard stats request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats DashboardStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard stats: %w", err)
	}
	return &stats, nil
}

// GetWidgetData executes analytical query for a widget and returns chart-ready data.
func (c *Client) GetWidgetData(dashboardID, widgetID, timeRange string) (*WidgetDataResponse, error) {
	u, err := url.Parse(fmt.Sprintf("/api/dashboards/%s/widgets/%s/data", url.PathEscape(dashboardID), url.PathEscape(widgetID)))
	if err != nil {
		return nil, fmt.Errorf("failed to build widget data url: %w", err)
	}
	if timeRange != "" {
		q := u.Query()
		q.Set("time_range", timeRange)
		u.RawQuery = q.Encode()
	}

	req, err := c.newRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get widget data request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get widget data request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var data WidgetDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse widget data: %w", err)
	}
	return &data, nil
}
