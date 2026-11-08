package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// UpdateUserRequest represents the payload for updating user attributes.
type UpdateUserRequest struct {
	Role        *string `json:"role,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
}

// CreateWorkspaceRequest represents parameters for provisioning a new workspace.
type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
}

// WorkspaceMember represents a user assigned to a workspace.
type WorkspaceMember struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	DisplayName  *string   `json:"display_name,omitempty"`
	Role         string    `json:"role"`
	GlobalRole   string    `json:"global_role"`
	IsSuperadmin bool      `json:"is_superadmin"`
	JoinedAt     time.Time `json:"joined_at"`
}

// AddWorkspaceMemberRequest defines parameters to grant a user access to a workspace.
type AddWorkspaceMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ListUsers lists all users in the Lake of Tears cluster (superadmin required).
func (c *Client) ListUsers() ([]User, error) {
	req, err := c.newRequest(http.MethodGet, "/api/users", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list users request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list users request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to parse users: %w", err)
	}
	return users, nil
}

// GetUser retrieves a single user's profile.
func (c *Client) GetUser(userID string) (*User, error) {
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.ID == userID || u.Email == userID {
			return &u, nil
		}
	}
	return nil, fmt.Errorf("user %q not found", userID)
}

// UpdateUser modifies a user's role, active status, or display name.
func (c *Client) UpdateUser(userID string, updateReq UpdateUserRequest) (*User, error) {
	path := fmt.Sprintf("/api/users/%s", url.PathEscape(userID))
	req, err := c.newRequest(http.MethodPatch, path, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create update user request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update user request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to parse updated user: %w", err)
	}
	return &user, nil
}

// ListAllWorkspaces lists workspaces available to the authenticated context.
func (c *Client) ListAllWorkspaces() ([]Workspace, error) {
	req, err := c.newRequest(http.MethodGet, "/api/workspaces", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list workspaces request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list workspaces request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var workspaces []Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, fmt.Errorf("failed to parse workspaces: %w", err)
	}
	return workspaces, nil
}

// GetWorkspace retrieves details of a specific workspace by ID or slug.
func (c *Client) GetWorkspace(workspaceID string) (*Workspace, error) {
	path := fmt.Sprintf("/api/workspaces/%s", url.PathEscape(workspaceID))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get workspace request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get workspace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var ws Workspace
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, fmt.Errorf("failed to parse workspace: %w", err)
	}
	return &ws, nil
}

// CreateWorkspace provisions a new analytical workspace.
func (c *Client) CreateWorkspace(createReq CreateWorkspaceRequest) (*Workspace, error) {
	req, err := c.newRequest(http.MethodPost, "/api/workspaces", createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create workspace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var ws Workspace
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, fmt.Errorf("failed to parse created workspace: %w", err)
	}
	return &ws, nil
}

// ListWorkspaceMembers returns members belonging to a workspace.
func (c *Client) ListWorkspaceMembers(workspaceID string) ([]WorkspaceMember, error) {
	path := fmt.Sprintf("/api/workspaces/%s/members", url.PathEscape(workspaceID))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list workspace members request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list workspace members request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var members []WorkspaceMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("failed to parse workspace members: %w", err)
	}
	return members, nil
}

// AddWorkspaceMember assigns a user to a workspace with a specific role.
func (c *Client) AddWorkspaceMember(workspaceID string, addReq AddWorkspaceMemberRequest) error {
	path := fmt.Sprintf("/api/workspaces/%s/members", url.PathEscape(workspaceID))
	req, err := c.newRequest(http.MethodPost, path, addReq)
	if err != nil {
		return fmt.Errorf("failed to create add workspace member request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute add workspace member request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// RemoveWorkspaceMember removes a member from a workspace.
func (c *Client) RemoveWorkspaceMember(workspaceID, userID string) error {
	path := fmt.Sprintf("/api/workspaces/%s/members/%s", url.PathEscape(workspaceID), url.PathEscape(userID))
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create remove workspace member request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute remove workspace member request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
