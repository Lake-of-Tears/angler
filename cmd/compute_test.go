package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestComputeCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundCompute bool
	var foundAdminServices bool
	for _, c := range commands {
		if c.Name() == "compute" {
			foundCompute = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			for _, exp := range []string{"status", "list", "topology", "services", "restart", "stop", "start"} {
				if !subMap[exp] {
					t.Errorf("expected %q subcommand on compute", exp)
				}
			}
		}
		if c.Name() == "admin" {
			for _, sub := range c.Commands() {
				if sub.Name() == "services" {
					foundAdminServices = true
					srvMap := make(map[string]bool)
					for _, sc := range sub.Commands() {
						srvMap[sc.Name()] = true
					}
					for _, exp := range []string{"list", "restart", "stop", "start"} {
						if !srvMap[exp] {
							t.Errorf("expected %q subcommand under admin services", exp)
						}
					}
				}
			}
		}
	}

	if !foundCompute {
		t.Errorf("expected 'compute' command to be registered on rootCmd")
	}
	if !foundAdminServices {
		t.Errorf("expected 'services' subcommand to be registered on adminCmd")
	}
}

func TestPlatformServiceLifecycleControls(t *testing.T) {
	podName := "lake-of-tears-superset-7c8d9-xyz"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/admin/services" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.PlatformServicesListResponse{
				ClusterConnected: true,
				Mode:             "in_cluster",
				Namespace:        "lake-of-tears",
				TotalServices:    2,
				HealthyCount:     2,
				Services: []client.PlatformServiceInfo{
					{
						ID:              "backend",
						Name:            "lake-of-tears-backend",
						DisplayName:     "Lake Backend API",
						Category:        "Core Platform",
						Status:          "running",
						DesiredReplicas: 1,
						ReadyReplicas:   1,
						IsCore:          true,
					},
					{
						ID:              "superset",
						Name:            "lake-of-tears-superset",
						DisplayName:     "Apache Superset",
						Category:        "Analytics",
						Status:          "running",
						DesiredReplicas: 1,
						ReadyReplicas:   1,
						IsCore:          false,
						PodName:         &podName,
					},
				},
			})
		case r.URL.Path == "/api/admin/services/superset/restart" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.PlatformServiceActionResponse{
				Success:    true,
				Service:    "superset",
				Deployment: "lake-of-tears-superset",
				Action:     "restart",
				Message:    "Service 'superset' rollout restart triggered.",
			})
		case r.URL.Path == "/api/admin/services/superset/stop" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.PlatformServiceActionResponse{
				Success:    true,
				Service:    "superset",
				Deployment: "lake-of-tears-superset",
				Action:     "stop",
				Message:    "Service 'superset' scaled to 0 replicas.",
			})
		case r.URL.Path == "/api/admin/services/superset/start" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.PlatformServiceActionResponse{
				Success:    true,
				Service:    "superset",
				Deployment: "lake-of-tears-superset",
				Action:     "start",
				Message:    "Service 'superset' scaled to 1 replica.",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	// 1. compute services
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "compute", "services")
	if code != ui.ExitOK {
		t.Fatalf("compute services failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Apache Superset") || !strings.Contains(stdout, "Lake Backend API") {
		t.Errorf("expected platform services in output, got:\n%s", stdout)
	}

	// 2. admin services list (alias)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "admin", "services", "list")
	if code != ui.ExitOK {
		t.Fatalf("admin services list failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Apache Superset") {
		t.Errorf("expected Apache Superset in admin services list, got:\n%s", stdout)
	}

	// 3. compute restart
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "compute", "restart", "superset")
	if code != ui.ExitOK {
		t.Fatalf("compute restart failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "restart") {
		t.Errorf("expected restart confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. compute stop
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "compute", "stop", "superset")
	if code != ui.ExitOK {
		t.Fatalf("compute stop failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "stop") {
		t.Errorf("expected stop confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 5. admin services start
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "admin", "services", "start", "superset")
	if code != ui.ExitOK {
		t.Fatalf("admin services start failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "start") {
		t.Errorf("expected start confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

