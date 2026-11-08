package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetCatalogTestFlags() {
	catDescription = ""
	catWorkspace = ""
	catPurge = false
	catForce = false
	catCascade = false
	catAnomalySource = ""
	catAnomalyTable = ""
	catAnomalyContam = 0.05
	jsonOutput = false
	outputFlag = ""
}

func executeCatalogCommand(t *testing.T, endpointURL string, args ...string) (string, string, int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut := ui.OutWriter
	origErr := ui.ErrWriter
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	stdoutTTY := false
	ui.SetTTYOverrides(&stdoutTTY, &stdoutTTY)

	resetCatalogTestFlags()

	defer func() {
		ui.OutWriter = origOut
		ui.ErrWriter = origErr
		ui.SetTTYOverrides(nil, nil)
		rootCmd.SetArgs(nil)
		endpoint = ""
		token = ""
	}()

	cmdArgs := []string{"--endpoint", endpointURL, "--token", "test-token"}
	cmdArgs = append(cmdArgs, args...)

	rootCmd.SetArgs(cmdArgs)
	code := ExecuteRoot()
	return outBuf.String(), errBuf.String(), code
}

func TestCatalogCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundCatalog bool
	for _, c := range commands {
		if c.Name() == "catalog" {
			foundCatalog = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			expected := []string{"ls", "get", "create", "delete", "schemas", "tables", "manifest", "anomalies"}
			for _, exp := range expected {
				if !subMap[exp] {
					t.Errorf("expected subcommand %q under catalogCmd", exp)
				}
			}

			// Verify schemas subcommands
			for _, sub := range subCmds {
				if sub.Name() == "schemas" {
					schMap := make(map[string]bool)
					for _, sc := range sub.Commands() {
						schMap[sc.Name()] = true
					}
					for _, exp := range []string{"list", "get", "create", "delete"} {
						if !schMap[exp] {
							t.Errorf("expected schema subcommand %q", exp)
						}
					}
				}
				if sub.Name() == "tables" {
					tblMap := make(map[string]bool)
					for _, tc := range sub.Commands() {
						tblMap[tc.Name()] = true
					}
					for _, exp := range []string{"list", "get", "delete"} {
						if !tblMap[exp] {
							t.Errorf("expected table subcommand %q", exp)
						}
					}
				}
			}
		}
	}

	if !foundCatalog {
		t.Errorf("expected 'catalog' command to be registered on rootCmd")
	}
}

func TestCatalogListAndGetAndCreateAndDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.Catalog{
				{
					ID:        "cat-main",
					Name:      "main",
					Slug:      "main",
					IsDefault: true,
					MyAccess:  "owner",
					CreatedAt: time.Now(),
				},
			})
		case r.URL.Path == "/api/catalogs/cat-main" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.Catalog{
				ID:          "cat-main",
				Name:        "main",
				Slug:        "main",
				Description: "Production lakehouse catalog",
				IsDefault:   true,
				MyAccess:    "owner",
				Schemas: []client.CatalogSchema{
					{Name: "default", Tier: "bronze"},
				},
			})
		case r.URL.Path == "/api/catalogs" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.Catalog{
				ID:   "cat-staging",
				Name: "staging",
				Slug: "staging",
			})
		case r.URL.Path == "/api/catalogs/cat-staging" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
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

	// 1. List catalogs
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "ls")
	if code != ui.ExitOK {
		t.Fatalf("catalog ls failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "main") {
		t.Errorf("expected catalog 'main' in output, got:\n%s", stdout)
	}

	// 2. Get catalog
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "get", "cat-main")
	if code != ui.ExitOK {
		t.Fatalf("catalog get failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Production lakehouse catalog") {
		t.Errorf("expected description in output, got:\n%s", stdout)
	}

	// 3. Create catalog
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "create", "staging", "-d", "Staging environment")
	if code != ui.ExitOK {
		t.Fatalf("catalog create failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "created") && !strings.Contains(stdout, "created") {
		t.Errorf("expected created confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. Delete catalog
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "delete", "cat-staging", "--force")
	if code != ui.ExitOK {
		t.Fatalf("catalog delete failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "deleted") && !strings.Contains(stdout, "deleted") {
		t.Errorf("expected deleted confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

func TestCatalogSchemas(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/cat-main/schemas" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.CatalogSchema{
				{
					ID:          "sch-001",
					CatalogID:   "cat-main",
					Name:        "telemetry",
					Slug:        "telemetry",
					Tier:        "gold",
					Description: "Aggregated metrics",
					CreatedAt:   time.Now(),
				},
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.CatalogSchema{
				ID:        "sch-002",
				CatalogID: "cat-main",
				Name:      "silver",
				Slug:      "silver",
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-001/tables" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.CatalogTable{
				{
					ID:            "tbl-100",
					Name:          "daily_active_users",
					S3PathPattern: "gold/dau/*.parquet",
				},
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-002" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
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

	// 1. Schemas list
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "schemas", "list", "cat-main")
	if code != ui.ExitOK {
		t.Fatalf("schemas list failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "telemetry") || !strings.Contains(stdout, "gold") {
		t.Errorf("expected telemetry schema in output, got:\n%s", stdout)
	}

	// 2. Schemas get
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "schemas", "get", "cat-main", "sch-001")
	if code != ui.ExitOK {
		t.Fatalf("schemas get failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "daily_active_users") {
		t.Errorf("expected daily_active_users in schema get output, got:\n%s", stdout)
	}

	// 3. Schemas create
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "schemas", "create", "cat-main", "silver")
	if code != ui.ExitOK {
		t.Fatalf("schemas create failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "created") && !strings.Contains(stdout, "created") {
		t.Errorf("expected created confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. Schemas delete
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "schemas", "delete", "cat-main", "sch-002", "--force")
	if code != ui.ExitOK {
		t.Fatalf("schemas delete failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "dropped") && !strings.Contains(stdout, "dropped") {
		t.Errorf("expected dropped confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

func TestCatalogTables(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-001/tables" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.CatalogTable{
				{
					ID:            "tbl-101",
					SchemaID:      "sch-001",
					Name:          "user_sessions",
					S3PathPattern: "gold/sessions/*.parquet",
					ColumnDefs: []map[string]interface{}{
						{"name": "user_id", "type": "VARCHAR"},
						{"name": "duration", "type": "INTEGER"},
					},
					SchemaDrift: false,
				},
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-001/tables/tbl-101" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.CatalogTable{
				ID:            "tbl-101",
				SchemaID:      "sch-001",
				Name:          "user_sessions",
				Description:   "User analytical session activity",
				S3PathPattern: "gold/sessions/*.parquet",
				ColumnDefs: []map[string]interface{}{
					{"name": "user_id", "type": "VARCHAR"},
					{"name": "duration", "type": "INTEGER"},
				},
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-001/tables/tbl-101" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
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

	// 1. Tables list
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "tables", "list", "cat-main", "sch-001")
	if code != ui.ExitOK {
		t.Fatalf("tables list failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "user_sessions") {
		t.Errorf("expected user_sessions in table list, got:\n%s", stdout)
	}

	// 2. Tables get
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "get", "cat-main", "sch-001", "tbl-101")
	if code != ui.ExitOK {
		t.Fatalf("tables get failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "user_id") || !strings.Contains(stdout, "duration") {
		t.Errorf("expected columns in table get output, got:\n%s", stdout)
	}

	// 3. Tables delete
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "delete", "cat-main", "sch-001", "tbl-101", "--force")
	if code != ui.ExitOK {
		t.Fatalf("tables delete failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "dropped") && !strings.Contains(stdout, "dropped") {
		t.Errorf("expected dropped confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

func TestCatalogManifestAndAnomalies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/manifest" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"catalog": "main",
					"schemas": []string{"bronze", "silver", "gold"},
				},
			})
		case r.URL.Path == "/api/workspaces/default/anomalies/detect" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.AnomalyDetectResponse{
				OK:        true,
				Workspace: "default",
				Summary: map[string]interface{}{
					"status":             "healthy",
					"anomalies_detected": 0,
				},
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

	// 1. Manifest
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "manifest")
	if code != ui.ExitOK {
		t.Fatalf("manifest failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, `"catalog": "main"`) {
		t.Errorf("expected manifest output, got:\n%s", stdout)
	}

	// 2. Anomalies detect
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "anomalies", "detect", "--contamination", "0.02")
	if code != ui.ExitOK {
		t.Fatalf("anomalies detect failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "healthy") {
		t.Errorf("expected healthy status in anomalies output, got:\n%s", stdout)
	}
}
