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

	scriptsDir := filepath.Join(tempDir, "scripts")
	_ = os.MkdirAll(scriptsDir, 0755)
	_ = os.WriteFile(filepath.Join(scriptsDir, "ingest.py"), []byte("import pandas"), 0644)

	pythonDir := filepath.Join(tempDir, "python")
	_ = os.MkdirAll(pythonDir, 0755)
	_ = os.WriteFile(filepath.Join(pythonDir, "transform.py"), []byte("import httpx"), 0644)

	srcDir := filepath.Join(tempDir, "src")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "requirements.txt"), []byte("pandas>=2.0\nhttpx\n"), 0644)

	files := collectLocalFiles(tempDir)
	if len(files) != 5 {
		t.Errorf("expected 5 files collected, got %d: %v", len(files), files)
	}
	if files["sql/test.sql"] != "SELECT 1;" {
		t.Errorf("expected sql/test.sql content 'SELECT 1;', got '%s'", files["sql/test.sql"])
	}
	if files["notebooks/model.py"] != "print('hello')" {
		t.Errorf("expected notebooks/model.py content, got '%s'", files["notebooks/model.py"])
	}
	if files["scripts/ingest.py"] != "import pandas" {
		t.Errorf("expected scripts/ingest.py content, got '%s'", files["scripts/ingest.py"])
	}
	if files["python/transform.py"] != "import httpx" {
		t.Errorf("expected python/transform.py content, got '%s'", files["python/transform.py"])
	}
	if files["src/requirements.txt"] != "pandas>=2.0\nhttpx\n" {
		t.Errorf("expected src/requirements.txt content, got '%s'", files["src/requirements.txt"])
	}
}

func TestBundleInitWebhookETL(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(tempDir)

	err := runBundleInit("webhook-lake", "webhook-etl", "yaml")
	if err != nil {
		t.Fatalf("runBundleInit webhook-etl failed: %v", err)
	}

	rawYaml, err := os.ReadFile(filepath.Join(tempDir, "lake.yml"))
	if err != nil {
		t.Fatalf("expected lake.yml to exist: %v", err)
	}
	content := string(rawYaml)
	if !strings.Contains(content, `schedule: "@webhook"`) || !strings.Contains(content, "webhook_enabled: true") {
		t.Errorf("expected @webhook schedule and webhook_enabled in lake.yml, got:\n%s", content)
	}
	if !strings.Contains(content, "type: python_script") || !strings.Contains(content, "pandas>=2.0") {
		t.Errorf("expected python_script task and virtualenv requirements in lake.yml, got:\n%s", content)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "scripts", "process_webhook.py")); os.IsNotExist(err) {
		t.Errorf("expected scripts/process_webhook.py to be created")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "sql", "transform_webhook_events.sql")); os.IsNotExist(err) {
		t.Errorf("expected sql/transform_webhook_events.sql to be created")
	}

	// Verify local validation passes on generated webhook-etl YAML
	if err := runLocalValidation(filepath.Join(tempDir, "lake.yml"), "yaml", rawYaml); err != nil {
		t.Errorf("expected local validation to pass on webhook-etl YAML: %v", err)
	}
}

