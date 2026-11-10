package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type SecretScope struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspace_id"`
	Name           string         `json:"name"`
	BackendType    string         `json:"backend_type"`
	ProviderConfig map[string]any `json:"provider_config,omitempty"`
	CreatedBy      *string        `json:"created_by,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      *string        `json:"updated_at,omitempty"`
	UserPermission string         `json:"user_permission"`
	KeyCount       int            `json:"key_count"`
}

type SecretScopeCreateRequest struct {
	Name           string         `json:"name"`
	BackendType    string         `json:"backend_type"`
	ProviderConfig map[string]any `json:"provider_config,omitempty"`
}

type SecretKeySummary struct {
	Key       string  `json:"key"`
	Version   int     `json:"version"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty"`
}

type SecretKeyPutRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SecretValue struct {
	Key       string  `json:"key"`
	Value     string  `json:"value"`
	Version   int     `json:"version"`
	UpdatedAt *string `json:"updated_at,omitempty"`
}

type SecretACL struct {
	ID            string  `json:"id"`
	ScopeID       string  `json:"scope_id"`
	PrincipalType string  `json:"principal_type"`
	PrincipalID   string  `json:"principal_id"`
	Permission    string  `json:"permission"`
	CreatedAt     string  `json:"created_at,omitempty"`
	CreatedBy     *string `json:"created_by,omitempty"`
}

type SecretACLSetRequest struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	Permission    string `json:"permission"`
}

type VaultProviderStatus struct {
	Provider     string         `json:"provider"`
	DisplayName  string         `json:"display_name"`
	IsConfigured bool           `json:"is_configured"`
	Config       map[string]any `json:"config,omitempty"`
}

type VaultProviderConfigUpdateRequest struct {
	Config map[string]any `json:"config"`
}

type VaultTestConnectionRequest struct {
	Config map[string]any `json:"config,omitempty"`
}

type VaultTestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type AdminVaultScopeSummary struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	WorkspaceID   *string `json:"workspace_id,omitempty"`
	WorkspaceName string  `json:"workspace_name"`
	BackendType   string  `json:"backend_type"`
	KeyCount      int     `json:"key_count"`
	CreatedAt     *string `json:"created_at,omitempty"`
}

func (c *Client) ListSecretScopes() ([]SecretScope, error) {
	req, err := c.newRequest(http.MethodGet, "/api/secrets/scopes", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secret scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list secret scopes error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var scopes []SecretScope
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode secret scopes: %w", err)
	}
	return scopes, nil
}

