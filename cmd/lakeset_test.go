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

func TestLakesetCommands(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/lakeset/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                    true,
			"superset_online":       true,
			"database_name":         "Lake of Tears (DuckDB Lakehouse)",
			"engine":                "duckdb",
			"workspace_id":          "ws-1",
			"workspace_slug":        "default",
			"workspace_name":        "Default Workspace",
			"role":                  "admin",
			"user_email":            "admin@lake.io",
			"datasets_count":        5,
			"synced_datasets_count": 5,
			"dashboards_count":      2,
		})
	})

	mux.HandleFunc("/api/v1/lakeset/datasets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"workspace_slug": "default",
			"total":          1,
			"datasets": []map[string]any{
				{
					"id":           "ds-1",
					"catalog_slug": "main",
					"schema_slug":  "gold",
					"tier":         "gold",
					"table_name":   "revenue_summary",
					"full_name":    "main.gold.revenue_summary",
					"sql_view":     "gold.revenue_summary",
					"column_count": 4,
					"synced":       true,
				},
			},
		})
	})

	mux.HandleFunc("/api/v1/lakeset/sync", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                    true,
			"workspace_slug":        "default",
			"database_id":           1,
			"database_name":         "Lake of Tears (DuckDB Lakehouse)",
			"engine":                "duckdb",
			"synced_datasets_count": 3,
			"synced_at":             "2026-11-10T12:00:00Z",
		})
	})

	mux.HandleFunc("/api/v1/lakeset/guest-token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"token":          "jwt-guest-token-xyz",
			"expires_in":     3600,
			"workspace_slug": "default",
			"user_email":     "admin@lake.io",
		})
	})

	mux.HandleFunc("/api/v1/lakeset/explore-data", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"dataset":    "main.gold.revenue_summary",
			"chart_type": "bar",
			"dimension":  "region",
			"metric":     "amount",
			"sql":        `SELECT "region", SUM("amount") FROM gold.revenue_summary GROUP BY "region"`,
			"data": map[string]any{
				"categories": []string{"NA", "EU"},
				"series": []map[string]any{
					{"name": "SUM(amount)", "data": []float64{500, 300}},
				},
			},
		})
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

	// 1. status
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "lakeset", "status"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset status failed with code %d: %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "DuckDB Lakehouse") {
		t.Fatalf("expected DuckDB Lakehouse in output, got: %s", outBuf.String())
	}

	// 2. datasets
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "lakeset", "datasets", "--tier", "gold"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset datasets failed with code %d: %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "main.gold.revenue_summary") {
		t.Fatalf("expected dataset in output, got: %s", outBuf.String())
	}

	// 3. sync
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "json", "lakeset", "sync", "--force"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset sync failed with code %d: %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), `"synced_datasets_count": 3`) {
		t.Fatalf("expected synced_datasets_count in JSON output, got: %s", outBuf.String())
	}

	// 4. guest-token --raw
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "lakeset", "guest-token", "--dataset", "main.gold.revenue_summary", "--raw"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset guest-token failed with code %d: %s", code, errBuf.String())
	}
	if strings.TrimSpace(outBuf.String()) != "jwt-guest-token-xyz" {
		t.Fatalf("expected raw token 'jwt-guest-token-xyz', got %q", outBuf.String())
	}

	// 5. explore
	outBuf.Reset()
	lakesetRawToken = false
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "lakeset", "explore", "--dataset", "main.gold.revenue_summary", "--dimension", "region", "--metric", "amount", "--agg", "SUM"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset explore failed with code %d: %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "gold.revenue_summary") {
		t.Fatalf("expected SQL preview in explore output, got: %s", outBuf.String())
	}

	// 6. open --print
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "lakeset", "open", "--dataset", "main.gold.revenue_summary", "--print"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("lakeset open failed with code %d: %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "/lakeset?dataset=main.gold.revenue_summary&tab=explore") {
		t.Fatalf("expected deep-link URL in output, got: %s", outBuf.String())
	}
}
