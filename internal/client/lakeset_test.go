package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestLakesetClientMethods(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/lakeset/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-On-Behalf-Of") != "analyst@lake.io" {
			t.Fatalf("expected X-On-Behalf-Of header 'analyst@lake.io', got %q", r.Header.Get("X-On-Behalf-Of"))
		}
		_ = json.NewEncoder(w).Encode(LakesetStatusResponse{
			OK:                  true,
			SupersetOnline:      true,
			DatabaseName:        "Lake of Tears (DuckDB Lakehouse)",
			Engine:              "duckdb",
			WorkspaceSlug:       "default",
			DatasetsCount:       4,
			SyncedDatasetsCount: 4,
			DashboardsCount:     2,
		})
	})

	mux.HandleFunc("/api/v1/lakeset/datasets", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tier") != "gold" {
			t.Fatalf("expected tier=gold query param, got %q", r.URL.Query().Get("tier"))
		}
		_ = json.NewEncoder(w).Encode(LakesetDatasetsResponse{
			OK:            true,
			WorkspaceSlug: "default",
			Total:         1,
			Datasets: []LakesetDataset{
				{
					ID:          "ds-1",
					CatalogSlug: "main",
					SchemaSlug:  "gold",
					Tier:        "gold",
					TableName:   "revenue_summary",
					FullName:    "main.gold.revenue_summary",
					SQLView:     "gold.revenue_summary",
					ColumnCount: 3,
					Synced:      true,
				},
			},
		})
	})

	mux.HandleFunc("/api/v1/lakeset/sync", func(w http.ResponseWriter, r *http.Request) {
		var req LakesetSyncRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !req.Force {
			t.Fatalf("expected Force=true")
		}
		_ = json.NewEncoder(w).Encode(LakesetSyncResponse{
			OK:                  true,
			WorkspaceSlug:       "default",
			DatabaseID:          1,
			DatabaseName:        "Lake of Tears (DuckDB Lakehouse)",
			Engine:              "duckdb",
			SyncedDatasetsCount: 2,
			SyncedAt:            "2026-11-10T12:00:00Z",
		})
	})

	mux.HandleFunc("/api/v1/lakeset/guest-token", func(w http.ResponseWriter, r *http.Request) {
		var req LakesetGuestTokenRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(LakesetGuestTokenResponse{
			OK:            true,
			Token:         "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.guest",
			ExpiresIn:     req.TTLSeconds,
			WorkspaceSlug: "default",
			UserEmail:     "analyst@lake.io",
		})
	})

	mux.HandleFunc("/api/v1/lakeset/explore-data", func(w http.ResponseWriter, r *http.Request) {
		var req LakesetExploreRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(LakesetExploreResponse{
			OK:        true,
			Dataset:   req.Dataset,
			ChartType: req.ChartType,
			Dimension: req.Dimension,
			Metric:    req.Metric,
			SQL:       `SELECT "region", SUM("amount") FROM gold.revenue_summary GROUP BY "region"`,
			Data: map[string]any{
				"categories": []any{"US-East", "EU-West"},
				"series": []any{
					map[string]any{"name": "SUM(amount)", "data": []any{1200.5, 850.0}},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(config.Context{
		Endpoint:   srv.URL,
		Token:      "test-token",
		Workspace:  "default",
		OnBehalfOf: "analyst@lake.io",
	})

	st, err := c.GetLakesetStatus()
	if err != nil {
		t.Fatalf("GetLakesetStatus failed: %v", err)
	}
	if !st.SupersetOnline || st.DatasetsCount != 4 {
		t.Fatalf("unexpected status: %+v", st)
	}

	ds, err := c.ListLakesetDatasets(LakesetDatasetsFilter{Tier: "gold"})
	if err != nil {
		t.Fatalf("ListLakesetDatasets failed: %v", err)
	}
	if len(ds.Datasets) != 1 || ds.Datasets[0].FullName != "main.gold.revenue_summary" {
		t.Fatalf("unexpected datasets: %+v", ds)
	}

	syncRes, err := c.SyncLakeset(LakesetSyncRequest{Force: true})
	if err != nil {
		t.Fatalf("SyncLakeset failed: %v", err)
	}
	if syncRes.SyncedDatasetsCount != 2 {
		t.Fatalf("unexpected sync count: %+v", syncRes)
	}

	gt, err := c.CreateLakesetGuestToken(LakesetGuestTokenRequest{Dataset: "main.gold.revenue_summary", TTLSeconds: 1800})
	if err != nil {
		t.Fatalf("CreateLakesetGuestToken failed: %v", err)
	}
	if gt.ExpiresIn != 1800 || gt.Token == "" {
		t.Fatalf("unexpected guest token: %+v", gt)
	}

	exp, err := c.ExploreLakesetData(LakesetExploreRequest{
		Dataset:   "main.gold.revenue_summary",
		ChartType: "bar",
		Dimension: "region",
		Metric:    "amount",
		MetricAgg: "SUM",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ExploreLakesetData failed: %v", err)
	}
	if exp.Dataset != "main.gold.revenue_summary" || exp.SQL == "" {
		t.Fatalf("unexpected explore response: %+v", exp)
	}
}
