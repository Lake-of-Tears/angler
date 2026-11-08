package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	MyRole    *string   `json:"my_role,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type WorkspaceDetail struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type PersonalAccessToken struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	TokenPrefix      string            `json:"token_prefix"`
	WorkspaceID      *string           `json:"workspace_id,omitempty"`
	WorkspaceName    *string           `json:"workspace_name,omitempty"`
	WorkspaceIDs     []string          `json:"workspace_ids,omitempty"`
	WorkspaceNames   []string          `json:"workspace_names,omitempty"`
	WorkspaceAccess  map[string]string `json:"workspace_access,omitempty"`
	WorkspaceDetails []WorkspaceDetail `json:"workspace_details,omitempty"`
	Scopes           []string          `json:"scopes,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time        `json:"last_used_at,omitempty"`
	IsRevoked        bool              `json:"is_revoked"`
	RevokedAt        *time.Time        `json:"revoked_at,omitempty"`
}

type TokenCreateRequest struct {
	Name            string            `json:"name"`
	LifetimeDays    *int              `json:"lifetime_days,omitempty"`
	WorkspaceID     *string           `json:"workspace_id,omitempty"`
	WorkspaceIDs    []string          `json:"workspace_ids,omitempty"`
	WorkspaceAccess map[string]string `json:"workspace_access,omitempty"`
	Scopes          []string          `json:"scopes,omitempty"`
}

type TokenCreateResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Token           string            `json:"token"`
	TokenPrefix     string            `json:"token_prefix"`
	WorkspaceID     *string           `json:"workspace_id,omitempty"`
	WorkspaceIDs    []string          `json:"workspace_ids,omitempty"`
	WorkspaceAccess map[string]string `json:"workspace_access,omitempty"`
	Scopes          []string          `json:"scopes,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty"`
}

type RevokeTokenResponse struct {
	OK        bool   `json:"ok"`
	RevokedID string `json:"revoked_id"`
}

func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: unable to read response body", resp.StatusCode)
	}

	var errPayload struct {
		Detail interface{} `json:"detail"`
		Error  string      `json:"error"`
	}

	if json.Unmarshal(body, &errPayload) == nil {
		if s, ok := errPayload.Detail.(string); ok && strings.TrimSpace(s) != "" {
			return fmt.Errorf("%s", s)
		}
		if strings.TrimSpace(errPayload.Error) != "" {
			return fmt.Errorf("%s", errPayload.Error)
		}
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed != "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, trimmed)
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
}

// CurrentUser returns the authenticated user profile for the current token/session.
func (c *Client) CurrentUser() (*User, error) {
	req, err := c.newRequest(http.MethodGet, "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication check failed: %w", parseAPIError(resp))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &user, nil
}

// ListWorkspaces retrieves the accessible workspaces for the authenticated user.
func (c *Client) ListWorkspaces() ([]Workspace, error) {
	req, err := c.newRequest(http.MethodGet, "/api/workspaces", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list workspaces: %w", parseAPIError(resp))
	}

	var workspaces []Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, fmt.Errorf("failed to decode workspaces response: %w", err)
	}

	return workspaces, nil
}

// CreateToken provisions a new Personal Access Token.
func (c *Client) CreateToken(req *TokenCreateRequest) (*TokenCreateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("token create request cannot be nil")
	}

	httpReq, err := c.newRequest(http.MethodPost, "/api/auth/tokens", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create personal access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create token: %w", parseAPIError(resp))
	}

	var created TokenCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode token creation response: %w", err)
	}

	return &created, nil
}

// ListTokens lists Personal Access Tokens for the current user.
func (c *Client) ListTokens(includeRevoked bool) ([]PersonalAccessToken, error) {
	path := "/api/auth/tokens"
	if includeRevoked {
		path += "?include_revoked=true"
	}

	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list tokens: %w", parseAPIError(resp))
	}

	var tokens []PersonalAccessToken
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode tokens list: %w", err)
	}

	return tokens, nil
}

// RevokeToken revokes an access token by its UUID.
func (c *Client) RevokeToken(tokenID string) (*RevokeTokenResponse, error) {
	trimmedID := strings.TrimSpace(tokenID)
	if trimmedID == "" {
		return nil, fmt.Errorf("token ID is required")
	}

	path := fmt.Sprintf("/api/auth/tokens/%s", url.PathEscape(trimmedID))
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to revoke token: %w", parseAPIError(resp))
	}

	var revokeResp RevokeTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&revokeResp); err != nil {
		return nil, fmt.Errorf("failed to decode revoke response: %w", err)
	}

	return &revokeResp, nil
}
