package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := &Config{
		CurrentContext: "test",
		Contexts: map[string]Context{
			"test": {
				Endpoint: "http://localhost:8000",
				Timeout:  15,
			},
		},
	}

	active := cfg.GetActiveContext("")
	if active.Endpoint != "http://localhost:8000" {
		t.Fatalf("expected http://localhost:8000, got %s", active.Endpoint)
	}

	// Test env override
	os.Setenv("ANGLER_ENDPOINT", "http://remote:9000")
	defer os.Unsetenv("ANGLER_ENDPOINT")

	activeEnv := cfg.GetActiveContext("")
	if activeEnv.Endpoint != "http://remote:9000" {
		t.Fatalf("expected env override http://remote:9000, got %s", activeEnv.Endpoint)
	}
}

func TestContextWorkspaceScoping(t *testing.T) {
	cfg := &Config{
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod": {
				Endpoint:  "https://lake.myorg.internal",
				Workspace: "analytics",
				Timeout:   30,
			},
		},
	}

	active := cfg.GetActiveContext("")
	if active.Workspace != "analytics" {
		t.Fatalf("expected workspace 'analytics', got %s", active.Workspace)
	}

	// Test LOT_WORKSPACE env fallback
	os.Setenv("LOT_WORKSPACE", "marketing")
	activeLot := cfg.GetActiveContext("")
	if activeLot.Workspace != "marketing" {
		t.Fatalf("expected LOT_WORKSPACE override 'marketing', got %s", activeLot.Workspace)
	}

	// Test ANGLER_WORKSPACE higher priority than LOT_WORKSPACE
	os.Setenv("ANGLER_WORKSPACE", "finance")
	activeAngler := cfg.GetActiveContext("")
	if activeAngler.Workspace != "finance" {
		t.Fatalf("expected ANGLER_WORKSPACE override 'finance', got %s", activeAngler.Workspace)
	}

	os.Unsetenv("ANGLER_WORKSPACE")
	os.Unsetenv("LOT_WORKSPACE")

	// Test LOT_WORKSPACE_ID and ANGLER_WORKSPACE_ID
	os.Setenv("LOT_WORKSPACE_ID", "lot-id-123")
	if cfg.GetActiveContext("").WorkspaceID != "lot-id-123" {
		t.Fatalf("expected LOT_WORKSPACE_ID 'lot-id-123'")
	}
	os.Setenv("ANGLER_WORKSPACE_ID", "angler-id-456")
	if cfg.GetActiveContext("").WorkspaceID != "angler-id-456" {
		t.Fatalf("expected ANGLER_WORKSPACE_ID 'angler-id-456'")
	}
	os.Unsetenv("ANGLER_WORKSPACE_ID")
	os.Unsetenv("LOT_WORKSPACE_ID")
}
