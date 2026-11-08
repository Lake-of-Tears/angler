package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestAuthLoginWithToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/me" {
			if r.Header.Get("Authorization") != "Bearer lot_pat_valid_secret" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"detail":"Invalid personal access token"}`))
				return
			}
			w.Write([]byte(`{"id":"user-111","email":"pat-user@lakeoftears.io","display_name":"PAT User","role":"admin","is_active":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Temp home dir for isolated config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{
		CurrentContext: "default",
		Contexts: map[string]config.Context{
			"default": {
				Endpoint: ts.URL,
			},
		},
	}

	buf := new(bytes.Buffer)
	ui.OutWriter = buf
	ui.ErrWriter = buf
	ui.SetNonInteractive(true)
	ui.SetOutputFormat("json")
	defer ui.SetOutputFormat("")

	loginToken = "lot_pat_valid_secret"
	loginTokenStdin = false
	loginEmail = ""
	loginPassword = ""
	defer func() {
		loginToken = ""
		loginTokenStdin = false
		loginEmail = ""
		loginPassword = ""
	}()

	if err := loginCmd.RunE(loginCmd, []string{}); err != nil {
		t.Fatalf("auth login with token failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "pat-user@lakeoftears.io") {
		t.Errorf("expected login output to contain user email, got: %s", out)
	}
	if !strings.Contains(out, "personal_access_token") {
		t.Errorf("expected auth_method personal_access_token, got: %s", out)
	}

	// Verify saved config
	savedPath := filepath.Join(tmpDir, ".angler", "config.yaml")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected config file to be saved at %s: %v", savedPath, err)
	}

	savedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if savedCfg.Contexts["default"].Token != "lot_pat_valid_secret" {
		t.Errorf("expected token saved in config, got: %s", savedCfg.Contexts["default"].Token)
	}

	// Test auth status
	buf.Reset()
	if err := authStatusCmd.RunE(authStatusCmd, []string{}); err != nil {
		t.Fatalf("auth status failed: %v", err)
	}

	statusOut := buf.String()
	if !strings.Contains(statusOut, "personal_access_token") {
		t.Errorf("expected token_type personal_access_token in status, got: %s", statusOut)
	}
}
