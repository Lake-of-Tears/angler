package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetDashboardsFlags() {
	dashTitle = ""
	dashDescription = ""
	dashTemplate = "blank"
	dashSpecFile = ""
	dashPublished = true
	dashForce = false
	dashTimeRange = ""
	jsonOutput = false
	outputFlag = ""
}

func TestDashboardsCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundDashboards bool
	for _, c := range commands {
		if c.Name() == "dashboards" {
			foundDashboards = true
			expectedSubcommands := map[string]bool{
				"list":    false,
				"get":     false,
				"create":  false,
				"delete":  false,
				"stats":   false,
				"widgets": false,
			}
			for _, sub := range c.Commands() {
				if _, ok := expectedSubcommands[sub.Name()]; ok {
					expectedSubcommands[sub.Name()] = true
				}
				if sub.Name() == "widgets" {
					subSubs := map[string]bool{
						"list":     false,
						"get-data": false,
					}
					for _, ss := range sub.Commands() {
						if _, ok := subSubs[ss.Name()]; ok {
							subSubs[ss.Name()] = true
						}
					}
					for sName, found := range subSubs {
						if !found {
							t.Errorf("expected subcommand %q under dashboards widgets", sName)
						}
					}
				}
			}
			for sub, found := range expectedSubcommands {
				if !found {
					t.Errorf("expected subcommand %q registered under dashboards", sub)
				}
			}
		}
	}
	if !foundDashboards {
		t.Fatal("dashboards command not registered on rootCmd")
	}
}

func setupDashboardsMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	now := time.Now().UTC().Truncate(time.Second)

	// List & Create: /api/dashboards
	mux.HandleFunc("/api/dashboards", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			dashboards := []client.Dashboard{
				{
					ID:          "dash-1",
					Title:       "Executive KPI Overview",
					Slug:        "executive-kpi-overview",
					Description: "High-level ARR, revenue and active lakehouse users",
					IsPublished: true,
					ChartsCount: 2,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}
			_ = json.NewEncoder(w).Encode(dashboards)

		case http.MethodPost:
			var req client.CreateDashboardRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(client.Dashboard{
				ID:           "dash-new-123",
				Title:        req.Title,
				Slug:         "new-dashboard",
				Description:  req.Description,
				IsPublished:  req.IsPublished,
				ChartsCount:  len(req.LayoutConfig),
				LayoutConfig: req.LayoutConfig,
				CreatedAt:    now,
				UpdatedAt:    now,
			})

		default:
			http.NotFound(w, r)
		}
	})

	// Stats: /api/dashboards/stats
	mux.HandleFunc("/api/dashboards/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.DashboardStats{
			TotalDashboards:     5,
			PublishedDashboards: 4,
			TotalCharts:         18,
			SupersetOnline:      true,
		})
	})

	// Get & Delete: /api/dashboards/dash-1
	mux.HandleFunc("/api/dashboards/dash-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(client.Dashboard{
				ID:          "dash-1",
				Title:       "Executive KPI Overview",
				Slug:        "executive-kpi-overview",
				Description: "High-level ARR, revenue and active lakehouse users",
				IsPublished: true,
				ChartsCount: 1,
				LayoutConfig: []map[string]any{
					{
						"id":    "w_1",
						"type":  "kpi",
						"title": "Monthly Recurring Revenue",
						"query": "SELECT sum(mrr) FROM gold.subscriptions",
						"width": "half",
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			})

		case http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"message": "Dashboard deleted",
			})

		default:
			http.NotFound(w, r)
		}
	})

	// Widget Data: /api/dashboards/dash-1/widgets/w_1/data
	mux.HandleFunc("/api/dashboards/dash-1/widgets/w_1/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timeRange := r.URL.Query().Get("time_range")
		_ = json.NewEncoder(w).Encode(client.WidgetDataResponse{
			WidgetID: "w_1",
			Type:     "kpi",
			Title:    "Monthly Recurring Revenue",
			Data: map[string]any{
				"mrr":        1250000,
				"time_range": timeRange,
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestDashboardsListAndGet(t *testing.T) {
	ts := setupDashboardsMockServer(t)
	defer ts.Close()

	// 1. List table
	resetDashboardsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "dashboards", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Executive KPI Overview") {
		t.Errorf("expected dashboard title in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "dash-1") {
		t.Errorf("expected dashboard ID in stdout: %s", stdout)
	}

	// 2. List JSON
	resetDashboardsFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "dash", "ls", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var list []client.Dashboard
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("failed to parse dashboards JSON: %v\nstdout: %s", err, stdout)
	}
	if len(list) != 1 || list[0].Slug != "executive-kpi-overview" {
		t.Errorf("unexpected list JSON: %+v", list)
	}

	// 3. Get detail
	resetDashboardsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "dashboard", "get", "dash-1", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Monthly Recurring Revenue") {
		t.Errorf("expected widget in stdout: %s", stdout)
	}
}

func TestDashboardsCreate(t *testing.T) {
	ts := setupDashboardsMockServer(t)
	defer ts.Close()

	// 1. Create with flags
	resetDashboardsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "dashboards", "create", "--title", "Operations Center", "--template", "executive", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Dashboard created successfully") {
		t.Errorf("expected success message: %s", stdout)
	}
	if !strings.Contains(stdout, "dash-new-123") {
		t.Errorf("expected ID in stdout: %s", stdout)
	}

	// 2. Create with JSON spec file
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "dash.json")
	specContent := `{"title":"Sales Performance","description":"Q4 Sales","layout_config":[{"id":"w_sale","type":"bar","title":"Sales By Rep"}]}`
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	resetDashboardsFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "dashboards", "create", "--spec", specFile, "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var created client.Dashboard
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("failed to parse created dashboard JSON: %v\nstdout: %s", err, stdout)
	}
	if created.Title != "Sales Performance" {
		t.Errorf("unexpected created title: %s", created.Title)
	}
}

