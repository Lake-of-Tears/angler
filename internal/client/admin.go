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
