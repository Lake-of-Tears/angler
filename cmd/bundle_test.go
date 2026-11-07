package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestBundleInit(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// 1. Default template (python-etl)
	err = runBundleInit("test-lake", "python-etl", "toon")
	if err != nil {
		t.Fatalf("runBundleInit failed: %v", err)
	}

	toonPath := filepath.Join(tempDir, "lake.toon")
	if _, err := os.Stat(toonPath); os.IsNotExist(err) {
		t.Errorf("expected lake.toon to exist, but was not found")
	}

	rawToon, _ := os.ReadFile(toonPath)
	if !strings.Contains(string(rawToon), "name: test-lake") {
		t.Errorf("expected lake.toon to contain bundle name 'test-lake', got: %s", string(rawToon))
	}

	sqlPath := filepath.Join(tempDir, "sql", "clean_orders.sql")
	if _, err := os.Stat(sqlPath); os.IsNotExist(err) {
		t.Errorf("expected sql/clean_orders.sql to exist")
	}

	// 2. Duplicate init should fail
	err = runBundleInit("test-lake", "python-etl", "toon")
	if err == nil {
		t.Errorf("expected error initializing over existing lake.toon, got nil")
	}
}

func TestBundleInitMedallionYAML(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(tempDir)

	err := runBundleInit("medallion-lake", "medallion", "yaml")
	if err != nil {
		t.Fatalf("runBundleInit failed: %v", err)
	}

	yamlPath := filepath.Join(tempDir, "lake.yml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Errorf("expected lake.yml to exist")
	}

	rawYaml, _ := os.ReadFile(yamlPath)
	if !strings.Contains(string(rawYaml), "name: medallion-lake") {
		t.Errorf("expected lake.yml to contain 'name: medallion-lake'")
	}
	if !strings.Contains(string(rawYaml), "bronze") || !strings.Contains(string(rawYaml), "gold") {
		t.Errorf("expected lake.yml to contain medallion tiers")
	}
}

func TestBundleInitRAGSearch(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(tempDir)

	err := runBundleInit("rag-bundle", "rag-search", "toon")
	if err != nil {
		t.Fatalf("runBundleInit failed: %v", err)
	}

	toonPath := filepath.Join(tempDir, "lake.toon")
	rawToon, _ := os.ReadFile(toonPath)
	if !strings.Contains(string(rawToon), "rag-bundle") {
		t.Errorf("expected lake.toon to contain 'rag-bundle'")
	}

	sqlPath := filepath.Join(tempDir, "sql", "embed_documents.sql")
	if _, err := os.Stat(sqlPath); os.IsNotExist(err) {
		t.Errorf("expected sql/embed_documents.sql to exist")
	}
}

func TestLocalValidationValid(t *testing.T) {
	tempDir := t.TempDir()
	manifestContent := `bundle:
  name: customer-analytics
  version: 1.2.0

resources:
  catalogs[1]{name,description}:
    main_lake,Core analytics catalog
`
	manifestPath := filepath.Join(tempDir, "lake.toon")
	_ = os.WriteFile(manifestPath, []byte(manifestContent), 0644)

	err := runLocalValidation(manifestPath, "toon", []byte(manifestContent))
	if err != nil {
		t.Errorf("expected local validation to pass, got error: %v", err)
	}
}

func TestLocalValidationMissingBundleName(t *testing.T) {
	tempDir := t.TempDir()
	manifestContent := `version: 1.2.0
resources:
  catalogs[1]{name}:
    main_lake
`
	manifestPath := filepath.Join(tempDir, "lake.toon")
	_ = os.WriteFile(manifestPath, []byte(manifestContent), 0644)

	err := runLocalValidation(manifestPath, "toon", []byte(manifestContent))
	if err == nil {
		t.Errorf("expected error for missing bundle name, got nil")
	}
}

func TestFindManifestFile(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(tempDir)

	// No file
	_, _, err := findManifestFile("")
	if err == nil {
		t.Errorf("expected error when no manifest exists, got nil")
	}

	// Create lake.yml
	_ = os.WriteFile("lake.yml", []byte("bundle:\n  name: test\n"), 0644)
	path, format, err := findManifestFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "lake.yml" || format != "yaml" {
		t.Errorf("expected lake.yml/yaml, got %s/%s", path, format)
	}

	// Explicit path override
	_ = os.WriteFile("custom.toon", []byte("bundle:\n  name: custom\n"), 0644)
	path2, format2, err := findManifestFile("custom.toon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path2 != "custom.toon" || format2 != "toon" {
		t.Errorf("expected custom.toon/toon, got %s/%s", path2, format2)
	}
}

func TestCollectLocalFiles(t *testing.T) {
	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "sql")
	_ = os.MkdirAll(sqlDir, 0755)
	_ = os.WriteFile(filepath.Join(sqlDir, "test.sql"), []byte("SELECT 1;"), 0644)

	notebooksDir := filepath.Join(tempDir, "notebooks")
	_ = os.MkdirAll(notebooksDir, 0755)
	_ = os.WriteFile(filepath.Join(notebooksDir, "model.py"), []byte("print('hello')"), 0644)

	files := collectLocalFiles(tempDir)
	if len(files) != 2 {
		t.Errorf("expected 2 files collected, got %d: %v", len(files), files)
	}
	if files["sql/test.sql"] != "SELECT 1;" {
		t.Errorf("expected sql/test.sql content 'SELECT 1;', got '%s'", files["sql/test.sql"])
	}
	if files["notebooks/model.py"] != "print('hello')" {
		t.Errorf("expected notebooks/model.py content, got '%s'", files["notebooks/model.py"])
	}
}