func (c *Client) GetSecretScope(scope string) (*SecretScope, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("/api/secrets/scopes/%s", url.PathEscape(scope)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret scope %s: %w", scope, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get secret scope error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var sc SecretScope
	if err := json.NewDecoder(resp.Body).Decode(&sc); err != nil {
		return nil, fmt.Errorf("failed to decode secret scope: %w", err)
	}
	return &sc, nil
}

func (c *Client) CreateSecretScope(reqPayload SecretScopeCreateRequest) (*SecretScope, error) {
	req, err := c.newRequest(http.MethodPost, "/api/secrets/scopes", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret scope: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create secret scope error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var sc SecretScope
	if err := json.NewDecoder(resp.Body).Decode(&sc); err != nil {
		return nil, fmt.Errorf("failed to decode created secret scope: %w", err)
	}
	return &sc, nil
}

func (c *Client) DeleteSecretScope(scope string) error {
	req, err := c.newRequest(http.MethodDelete, fmt.Sprintf("/api/secrets/scopes/%s", url.PathEscape(scope)), nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete secret scope %s: %w", scope, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete secret scope error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) ListSecretKeys(scope string) ([]SecretKeySummary, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("/api/secrets/scopes/%s/keys", url.PathEscape(scope)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secret keys in scope %s: %w", scope, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list secret keys error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var keys []SecretKeySummary
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("failed to decode secret keys: %w", err)
	}
	return keys, nil
}

func (c *Client) PutSecretKey(scope, key, value string) (*SecretKeySummary, error) {
	req, err := c.newRequest(
		http.MethodPost,
		fmt.Sprintf("/api/secrets/scopes/%s/keys", url.PathEscape(scope)),
		SecretKeyPutRequest{Key: key, Value: value},
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to put secret key %s/%s: %w", scope, key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("put secret key error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var summary SecretKeySummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode secret key summary: %w", err)
	}
	return &summary, nil
}

func (c *Client) GetSecretKey(scope, key string) (*SecretValue, error) {
	req, err := c.newRequest(
		http.MethodGet,
		fmt.Sprintf("/api/secrets/scopes/%s/keys/%s", url.PathEscape(scope), url.PathEscape(key)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key %s/%s: %w", scope, key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get secret key error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var val SecretValue
	if err := json.NewDecoder(resp.Body).Decode(&val); err != nil {
		return nil, fmt.Errorf("failed to decode secret value: %w", err)
	}
	return &val, nil
}

func (c *Client) DeleteSecretKey(scope, key string) error {
	req, err := c.newRequest(
		http.MethodDelete,
		fmt.Sprintf("/api/secrets/scopes/%s/keys/%s", url.PathEscape(scope), url.PathEscape(key)),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete secret key %s/%s: %w", scope, key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete secret key error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) ListSecretACLs(scope string) ([]SecretACL, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("/api/secrets/scopes/%s/acls", url.PathEscape(scope)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secret ACLs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list secret ACLs error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var acls []SecretACL
	if err := json.NewDecoder(resp.Body).Decode(&acls); err != nil {
		return nil, fmt.Errorf("failed to decode secret ACLs: %w", err)
	}
	return acls, nil
}

func (c *Client) SetSecretACL(scope string, reqPayload SecretACLSetRequest) (*SecretACL, error) {
	req, err := c.newRequest(http.MethodPost, fmt.Sprintf("/api/secrets/scopes/%s/acls", url.PathEscape(scope)), reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to set secret ACL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("set secret ACL error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var acl SecretACL
	if err := json.NewDecoder(resp.Body).Decode(&acl); err != nil {
		return nil, fmt.Errorf("failed to decode secret ACL: %w", err)
	}
	return &acl, nil
}

func (c *Client) DeleteSecretACL(scope, aclID string) error {
	req, err := c.newRequest(
		http.MethodDelete,
		fmt.Sprintf("/api/secrets/scopes/%s/acls/%s", url.PathEscape(scope), url.PathEscape(aclID)),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete secret ACL %s: %w", aclID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete secret ACL error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── Admin Vault Providers (`/api/admin/vaults` & `/api/admin/vault-providers`) ──

func (c *Client) doAdminVaultRequest(method, subPath string, body any) (*http.Response, error) {
	primaryPath := "/api/admin/vaults" + subPath
	req, err := c.newRequest(method, primaryPath, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		fallbackPath := "/api/admin/vault-providers" + subPath
		req2, err2 := c.newRequest(method, fallbackPath, body)
		if err2 != nil {
			return nil, err2
		}
		return c.HTTPClient.Do(req2)
	}
	return resp, nil
}

func (c *Client) ListVaultProviders() ([]VaultProviderStatus, error) {
	resp, err := c.doAdminVaultRequest(http.MethodGet, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list vault providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list vault providers error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault providers response: %w", err)
	}

	var providers []VaultProviderStatus
	if err := json.Unmarshal(rawBody, &providers); err == nil {
		return providers, nil
	}

	var wrapped struct {
		Providers []VaultProviderStatus `json:"providers"`
	}
	if err := json.Unmarshal(rawBody, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to decode vault providers: %w", err)
	}
	return wrapped.Providers, nil
}

func (c *Client) GetVaultProvider(provider string) (*VaultProviderStatus, error) {
	resp, err := c.doAdminVaultRequest(http.MethodGet, "/"+url.PathEscape(provider), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault provider %s: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get vault provider error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var prov VaultProviderStatus
	if err := json.NewDecoder(resp.Body).Decode(&prov); err != nil {
		return nil, fmt.Errorf("failed to decode vault provider: %w", err)
	}
	return &prov, nil
}

func (c *Client) UpdateVaultProvider(provider string, configMap map[string]any) (*VaultProviderStatus, error) {
	resp, err := c.doAdminVaultRequest(
		http.MethodPut,
		"/"+url.PathEscape(provider),
		VaultProviderConfigUpdateRequest{Config: configMap},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to configure vault provider %s: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("configure vault provider error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var prov VaultProviderStatus
	if err := json.NewDecoder(resp.Body).Decode(&prov); err != nil {
		return nil, fmt.Errorf("failed to decode updated vault provider: %w", err)
	}
	return &prov, nil
}

func (c *Client) TestVaultProvider(provider string, configMap map[string]any) (*VaultTestConnectionResponse, error) {
	resp, err := c.doAdminVaultRequest(
		http.MethodPost,
		"/"+url.PathEscape(provider)+"/test",
		VaultTestConnectionRequest{Config: configMap},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to test vault provider %s: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("test vault provider error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var testRes VaultTestConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&testRes); err != nil {
		return nil, fmt.Errorf("failed to decode vault test response: %w", err)
	}
	return &testRes, nil
}

func (c *Client) ListAllVaultScopes() ([]AdminVaultScopeSummary, error) {
	resp, err := c.doAdminVaultRequest(http.MethodGet, "/scopes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list global secret scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list global secret scopes error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var scopes []AdminVaultScopeSummary
	if err := json.NewDecoder(resp.Body).Decode(&scopes); err != nil {
		return nil, fmt.Errorf("failed to decode global secret scopes: %w", err)
	}
	return scopes, nil
}
