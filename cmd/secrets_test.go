package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestSecretsCommands(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/secrets/scopes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"id":              "sc-1",
					"workspace_id":    "ws-1",
					"name":            "prod-vault",
					"backend_type":    "hashicorp_vault",
					"user_permission": "MANAGE",
					"key_count":       2,
					"created_at":      "2026-11-10T10:00:00Z",
				},
			})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "sc-2",
				"workspace_id":    "ws-1",
				"name":            "gcp-vault",
				"backend_type":    "gcp_secret_manager",
				"user_permission": "MANAGE",
				"key_count":       0,
				"created_at":      "2026-11-10T10:05:00Z",
			})
			return
		}
	})

	mux.HandleFunc("/api/secrets/scopes/prod-vault/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"key": "api_key", "version": 2, "created_at": "2026-11-10T10:00:00Z"},
			})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "api_key", "version": 3, "created_at": "2026-11-10T10:10:00Z",
			})
			return
		}
	})

	mux.HandleFunc("/api/secrets/scopes/prod-vault/keys/api_key", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":     "api_key",
			"value":   "top-secret-token-999",
			"version": 3,
		})
	})

	mux.HandleFunc("/api/secrets/scopes/prod-vault/acls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"id":             "acl-100",
					"scope_id":       "sc-1",
					"principal_type": "role",
					"principal_id":   "read",
					"permission":     "READ",
					"created_at":     "2026-11-10T10:00:00Z",
				},
			})
			return
		}
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "acl-101",
				"scope_id":       "sc-1",
				"principal_type": "role",
				"principal_id":   "write",
				"permission":     "WRITE",
			})
			return
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	var outBuf, errBuf bytes.Buffer
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	t.Cleanup(func() {
		ui.OutWriter = os.Stdout
		ui.ErrWriter = os.Stderr
	})

	// 1. scopes list
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "secrets", "scopes", "list"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("secrets scopes list failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "prod-vault") || !strings.Contains(outBuf.String(), "hashicorp_vault") {
		t.Fatalf("expected prod-vault in output, got: %s", outBuf.String())
	}

	// 2. put with --string-value
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "secrets", "put", "prod-vault", "api_key", "--string-value", "top-secret-token-999"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("secrets put failed (%d): %s", code, errBuf.String())
	}

	// 3. get (masked by default in table mode)
	outBuf.Reset()
	secretGetRaw = false
	secretGetUnmask = false
	secretGetShowValue = false
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "secrets", "get", "prod-vault", "api_key"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("secrets get failed (%d): %s", code, errBuf.String())
	}
	if strings.Contains(outBuf.String(), "top-secret-token-999") {
		t.Fatalf("expected secret value to be masked by default in table output, got: %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "••••••••") {
		t.Fatalf("expected mask indicator in table output, got: %s", outBuf.String())
	}

	// 4. get --raw (unmasked for piping)
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "secrets", "get", "prod-vault", "api_key", "--raw"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("secrets get --raw failed (%d): %s", code, errBuf.String())
	}
	if strings.TrimSpace(outBuf.String()) != "top-secret-token-999" {
		t.Fatalf("expected raw secret 'top-secret-token-999', got %q", outBuf.String())
	}

	// 5. acls set
	outBuf.Reset()
	secretGetRaw = false
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "json", "secrets", "acls", "set", "prod-vault", "--principal", "write", "--principal-type", "role", "--permission", "WRITE"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("secrets acls set failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), `"permission": "WRITE"`) {
		t.Fatalf("expected WRITE permission in JSON output, got: %s", outBuf.String())
	}
}
