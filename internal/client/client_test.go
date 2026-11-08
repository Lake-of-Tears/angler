package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestClientHealth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  "healthy",
			Version: "1.0.0",
			Services: map[string]string{
				"duckdb": "ready",
				"minio":  "ready",
			},
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Timeout:  5,
	})

	health, err := c.Health()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", health.Status)
	}
	if health.Services["duckdb"] != "ready" {
		t.Errorf("expected duckdb service ready")
	}
}

func TestClientQuery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(QueryResponse{
			Columns:  []string{"id", "metric"},
			Rows:     [][]interface{}{{1, "latency"}, {2, "throughput"}},
			RowCount: 2,
			Duration: "12ms",
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Timeout:  5,
	})

	res, err := c.Query("SELECT 1")
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}

	if res.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", res.RowCount)
	}
	if len(res.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(res.Columns))
	}
}

func TestClientClusterStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cluster/status" {
			t.Errorf("expected /api/cluster/status, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ClusterStatusResponse{
			Status:           "healthy",
			ClusterConnected: true,
			Mode:             "in_cluster",
			Namespace:        "lake-of-tears",
			TotalServices:    2,
			HealthyCount:     2,
			Services: map[string]ServiceHealthDetail{
				"duckdb": {
					Name:        "duckdb",
					DisplayName: "DuckDB Analytical Engine",
					Status:      "running",
					Category:    "Analytics",
				},
				"minio": {
					Name:        "minio",
					DisplayName: "MinIO Object Storage",
					Status:      "running",
					Category:    "Storage",
				},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Timeout:  5,
	})

	status, err := c.ClusterStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", status.Status)
	}
	if status.Services["duckdb"].Status != "running" {
		t.Errorf("expected duckdb running")
	}
}

func TestClientClusterTopology(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cluster/topology" {
			t.Errorf("expected /api/cluster/topology, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ClusterTopologyResponse{
			Namespace: "lake-of-tears",
			Mode:      "kubectl_cli",
			IngressRoutes: []IngressRoute{
				{
					Service:    "lakeui",
					URL:        "http://localhost/",
					Host:       "localhost",
					Path:       "/",
					TLSEnabled: false,
				},
			},
			StorageVolumes: []StorageVolume{
				{
					Name:      "minio-data",
					MountPath: "/data",
				},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Timeout:  5,
	})

	topology, err := c.ClusterTopology()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if topology.Namespace != "lake-of-tears" {
		t.Errorf("expected namespace 'lake-of-tears', got %s", topology.Namespace)
	}
	if len(topology.IngressRoutes) != 1 {
		t.Errorf("expected 1 ingress route, got %d", len(topology.IngressRoutes))
	}
	if len(topology.StorageVolumes) != 1 {
		t.Errorf("expected 1 storage volume, got %d", len(topology.StorageVolumes))
	}
}

func TestWorkspaceHeaderPropagation(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		workspaceID  string
		expectedSlug string
		expectedID   string
	}{
		{
			name:         "default workspace when none specified",
			workspace:    "",
			workspaceID:  "",
			expectedSlug: "default",
			expectedID:   "",
		},
		{
			name:         "named slug workspace",
			workspace:    "analytics",
			workspaceID:  "",
			expectedSlug: "analytics",
			expectedID:   "",
		},
		{
			name:         "explicit workspace ID",
			workspace:    "marketing",
			workspaceID:  "ws-id-12345",
			expectedSlug: "marketing",
			expectedID:   "ws-id-12345",
		},
		{
			name:         "UUID workspace auto-detection",
			workspace:    "123e4567-e89b-12d3-a456-426614174000",
			workspaceID:  "",
			expectedSlug: "123e4567-e89b-12d3-a456-426614174000",
			expectedID:   "123e4567-e89b-12d3-a456-426614174000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedSlug string
			var capturedID string

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedSlug = r.Header.Get("X-Workspace-Slug")
				capturedID = r.Header.Get("X-Workspace-Id")
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
			}))
			defer ts.Close()

			c := NewClient(config.Context{
				Endpoint:    ts.URL,
				Timeout:     5,
				Workspace:   tt.workspace,
				WorkspaceID: tt.workspaceID,
			})

			_, err := c.Health()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedSlug != tt.expectedSlug {
				t.Errorf("expected X-Workspace-Slug %q, got %q", tt.expectedSlug, capturedSlug)
			}
			if capturedID != tt.expectedID {
				t.Errorf("expected X-Workspace-Id %q, got %q", tt.expectedID, capturedID)
			}
		})
	}
}
