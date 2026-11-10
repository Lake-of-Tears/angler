package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestSecretsAndVaultsClient(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/secrets/scopes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]SecretScope{
				{ID: "sc-1", Name: "prod-creds", BackendType: "hashicorp_vault", UserPermission: "MANAGE", KeyCount: 2},
			})
			return
		}
		if r.Method == http.MethodPost {
			var req SecretScopeCreateRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(SecretScope{
				ID: "sc-2", Name: req.Name, BackendType: req.BackendType, UserPermission: "MANAGE", KeyCount: 0,
			})
			return
		}
	})

	mux.HandleFunc("/api/secrets/scopes/prod-creds/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]SecretKeySummary{
				{Key: "db_password", Version: 3},
			})
			return
		}
		if r.Method == http.MethodPost {
			var req SecretKeyPutRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(SecretKeySummary{Key: req.Key, Version: 4})
			return
		}
	})

	mux.HandleFunc("/api/secrets/scopes/prod-creds/keys/db_password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(SecretValue{Key: "db_password", Value: "super-secret-123", Version: 4})
			return
		}
		if r.Method == http.MethodDelete {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "deleted"})
			return
		}
	})

	mux.HandleFunc("/api/secrets/scopes/prod-creds/acls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]SecretACL{
				{ID: "acl-1", ScopeID: "sc-1", PrincipalType: "role", PrincipalID: "read", Permission: "READ"},
			})
			return
		}
		if r.Method == http.MethodPost {
			var req SecretACLSetRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode(SecretACL{
				ID: "acl-2", ScopeID: "sc-1", PrincipalType: req.PrincipalType, PrincipalID: req.PrincipalID, Permission: req.Permission,
			})
			return
		}
	})

	// Test fallback from /api/admin/vaults to /api/admin/vault-providers
	mux.HandleFunc("/api/admin/vault-providers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]VaultProviderStatus{
			{Provider: "internal", DisplayName: "Internal Encrypted Database", IsConfigured: true},
			{Provider: "hashicorp_vault", DisplayName: "HashiCorp Vault", IsConfigured: true},
		})
	})

	mux.HandleFunc("/api/admin/vault-providers/hashicorp_vault/test", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(VaultTestConnectionResponse{
			Success: true,
			Message: "Connected to HashiCorp Vault (KV v2 unsealed)",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(config.Context{Endpoint: srv.URL, Token: "tok", Workspace: "default"})

	scopes, err := c.ListSecretScopes()
	if err != nil || len(scopes) != 1 || scopes[0].Name != "prod-creds" {
		t.Fatalf("ListSecretScopes failed: %v, %+v", err, scopes)
	}

	created, err := c.CreateSecretScope(SecretScopeCreateRequest{Name: "gcp-scope", BackendType: "gcp_secret_manager"})
	if err != nil || created.BackendType != "gcp_secret_manager" {
		t.Fatalf("CreateSecretScope failed: %v, %+v", err, created)
	}

	putRes, err := c.PutSecretKey("prod-creds", "db_password", "super-secret-123")
	if err != nil || putRes.Version != 4 {
		t.Fatalf("PutSecretKey failed: %v, %+v", err, putRes)
	}

	secVal, err := c.GetSecretKey("prod-creds", "db_password")
	if err != nil || secVal.Value != "super-secret-123" {
		t.Fatalf("GetSecretKey failed: %v, %+v", err, secVal)
	}

	acls, err := c.ListSecretACLs("prod-creds")
	if err != nil || len(acls) != 1 {
		t.Fatalf("ListSecretACLs failed: %v, %+v", err, acls)
	}

	provs, err := c.ListVaultProviders()
	if err != nil || len(provs) != 2 {
		t.Fatalf("ListVaultProviders fallback failed: %v, %+v", err, provs)
	}

	testRes, err := c.TestVaultProvider("hashicorp_vault", nil)
	if err != nil || !testRes.Success {
		t.Fatalf("TestVaultProvider failed: %v, %+v", err, testRes)
	}
}
