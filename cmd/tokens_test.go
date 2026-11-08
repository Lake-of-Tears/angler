package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestTokensCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundTokens bool
	for _, c := range commands {
		if c.Name() == "tokens" {
			foundTokens = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			if !subMap["create"] {
				t.Errorf("expected 'create' subcommand on tokens")
			}
			if !subMap["list"] {
				t.Errorf("expected 'list' subcommand on tokens")
			}
			if !subMap["revoke"] {
				t.Errorf("expected 'revoke' subcommand on tokens")
			}
		}
	}

	if !foundTokens {
		t.Errorf("expected 'tokens' command to be registered on rootCmd")
	}
}

func TestTokensCreateAndListAndRevoke(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/workspaces" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":      "11111111-2222-3333-4444-555555555555",
					"name":    "Default Workspace",
					"slug":    "default",
					"status":  "active",
					"my_role": "admin",
				},
			})
		case r.URL.Path == "/api/auth/tokens" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "tok-uuid-12345",
				"name":         "ci-runner",
				"token":        "lot_pat_9876543210abcdef9876543210abcdef",
				"token_prefix": "lot_pat_98765432",
				"workspace_id": "11111111-2222-3333-4444-555555555555",
				"created_at":   time.Now().Format(time.RFC3339),
			})
		case r.URL.Path == "/api/auth/tokens" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":              "tok-uuid-12345",
					"name":            "ci-runner",
					"token_prefix":    "lot_pat_98765432",
					"workspace_names": []string{"Default Workspace"},
					"is_revoked":      false,
					"created_at":      time.Now().Format(time.RFC3339),
				},
			})
		case r.URL.Path == "/api/auth/tokens/tok-uuid-12345" && r.Method == http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":         true,
				"revoked_id": "tok-uuid-12345",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// Setup mock config and output buffers
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {
				Endpoint: ts.URL,
				Token:    "session-active-token",
			},
		},
	}

	buf := new(bytes.Buffer)
	ui.OutWriter = buf
	ui.ErrWriter = buf
	ui.SetNonInteractive(true)
	ui.SetOutputFormat("json")
	defer ui.SetOutputFormat("")

	// 1. Test tokens create with --json
	createCmd := tokensCreateCmd
	tokenName = "ci-runner"
	tokenWorkspace = "default"
	tokenLifetimeDays = 30
	tokenRole = "write"
	defer func() {
		tokenName = ""
		tokenComment = ""
		tokenWorkspace = ""
		tokenRole = "read"
		tokenLifetimeDays = 30
		tokenIncludeAll = false
		tokenForce = false
	}()

	if err := createCmd.RunE(createCmd, []string{}); err != nil {
		t.Fatalf("tokens create failed: %v", err)
	}

	outStr := buf.String()
	if !strings.Contains(outStr, "lot_pat_9876543210abcdef") {
		t.Errorf("expected output to contain generated token, got: %s", outStr)
	}

	// 2. Test tokens list with --json
	buf.Reset()
	listCmd := tokensListCmd
	tokenIncludeAll = false
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("tokens list failed: %v", err)
	}

	outList := buf.String()
	if !strings.Contains(outList, "tok-uuid-12345") || !strings.Contains(outList, "ci-runner") {
		t.Errorf("expected tokens list to include tok-uuid-12345, got: %s", outList)
	}

	// 3. Test tokens revoke
	buf.Reset()
	revokeCmd := tokensRevokeCmd
	tokenForce = true
	if err := revokeCmd.RunE(revokeCmd, []string{"tok-uuid-12345"}); err != nil {
		t.Fatalf("tokens revoke failed: %v", err)
	}

	outRevoke := buf.String()
	if !strings.Contains(outRevoke, "tok-uuid-12345") {
		t.Errorf("expected revoke output to contain token id, got: %s", outRevoke)
	}
}