// -----------------------------------------------------------------------------
// Mock Server Integration Tests for Client
// -----------------------------------------------------------------------------

func setupMockBundleServer(t *testing.T) (*httptest.Server, *client.Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/bundles/validate":
			var req client.BundleValidateRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := client.BundleValidationResponse{
				Valid:        true,
				BundleName:   "mock-bundle",
				Target:       req.Target,
				DagValidated: true,
				Diff: &client.BundleValidationDiff{
					ToCreate:  []string{"catalog.main_lake", "schema.bronze"},
					ToUpdate:  []string{},
					ToDelete:  []string{},
					Unchanged: []string{},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/api/bundles/deploy":
			var req client.BundleDeployRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp := client.BundleDeployResponse{
				BundleID:            "mock-bundle-uuid-1234",
				BundleName:          "mock-bundle",
				Target:              req.Target,
				Status:              "active",
				Version:             "0.1.0",
				ExecutionDurationMs: 42.5,
				ProvisionedCounts: map[string]int{
					"catalogs":  1,
					"tables":    2,
					"pipelines": 1,
				},
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/bundles":
			resp := client.BundleListResponse{
				Bundles: []client.BundleListItem{
					{
						ID:             "mock-bundle-uuid-1234",
						BundleName:     "mock-bundle",
						Target:         "dev",
						Status:         "active",
						Version:        "0.1.0",
						CatalogsCount:  1,
						TablesCount:    2,
						PipelinesCount: 1,
					},
				},
				Total: 1,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/bundles/mock-bundle-uuid-1234"):
			resp := client.BundleDetailResponse{
				ID:          "mock-bundle-uuid-1234",
				BundleName:  "mock-bundle",
				Target:      "dev",
				Status:      "active",
				Version:     "0.1.0",
				ManifestRaw: "bundle:\n  name: mock-bundle\n",
				Drift: map[string]interface{}{
					"in_sync": true,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/run"):
			resp := client.BundleRunResponse{
				BundleID:     "mock-bundle-uuid-1234",
				BundleName:   "mock-bundle",
				PipelineName: "orders_etl",
				DAGID:        "mock_bundle_orders_etl",
				DAGRunID:     "manual__2026-11-07T12:00:00",
				Status:       "running",
				Message:      "Pipeline triggered",
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/bundles/mock-bundle-uuid-1234"):
			resp := client.BundleDestroyResponse{
				BundleID: "mock-bundle-uuid-1234",
				Status:   "destroyed",
				Message:  "Bundle destroyed",
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))

	c := client.NewClient(config.Context{
		Endpoint: server.URL,
		Token:    "test-jwt-token",
	})
	return server, c
}

func TestClientBundleOperations(t *testing.T) {
	server, c := setupMockBundleServer(t)
	defer server.Close()

	// 1. Validate
	valRes, err := c.ValidateBundle(&client.BundleValidateRequest{
		ManifestContent: "bundle:\n  name: mock-bundle\n",
		ManifestFormat:  "toon",
		Target:          "dev",
	})
	if err != nil {
		t.Fatalf("ValidateBundle failed: %v", err)
	}
	if !valRes.Valid || valRes.BundleName != "mock-bundle" {
		t.Errorf("unexpected ValidateBundle result: %+v", valRes)
	}
	if len(valRes.Diff.ToCreate) != 2 {
		t.Errorf("expected 2 resources to create in diff")
	}

	// 2. Deploy
	depRes, err := c.DeployBundle(&client.BundleDeployRequest{
		ManifestContent: "bundle:\n  name: mock-bundle\n",
		ManifestFormat:  "toon",
		Target:          "dev",
	})
	if err != nil {
		t.Fatalf("DeployBundle failed: %v", err)
	}
	if depRes.BundleID != "mock-bundle-uuid-1234" || depRes.Status != "active" {
		t.Errorf("unexpected DeployBundle result: %+v", depRes)
	}

	// 3. List
	listRes, err := c.ListBundles("dev", "")
	if err != nil {
		t.Fatalf("ListBundles failed: %v", err)
	}
	if listRes.Total != 1 || listRes.Bundles[0].BundleName != "mock-bundle" {
		t.Errorf("unexpected ListBundles result: %+v", listRes)
	}

	// 4. Get Detail
	detailRes, err := c.GetBundle("mock-bundle-uuid-1234")
	if err != nil {
		t.Fatalf("GetBundle failed: %v", err)
	}
	if detailRes.BundleName != "mock-bundle" || detailRes.Drift["in_sync"] != true {
		t.Errorf("unexpected GetBundle result: %+v", detailRes)
	}

	// 5. Run Pipeline
	runRes, err := c.RunBundle("mock-bundle-uuid-1234", "orders_etl")
	if err != nil {
		t.Fatalf("RunBundle failed: %v", err)
	}
	if runRes.Status != "running" || runRes.DAGID != "mock_bundle_orders_etl" {
		t.Errorf("unexpected RunBundle result: %+v", runRes)
	}

	// 6. Destroy
	destRes, err := c.DestroyBundle("mock-bundle-uuid-1234", true, false)
	if err != nil {
		t.Fatalf("DestroyBundle failed: %v", err)
	}
	if destRes.Status != "destroyed" {
		t.Errorf("unexpected DestroyBundle result: %+v", destRes)
	}
}
