package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestContextSetAndListWithWorkspace(t *testing.T) {
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
				Endpoint: "http://localhost:8000",
			},
		},
	}

	buf := new(bytes.Buffer)
	ui.OutWriter = buf
	ui.ErrWriter = buf
	ui.SetNonInteractive(true)
	ui.SetOutputFormat("json")
	defer ui.SetOutputFormat("")

	// 1. Run context set with --workspace
	setCmd := contextSetCmd
	newEndpoint = "https://analytics.lake.internal"
	newWorkspace = "analytics"
	newWorkspaceID = "ws-analytics-123"
	_ = setCmd.Flags().Set("endpoint", newEndpoint)
	_ = setCmd.Flags().Set("workspace", newWorkspace)
	_ = setCmd.Flags().Set("workspace-id", newWorkspaceID)

	defer func() {
		newEndpoint = ""
		newWorkspace = ""
		newWorkspaceID = ""
	}()

	if err := setCmd.RunE(setCmd, []string{"analytics-ctx"}); err != nil {
		t.Fatalf("context set failed: %v", err)
	}

	saved, exists := cfg.Contexts["analytics-ctx"]
	if !exists {
		t.Fatalf("context 'analytics-ctx' was not saved")
	}
	if saved.Workspace != "analytics" {
		t.Errorf("expected saved workspace 'analytics', got %s", saved.Workspace)
	}
	if saved.WorkspaceID != "ws-analytics-123" {
		t.Errorf("expected saved workspace ID 'ws-analytics-123', got %s", saved.WorkspaceID)
	}

	// 2. Run context list
	buf.Reset()
	listCmd := contextListCmd
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("context list failed: %v", err)
	}

	var items []ContextItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse context list output: %v", err)
	}

	var found bool
	for _, item := range items {
		if item.Name == "analytics-ctx" {
			found = true
			if item.Workspace != "analytics" {
				t.Errorf("expected workspace 'analytics', got %s", item.Workspace)
			}
			if item.WorkspaceID != "ws-analytics-123" {
				t.Errorf("expected workspace_id 'ws-analytics-123', got %s", item.WorkspaceID)
			}
		}
	}
	if !found {
		t.Errorf("expected analytics-ctx in context list output")
	}
}

func TestGlobalWorkspaceFlagPropagation(t *testing.T) {
	var capturedSlug string
	var capturedID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSlug = r.Header.Get("X-Workspace-Slug")
		capturedID = r.Header.Get("X-Workspace-Id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.HealthResponse{Status: "healthy"})
	}))
	defer ts.Close()

	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{
		CurrentContext: "default",
		Contexts: map[string]config.Context{
			"default": {
				Endpoint:  ts.URL,
				Workspace: "profile-workspace",
			},
		},
	}

	// 1. Without flag: uses active context workspace
	c := getClient()
	_, err := c.Health()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSlug != "profile-workspace" {
		t.Errorf("expected header 'profile-workspace', got %s", capturedSlug)
	}

	// 2. With global workspace flag override: flag takes precedence over profile
	workspaceFlag = "override-flag-ws"
	defer func() {
		workspaceFlag = ""
		workspaceIDFlag = ""
	}()

	c2 := getClient()
	_, err = c2.Health()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSlug != "override-flag-ws" {
		t.Errorf("expected header 'override-flag-ws', got %s", capturedSlug)
	}

	// 3. With explicit workspace-id flag
	workspaceIDFlag = "id-override-999"
	c3 := getClient()
	_, err = c3.Health()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "id-override-999" {
		t.Errorf("expected X-Workspace-Id 'id-override-999', got %s", capturedID)
	}
}