func TestDashboardsDelete(t *testing.T) {
	ts := setupDashboardsMockServer(t)
	defer ts.Close()

	resetDashboardsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "dashboards", "delete", "dash-1", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "deleted successfully") {
		t.Errorf("expected deletion message: %s", stdout)
	}
}

func TestDashboardsStats(t *testing.T) {
	ts := setupDashboardsMockServer(t)
	defer ts.Close()

	// 1. Text display
	resetDashboardsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "dashboards", "stats", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Total Dashboards:     5") {
		t.Errorf("expected total in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Online") {
		t.Errorf("expected online status: %s", stdout)
	}

	// 2. JSON output
	resetDashboardsFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "dashboards", "stats", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var stats client.DashboardStats
	if err := json.Unmarshal([]byte(stdout), &stats); err != nil {
		t.Fatalf("failed to decode stats JSON: %v\nstdout: %s", err, stdout)
	}
	if stats.TotalDashboards != 5 || stats.TotalCharts != 18 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestDashboardsWidgets(t *testing.T) {
	ts := setupAIMockServer(t)
	_ = ts
	tsDash := setupDashboardsMockServer(t)
	defer tsDash.Close()

	// 1. Widgets list
	resetDashboardsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "dashboards", "widgets", "list", "dash-1", "--endpoint", tsDash.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Monthly Recurring Revenue") {
		t.Errorf("expected widget title: %s", stdout)
	}
	if !strings.Contains(stdout, "w_1") {
		t.Errorf("expected widget id: %s", stdout)
	}

	// 2. Widgets get-data
	resetDashboardsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "dashboards", "widgets", "get-data", "dash-1", "w_1", "--time-range", "last_30_days", "--endpoint", tsDash.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Widget Data: Monthly Recurring Revenue") {
		t.Errorf("expected widget data header: %s", stdout)
	}
	if !strings.Contains(stdout, "1250000") {
		t.Errorf("expected data value: %s", stdout)
	}
}
