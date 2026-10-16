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
