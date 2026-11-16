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
	catName = ""
	catS3Path = ""
	catWorkspace = ""
	catPurge = false
	catForce = false
	catCascade = false
	catAnomalySource = ""
	catAnomalyTable = ""
	catAnomalyContam = 0.05
	catRestoreStrat = ""
	catRestoreNewName = ""
	catAccessMode = "read"
	catAccessMessage = "Requested via Angler CLI"
	catToonRaw = false
	catDryRun = false
	catSkipVerify = false
	catToParquet = false
	catToIceberg = false
	catMaterialize = false
	jsonOutput = false
	yamlOutput = false
	toonOutput = false
	outputFlag = ""
	assumeYes = false
	forceExec = false
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

			expected := []string{
				"ls", "get", "create", "update", "delete",
				"migrate", "rollback",
				"schemas", "tables", "toon", "trash", "undrop",
				"directory", "access", "manifest", "anomalies",
			}
			for _, exp := range expected {
				if !subMap[exp] {
					t.Errorf("expected subcommand %q under catalogCmd", exp)
				}
			}

			// Verify schemas, tables, trash, and access subcommands
			for _, sub := range subCmds {
				if sub.Name() == "schemas" {
					schMap := make(map[string]bool)
					for _, sc := range sub.Commands() {
						schMap[sc.Name()] = true
					}
					for _, exp := range []string{"list", "get", "create", "update", "delete"} {
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
					for _, exp := range []string{"list", "get", "update", "delete", "migrate", "rollback"} {
						if !tblMap[exp] {
							t.Errorf("expected table subcommand %q", exp)
						}
					}
				}
				if sub.Name() == "trash" {
					trashMap := make(map[string]bool)
					for _, tc := range sub.Commands() {
						trashMap[tc.Name()] = true
					}
					for _, exp := range []string{"list", "restore", "purge", "purge-expired"} {
						if !trashMap[exp] {
							t.Errorf("expected trash subcommand %q", exp)
						}
					}
				}
				if sub.Name() == "access" {
					accMap := make(map[string]bool)
					for _, ac := range sub.Commands() {
						accMap[ac.Name()] = true
					}
					for _, exp := range []string{"list", "request", "approve", "reject", "revoke"} {
						if !accMap[exp] {
							t.Errorf("expected access subcommand %q", exp)
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

func TestCatalogTrashAndUndrop(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/trash" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.DeletedCatalogAsset{
				{
					ID:            "trash-001",
					CatalogName:   "main",
					SchemaName:    "gold",
					AssetType:     "table",
					Name:          "dropped_orders",
					Slug:          "dropped_orders",
					DroppedAt:     time.Now(),
					DaysRemaining: 6.8,
				},
			})
		case r.URL.Path == "/api/catalogs/trash/trash-001/restore" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":          true,
				"restored_id": "trash-001",
			})
		case r.URL.Path == "/api/catalogs/trash/trash-001/purge" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":        true,
				"purged_id": "trash-001",
			})
		case r.URL.Path == "/api/catalogs/trash/purge-expired" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":           true,
				"purged_count": 2,
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

	// 1. Trash list
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "trash", "list")
	if code != ui.ExitOK {
		t.Fatalf("catalog trash list failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "dropped_orders") || !strings.Contains(stdout, "trash-001") {
		t.Errorf("expected dropped_orders in trash list, got:\n%s", stdout)
	}

	// 2. Trash restore
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "trash", "restore", "trash-001")
	if code != ui.ExitOK {
		t.Fatalf("catalog trash restore failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "restored") {
		t.Errorf("expected restored confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 3. Top-level undrop alias
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "undrop", "trash-001", "--strategy", "rename", "--new-name", "orders_restored")
	if code != ui.ExitOK {
		t.Fatalf("catalog undrop failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "restored") {
		t.Errorf("expected restored confirmation from undrop, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. Trash purge
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "trash", "purge", "trash-001", "--force")
	if code != ui.ExitOK {
		t.Fatalf("catalog trash purge failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "purged") {
		t.Errorf("expected purged confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 5. Trash purge-expired
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "trash", "purge-expired", "--force")
	if code != ui.ExitOK {
		t.Fatalf("catalog trash purge-expired failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "purged") {
		t.Errorf("expected purged confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

func TestCatalogUpdateAndTOON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/cat-main" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(client.Catalog{
				ID:          "cat-main",
				Name:        "main",
				Slug:        "main",
				Description: "Updated catalog desc",
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-gold" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(client.CatalogSchema{
				ID:          "sch-gold",
				CatalogID:   "cat-main",
				Name:        "gold",
				Description: "Updated schema desc",
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-gold/tables/tbl-users" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(client.CatalogTable{
				ID:          "tbl-users",
				SchemaID:    "sch-gold",
				Name:        "users",
				Description: "Updated table desc",
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-gold/toon" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.TOONExportResponse{
				OK:      true,
				TOON:    "schema: gold\ncatalog: main\ntables[1]{name}:\n  users",
				Catalog: "main",
				Schema:  "gold",
			})
		case r.URL.Path == "/api/catalogs/cat-main/schemas/sch-gold/tables/tbl-users/toon" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.TOONExportResponse{
				OK:      true,
				TOON:    "table: users\nschema: gold\ncolumns[2]{name,type}:\n  id,UUID\n  email,VARCHAR",
				Catalog: "main",
				Schema:  "gold",
				Table:   "users",
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

	// 1. Update catalog
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "update", "cat-main", "--description", "Updated catalog desc")
	if code != ui.ExitOK {
		t.Fatalf("catalog update failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Updated catalog desc") {
		t.Errorf("expected updated catalog description, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 2. Update schema
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "schemas", "update", "cat-main", "sch-gold", "--description", "Updated schema desc")
	if code != ui.ExitOK {
		t.Fatalf("catalog schemas update failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Updated schema desc") {
		t.Errorf("expected updated schema description, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 3. Update table
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "update", "cat-main", "sch-gold", "tbl-users", "--description", "Updated table desc")
	if code != ui.ExitOK {
		t.Fatalf("catalog tables update failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Updated table desc") {
		t.Errorf("expected updated table description, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. Schema TOON export
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "toon", "cat-main", "sch-gold")
	if code != ui.ExitOK {
		t.Fatalf("catalog toon schema failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "schema: gold") {
		t.Errorf("expected TOON schema output, got:\n%s", stdout)
	}

	// 5. Table TOON export (--raw)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "toon", "cat-main", "sch-gold", "tbl-users", "--raw")
	if code != ui.ExitOK {
		t.Fatalf("catalog toon table failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "table: users") || !strings.Contains(stdout, "email,VARCHAR") {
		t.Errorf("expected TOON table output, got:\n%s", stdout)
	}
}

func TestCatalogDirectoryAndAccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/workspaces" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.Workspace{
				{ID: "ws-default", Name: "default", Slug: "default"},
			})
		case r.URL.Path == "/api/catalogs/directory" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.CatalogDirectoryItem{
				{
					ID:                 "cat-finance",
					Name:               "finance_lake",
					Slug:               "finance_lake",
					OwnerWorkspaceID:   "ws-finance",
					OwnerWorkspaceName: "Finance Workspace",
					SchemaCount:        3,
					AccessStatus:       "approved",
					AccessID:           "acc-100",
				},
			})
		case r.URL.Path == "/api/catalogs/cat-finance/access" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.CatalogAccessItem{
				{
					ID:               "acc-100",
					CatalogID:        "cat-finance",
					WorkspaceID:      "ws-default",
					WorkspaceName:    "default",
					Mode:             "read",
					Status:           "pending",
					RequestedByEmail: "analyst@lake.io",
					RequestedAt:      time.Now(),
				},
			})
		case r.URL.Path == "/api/catalogs/cat-finance/access/request" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/catalogs/cat-finance/access/acc-100" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/catalogs/cat-finance/access/acc-100" && r.Method == http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
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

	// 1. Directory list
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "directory")
	if code != ui.ExitOK {
		t.Fatalf("catalog directory failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "finance_lake") || !strings.Contains(stdout, "approved") {
		t.Errorf("expected finance_lake in directory output, got:\n%s", stdout)
	}

	// 2. Access list
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "access", "list", "cat-finance")
	if code != ui.ExitOK {
		t.Fatalf("catalog access list failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "acc-100") || !strings.Contains(stdout, "analyst@lake.io") {
		t.Errorf("expected acc-100 in access list output, got:\n%s", stdout)
	}

	// 3. Access request
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "access", "request", "cat-finance", "--message", "Need Q4 metrics")
	if code != ui.ExitOK {
		t.Fatalf("catalog access request failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, `"ok": true`) {
		t.Errorf("expected ok confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 4. Access approve
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "access", "approve", "cat-finance", "acc-100")
	if code != ui.ExitOK {
		t.Fatalf("catalog access approve failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, `"ok": true`) {
		t.Errorf("expected ok confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 5. Access reject
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "access", "reject", "cat-finance", "acc-100")
	if code != ui.ExitOK {
		t.Fatalf("catalog access reject failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, `"ok": true`) {
		t.Errorf("expected ok confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 6. Access revoke
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "access", "revoke", "cat-finance", "acc-100", "--force")
	if code != ui.ExitOK {
		t.Fatalf("catalog access revoke failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "revoked") {
		t.Errorf("expected revoked confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}

func TestCatalogMigrationCommands(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables/tbl-001/migrate-to-iceberg" && r.Method == http.MethodPost:
			var req client.MigrateTableRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.DryRun {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":             true,
					"status":         "planned",
					"table":          "users",
					"files":          4,
					"rows":           1000,
					"bytes":          50000,
					"partitioned_by": []string{"region"},
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":          true,
					"status":      "migrated",
					"table":       "users",
					"table_id":    "tbl-001",
					"format":      "iceberg",
					"snapshot_id": 987654321,
				})
			}
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables/tbl-001/migrate-to-parquet" && r.Method == http.MethodPost:
			var req client.RollbackTableRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.DryRun {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":              true,
					"status":          "planned",
					"table":           "users",
					"type":            "zero_copy",
					"target_format":   "parquet",
					"s3_path_pattern": "s3://lake/users/*.parquet",
				})
			} else if req.Materialize {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":              true,
					"status":          "converted",
					"table":           "users",
					"table_id":        "tbl-001",
					"format":          "parquet",
					"type":            "materialized",
					"s3_path_pattern": "s3://lake/users/parquet_export/*.parquet",
					"rows":            1000,
					"bytes":           50000,
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":              true,
					"status":          "rolled_back",
					"table":           "users",
					"table_id":        "tbl-001",
					"format":          "parquet",
					"type":            "zero_copy",
					"s3_path_pattern": "s3://lake/users/*.parquet",
				})
			}
		case r.URL.Path == "/api/catalogs/cat-001/migrate-to-iceberg" && r.Method == http.MethodPost:
			var req client.MigrateTableRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			statusKey := "migrated"
			if req.DryRun {
				statusKey = "planned"
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":      true,
				"catalog": "main",
				"summary": map[string]interface{}{
					statusKey: 3,
					"skipped": 0,
					"failed":  0,
				},
			})
		case r.URL.Path == "/api/catalogs/cat-001/migrate-to-parquet" && r.Method == http.MethodPost:
			var req client.RollbackTableRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			statusKey := "rolled_back"
			if req.DryRun {
				statusKey = "planned"
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":      true,
				"catalog": "main",
				"summary": map[string]interface{}{
					statusKey: 3,
					"skipped": 0,
					"failed":  0,
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

	// 1. Table migrate dry-run (table mode)
	stdout, stderr, code := executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "--dry-run", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table migrate dry-run failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "users") || !strings.Contains(stdout, "planned") {
		t.Errorf("expected planned in table migrate dry-run, got:\n%s", stdout)
	}

	// 2. Table migrate execute (-y, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table migrate failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "migrated") || !strings.Contains(stdout+stderr, "987654321") {
		t.Errorf("expected migration success confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 3. Table migrate JSON output (--json)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "-y", "--json")
	if code != ui.ExitOK {
		t.Fatalf("table migrate json failed: code %d, stderr: %s", code, stderr)
	}
	var jsonRes map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &jsonRes); err != nil {
		t.Fatalf("invalid json output: %v\nstdout: %s", err, stdout)
	}
	if jsonRes["status"] != "migrated" || jsonRes["format"] != "iceberg" {
		t.Errorf("expected migrated iceberg in json, got: %v", jsonRes)
	}

	// 4. Table migrate YAML output (--yaml)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "-y", "--yaml")
	if code != ui.ExitOK {
		t.Fatalf("table migrate yaml failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "format: iceberg") || !strings.Contains(stdout, "status: migrated") {
		t.Errorf("expected yaml output with format: iceberg, got:\n%s", stdout)
	}

	// 5. Table migrate TOON output (--toon)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "-y", "--toon")
	if code != ui.ExitOK {
		t.Fatalf("table migrate toon failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "format: iceberg") && !strings.Contains(stdout, "status: migrated") {
		t.Errorf("expected toon output with format/status, got:\n%s", stdout)
	}

	// 6. Table rollback dry-run (table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "rollback", "cat-001", "sch-001", "tbl-001", "--dry-run", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table rollback dry-run failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "planned") || !strings.Contains(stdout, "zero_copy") {
		t.Errorf("expected zero_copy planned in rollback dry-run, got:\n%s", stdout)
	}

	// 7. Table rollback zero-copy execute (-y, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "rollback", "cat-001", "sch-001", "tbl-001", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table rollback failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "rolled back") {
		t.Errorf("expected rolled back confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 8. Table rollback materialized execute (-y --materialize, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "rollback", "cat-001", "sch-001", "tbl-001", "-y", "--materialize", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table rollback materialize failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "converted to Parquet via materialized export") {
		t.Errorf("expected materialized export confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 9. Table migrate cross-command with --to-parquet (table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "tables", "migrate", "cat-001", "sch-001", "tbl-001", "--to-parquet", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("table migrate --to-parquet failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "rolled back") {
		t.Errorf("expected rolled back confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 10. Catalog-wide migrate dry-run (table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "migrate", "cat-001", "--dry-run", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("catalog migrate dry-run failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "PLANNED") {
		t.Errorf("expected PLANNED in catalog migrate dry-run, got:\n%s", stdout)
	}

	// 11. Catalog-wide migrate execute (-y, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "migrate", "cat-001", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("catalog migrate failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Migration to Iceberg completed") {
		t.Errorf("expected migration completed confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 12. Catalog-wide rollback execute (-y, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "rollback", "cat-001", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("catalog rollback failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Rollback to Parquet completed") {
		t.Errorf("expected rollback completed confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 13. Catalog-wide migrate cross-command with --to-parquet (-y, table mode)
	stdout, stderr, code = executeCatalogCommand(t, ts.URL, "catalog", "migrate", "cat-001", "--to-parquet", "-y", "-o", "table")
	if code != ui.ExitOK {
		t.Fatalf("catalog migrate --to-parquet failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout+stderr, "Rollback to Parquet completed") {
		t.Errorf("expected rollback completed confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}