func TestLocalValidationPipelinesCyclesAndRequirements(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Valid pipeline with @webhook and python_script requirements
	validYAML := `bundle:
  name: valid-webhook-bundle
  version: 0.1.0
resources:
  pipelines:
    ingest_pipe:
      schedule: "@webhook"
      webhook_enabled: true
      tasks:
        - name: step_one
          type: python_script
          entrypoint: scripts/step1.py
          requirements:
            - "pandas>=2.0"
            - "httpx"
        - name: step_two
          type: duckdb_sql
          entrypoint: sql/step2.sql
          upstream_tasks:
            - step_one
`
	if err := runLocalValidation(filepath.Join(tempDir, "valid.yml"), "yaml", []byte(validYAML)); err != nil {
		t.Fatalf("expected valid webhook + requirements pipeline to pass, got: %v", err)
	}

	// 2. Cyclic dependency between tasks should fail local validation
	cyclicYAML := `bundle:
  name: cyclic-bundle
resources:
  pipelines:
    bad_pipe:
      schedule: "0 1 * * *"
      tasks:
        - name: task_a
          type: duckdb_sql
          entrypoint: sql/a.sql
          upstream_tasks: [task_b]
        - name: task_b
          type: duckdb_sql
          entrypoint: sql/b.sql
          upstream_tasks: [task_a]
`
	if err := runLocalValidation(filepath.Join(tempDir, "cyclic.yml"), "yaml", []byte(cyclicYAML)); err == nil {
		t.Errorf("expected error for cyclic task dependency, got nil")
	}

	// 3. Unknown upstream task should fail local validation
	unknownUpstreamYAML := `bundle:
  name: unknown-upstream-bundle
resources:
  pipelines:
    bad_pipe:
      tasks:
        - name: task_a
          type: duckdb_sql
          entrypoint: sql/a.sql
          upstream_tasks: [non_existent_task]
`
	if err := runLocalValidation(filepath.Join(tempDir, "unknown.yml"), "yaml", []byte(unknownUpstreamYAML)); err == nil {
		t.Errorf("expected error for unknown upstream task, got nil")
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
					ToCreate:  []interface{}{"catalog.main_lake", "schema.bronze"},
					ToUpdate:  []interface{}{},
					ToDelete:  []interface{}{},
					Unchanged: []interface{}{},
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
			includeDrift := r.URL.Query().Get("include_drift") != "false"
			var drift map[string]interface{}
			if includeDrift {
				drift = map[string]interface{}{
					"in_sync":       true,
					"total_checked": 4,
					"missing_count": 0,
				}
			}
			resp := client.BundleDetailResponse{
				ID:              "mock-bundle-uuid-1234",
				BundleName:      "mock-bundle",
				Target:          "dev",
				Status:          "active",
				Version:         "0.1.0",
				GitCommitSHA:    "adbc4d1",
				CatalogsCount:   1,
				TablesCount:     2,
				PipelinesCount:  1,
				DashboardsCount: 0,
				ManifestRaw:     "bundle:\n  name: mock-bundle\n",
				Drift:           drift,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/bundles/drifted-bundle-uuid"):
			resp := client.BundleDetailResponse{
				ID:             "drifted-bundle-uuid",
				BundleName:     "drifted-bundle",
				Target:         "prod",
				Status:         "pending_drift_sync",
				Version:        "0.2.0",
				CatalogsCount:  1,
				TablesCount:    2,
				PipelinesCount: 1,
				ManifestRaw:    "bundle:\n  name: drifted-bundle\n",
				Drift: map[string]interface{}{
					"in_sync":       false,
					"total_checked": 4,
					"missing_count": 1,
					"missing_resources": map[string]interface{}{
						"tables": []string{"main_lake.silver.orders"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/run"):
			var runReq client.BundleRunRequest
			_ = json.NewDecoder(r.Body).Decode(&runReq)
			msg := "Pipeline triggered"
			if runReq.Conf != nil && runReq.Conf["batch_date"] != nil {
				msg = "Pipeline triggered with conf batch_date=" + runReq.Conf["batch_date"].(string)
			}
			resp := client.BundleRunResponse{
				BundleID:     "mock-bundle-uuid-1234",
				BundleName:   "mock-bundle",
				PipelineName: "orders_etl",
				DAGID:        "mock_bundle_orders_etl",
				DAGRunID:     "manual__2026-11-07T12:00:00",
				Status:       "running",
				Message:      msg,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/api/pipelines/mock_bundle_orders_etl/runs/manual__2026-11-07T12:00:00"):
			dur := 12.4
			_ = json.NewEncoder(w).Encode(client.PipelineRun{
				DagRunID:        "manual__2026-11-07T12:00:00",
				State:           "success",
				DurationSeconds: &dur,
			})

		case r.Method == http.MethodPost && r.URL.Path == "/api/bundles/generate-ai":
			var genReq client.BundleGenerateRequest
			_ = json.NewDecoder(r.Body).Decode(&genReq)
			resp := client.BundleGenerateResponse{
				Manifest: "# Lake Asset Bundle (TOON Specification)\nbundle:\n  name: ai-generated-lake\n  version: 0.1.0\n",
				Format:   "toon",
				Model:    "gemini-2.5-flash",
				Valid:    true,
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

	// 5. Run Pipeline with Conf
	runRes, err := c.RunBundle("mock-bundle-uuid-1234", "orders_etl", map[string]interface{}{"batch_date": "2026-11-10"})
	if err != nil {
		t.Fatalf("RunBundle failed: %v", err)
	}
	if runRes.Status != "running" || runRes.DAGID != "mock_bundle_orders_etl" || !strings.Contains(runRes.Message, "2026-11-10") {
		t.Errorf("unexpected RunBundle result: %+v", runRes)
	}

	// 6. GetBundleWithOptions (--no-drift)
	noDriftRes, err := c.GetBundleWithOptions("mock-bundle-uuid-1234", false)
	if err != nil {
		t.Fatalf("GetBundleWithOptions failed: %v", err)
	}
	if noDriftRes.Drift != nil {
		t.Errorf("expected nil drift when includeDrift=false, got: %+v", noDriftRes.Drift)
	}

	// 7. GenerateBundleAI
	genRes, err := c.GenerateBundleAI("Build an AI telemetry pipeline", "toon")
	if err != nil {
		t.Fatalf("GenerateBundleAI failed: %v", err)
	}
	if !genRes.Valid || !strings.Contains(genRes.Manifest, "ai-generated-lake") {
		t.Errorf("unexpected GenerateBundleAI result: %+v", genRes)
	}

	// 8. Destroy
	destRes, err := c.DestroyBundle("mock-bundle-uuid-1234", true, false)
	if err != nil {
		t.Fatalf("DestroyBundle failed: %v", err)
	}
	if destRes.Status != "destroyed" {
		t.Errorf("unexpected DestroyBundle result: %+v", destRes)
	}
}

func TestHeuristicBundleGeneratorFormats(t *testing.T) {
	// TOON with webhook + RAG keywords
	toonManifest := buildHeuristicBundleManifest("Create a customer webhook RAG vector search lakehouse", "toon", "analytics-ws")
	if !strings.Contains(toonManifest, `schedule: "@webhook"`) || !strings.Contains(toonManifest, "webhook_enabled: true") {
		t.Errorf("expected @webhook schedule in TOON heuristic output, got:\n%s", toonManifest)
	}
	if !strings.Contains(toonManifest, "vector_indexes[1]") {
		t.Errorf("expected vector_indexes in TOON heuristic output, got:\n%s", toonManifest)
	}

	// YAML format
	yamlManifest := buildHeuristicBundleManifest("Create a biomedical PubMed RAG pipeline", "yaml", "bio-ws")
	if !strings.Contains(yamlManifest, "type: python_script") || !strings.Contains(yamlManifest, "pandas>=2.0") {
		t.Errorf("expected python_script with requirements in YAML heuristic output, got:\n%s", yamlManifest)
	}
	if err := runLocalValidation("generated.yml", "yaml", []byte(yamlManifest)); err != nil {
		t.Errorf("expected generated YAML manifest to pass local validation: %v", err)
	}
}
