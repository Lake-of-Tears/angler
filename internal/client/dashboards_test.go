package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestDashboardsClientMethods(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/dashboards/stats" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(DashboardStats{
				TotalDashboards:     4,
				PublishedDashboards: 3,
				TotalCharts:         12,
				SupersetOnline:      true,
			})

		case r.URL.Path == "/api/dashboards" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]Dashboard{
				{
					ID:          "dash-1",
					Title:       "Executive KPI Overview",
					Slug:        "executive-kpi-overview",
					Description: "High-level ARR, revenue and active lakehouse users",
					IsPublished: true,
					ChartsCount: 4,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
				{
					ID:          "dash-2",
					Title:       "Data Pipeline Latency",
					Slug:        "data-pipeline-latency",
					Description: "P95 and P99 ingestion latency metrics",
					IsPublished: false,
					ChartsCount: 2,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			})

		case r.URL.Path == "/api/dashboards" && r.Method == http.MethodPost:
			var req CreateDashboardRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Dashboard{
				ID:           "dash-new",
				Title:        req.Title,
				Slug:         "new-dashboard",
				Description:  req.Description,
				IsPublished:  req.IsPublished,
				ChartsCount:  len(req.LayoutConfig),
				LayoutConfig: req.LayoutConfig,
				CreatedAt:    now,
				UpdatedAt:    now,
			})

		case r.URL.Path == "/api/dashboards/dash-1" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(Dashboard{
				ID:          "dash-1",
				Title:       "Executive KPI Overview",
				Slug:        "executive-kpi-overview",
				Description: "High-level ARR, revenue and active lakehouse users",
				IsPublished: true,
				ChartsCount: 2,
				LayoutConfig: []map[string]any{
					{
						"id":    "w_1",
						"type":  "kpi",
						"title": "Monthly Recurring Revenue",
						"query": "SELECT sum(mrr) FROM gold.subscriptions",
					},
					{
						"id":    "w_2",
						"type":  "bar",
						"title": "Daily Active Users",
						"query": "SELECT date, count(distinct user_id) FROM gold.events GROUP BY 1",
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			})

		case r.URL.Path == "/api/dashboards/dash-1" && r.Method == http.MethodPatch:
			var req UpdateDashboardRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			title := "Executive KPI Overview Updated"
			if req.Title != nil {
				title = *req.Title
			}
			_ = json.NewEncoder(w).Encode(Dashboard{
				ID:          "dash-1",
				Title:       title,
				Slug:        "executive-kpi-overview",
				IsPublished: true,
				ChartsCount: 2,
				CreatedAt:   now,
				UpdatedAt:   now,
			})

		case r.URL.Path == "/api/dashboards/dash-1" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"message": "Dashboard deleted",
			})

		case r.URL.Path == "/api/dashboards/dash-1/widgets/w_1/data" && r.Method == http.MethodGet:
			timeRange := r.URL.Query().Get("time_range")
			_ = json.NewEncoder(w).Encode(WidgetDataResponse{
				WidgetID: "w_1",
				Type:     "kpi",
				Title:    "Monthly Recurring Revenue",
				Data: map[string]any{
					"mrr":        1250000,
					"time_range": timeRange,
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "test-tok",
	})

	// 1. ListDashboards
	list, err := c.ListDashboards()
	if err != nil {
		t.Fatalf("ListDashboards failed: %v", err)
	}
	if len(list) != 2 || list[0].ID != "dash-1" {
		t.Errorf("unexpected dashboards list: %+v", list)
	}

	// 2. GetDashboard
	dash, err := c.GetDashboard("dash-1")
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}
	if dash.Title != "Executive KPI Overview" || len(dash.LayoutConfig) != 2 {
		t.Errorf("unexpected dashboard details: %+v", dash)
	}

	// 3. CreateDashboard
	created, err := c.CreateDashboard(CreateDashboardRequest{
		Title:       "Operations Monitor",
		Description: "Realtime ops monitoring",
		IsPublished: true,
		Template:    "executive",
	})
	if err != nil {
		t.Fatalf("CreateDashboard failed: %v", err)
	}
	if created.ID != "dash-new" || created.Title != "Operations Monitor" {
		t.Errorf("unexpected created dashboard: %+v", created)
	}

	// 4. UpdateDashboard
	newTitle := "Executive KPI Overview Updated"
	updated, err := c.UpdateDashboard("dash-1", UpdateDashboardRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("UpdateDashboard failed: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("unexpected updated title: %s", updated.Title)
	}

	// 5. DeleteDashboard
	if err := c.DeleteDashboard("dash-1"); err != nil {
		t.Fatalf("DeleteDashboard failed: %v", err)
	}

	// 6. GetDashboardStats
	stats, err := c.GetDashboardStats()
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}
	if stats.TotalDashboards != 4 || !stats.SupersetOnline {
		t.Errorf("unexpected stats: %+v", stats)
	}

	// 7. GetWidgetData
	wData, err := c.GetWidgetData("dash-1", "w_1", "last_30_days")
	if err != nil {
		t.Fatalf("GetWidgetData failed: %v", err)
	}
	if wData.WidgetID != "w_1" || wData.Data["time_range"] != "last_30_days" {
		t.Errorf("unexpected widget data: %+v", wData)
	}
}
