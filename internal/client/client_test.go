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
