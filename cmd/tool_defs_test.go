package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func executeCLIForTest(t *testing.T, stdoutTTY bool, args ...string) (string, string, int) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	ui.SetTTYOverrides(&stdoutTTY, &stdoutTTY)

	// Reset global flag variables before each run
	jsonOutput = false
	outputFlag = ""
	nonInteractive = false
	assumeYes = false
	forceExec = false
	ctxOverride = ""
	endpoint = ""
	token = ""
	sqlStatement = ""
	outputFormat = "table"
	toolDefsFormat = SchemaFormatOpenAI
	includeGlobalFlags = false
	loginEmail = ""
	loginPassword = ""
	wsRecursive = false
	wsOverwrite = false
	wsForce = false
	wsFormat = "table"
	wsDest = ""
	wsKernel = "python3"
	wsTimeout = 2 * time.Minute
	wsSubpath = ""
	wsSavePath = ""
	jobsWait = false
	jobsForce = false
	jobsConf = ""
	jobsLimit = 25
	jobsTryNumber = 1
	jobsTimeout = 10 * time.Minute
	jobsPollInterval = 3 * time.Second
	aiStream = false
	aiNoStream = false
	aiMode = "auto"
	aiModel = ""
	aiKey = ""
	dashTitle = ""
	dashDescription = ""
	dashTemplate = "blank"
	dashSpecFile = ""
	dashPublished = true
	dashForce = false
	dashTimeRange = ""
	userRole = ""
	userDisplayName = ""
	userActive = true
	wsAdminName = ""
	wsAdminSlug = ""
	wsAdminDescription = ""
	wsMemberEmail = ""
	wsMemberRole = "viewer"
	wsMemberForce = false
	adminSessionTimeout = 60
	adminMaxHours = 24
	adminGeminiModel = ""
	adminGeminiKey = ""
	adminForce = false
	mcpClient = "claude"
	mcpConfigPath = ""
	docsCompletionsDir = ""
	docsClean = true

	defer func() {
		ui.OutWriter = os.Stdout
		ui.ErrWriter = os.Stderr
		ui.SetTTYOverrides(nil, nil)
		rootCmd.SetArgs(nil)
		endpoint = ""
		token = ""
		ctxOverride = ""
		workspaceFlag = ""
		workspaceIDFlag = ""
	}()

	rootCmd.SetArgs(args)
	code := ExecuteRoot()
	return outBuf.String(), errBuf.String(), code
}

func TestToolDefsGeminiFormat(t *testing.T) {
	stdout, stderr, code := executeCLIForTest(t, true, "tool-defs", "--format", "gemini")
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	var spec GeminiToolsSpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to decode gemini schema JSON: %v\nstdout:\n%s", err, stdout)
	}

	if len(spec.FunctionDeclarations) == 0 {
		t.Fatalf("expected non-empty function_declarations")
	}

	foundQuery := false
	foundContextUse := false
	for _, fn := range spec.FunctionDeclarations {
		if fn.Name == "angler_tool_defs" || fn.Name == "angler_help" {
			t.Errorf("meta command %s should be excluded from tool-defs", fn.Name)
		}
		if fn.Name == "angler_query" {
			foundQuery = true
			if _, ok := fn.Parameters.Properties["sql"]; !ok {
				t.Errorf("expected 'sql' positional parameter in angler_query")
			}
			if _, ok := fn.Parameters.Properties["execute"]; !ok {
				t.Errorf("expected 'execute' flag parameter in angler_query")
			}
			if _, ok := fn.Parameters.Properties["token"]; ok {
				t.Errorf("global flag 'token' should be omitted by default")
			}
		}
		if fn.Name == "angler_context_use" {
			foundContextUse = true
			if _, ok := fn.Parameters.Properties["name"]; !ok {
				t.Errorf("expected 'name' positional parameter in angler_context_use")
			}
			if len(fn.Parameters.Required) == 0 || fn.Parameters.Required[0] != "name" {
				t.Errorf("expected 'name' to be required in angler_context_use, got %v", fn.Parameters.Required)
			}
		}
	}

	if !foundQuery {
		t.Errorf("expected angler_query in gemini declarations")
	}
	if !foundContextUse {
		t.Errorf("expected angler_context_use in gemini declarations")
	}
}

func TestToolDefsOpenAIAndIncludeGlobalFlags(t *testing.T) {
	stdout, stderr, code := executeCLIForTest(t, true, "schema", "--format", "openai", "--include-global-flags")
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	var tools []OpenAITool
	if err := json.Unmarshal([]byte(stdout), &tools); err != nil {
		t.Fatalf("failed to decode openai tools JSON: %v", err)
	}

	if len(tools) == 0 {
		t.Fatalf("expected non-empty tools slice")
	}

	for _, tool := range tools {
		if tool.Type != "function" {
			t.Errorf("expected tool.Type == 'function', got %q", tool.Type)
		}
		if tool.Function.Name == "angler_status" {
			if _, ok := tool.Function.Parameters.Properties["endpoint"]; !ok {
				t.Errorf("expected global flag 'endpoint' when --include-global-flags is passed")
			}
		}
	}
}

func TestToolDefsOpenAPI31Format(t *testing.T) {
	stdout, stderr, code := executeCLIForTest(t, true, "tool-defs", "--format", "openapi")
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to decode openapi JSON: %v", err)
	}

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0, got %s", spec.OpenAPI)
	}

	pathItem, ok := spec.Paths["/commands/catalog/ls"]
	if !ok {
		t.Fatalf("expected '/commands/catalog/ls' path in OpenAPI spec, got keys: %v", spec.Paths)
	}
	if pathItem.Post.OperationID != "angler_catalog_ls" {
		t.Errorf("expected operationId 'angler_catalog_ls', got %s", pathItem.Post.OperationID)
	}
}

func TestGlobalOutputFormatsAndNonTTYAutoJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/catalog":
			_ = json.NewEncoder(w).Encode([]client.CatalogItem{
				{Name: "lake.telemetry", Type: "table", Format: "parquet", Rows: 1024, Size: "4.2MB"},
			})
		case "/api/query":
			_ = json.NewEncoder(w).Encode(client.QueryResponse{
				Columns:  []string{"dept", "headcount"},
				Rows:     [][]interface{}{{"eng", 42}},
				RowCount: 1,
				Duration: "5ms",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// 1. Non-TTY auto-switches to JSON even without --json flag
	stdout, stderr, code := executeCLIForTest(t, false, "catalog", "ls", "--endpoint", ts.URL)
	if code != ui.ExitOK {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr)
	}
	var items []client.CatalogItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("expected auto-JSON output when stdout is non-TTY, got:\n%s\nerr: %v", stdout, err)
	}
	if len(items) != 1 || items[0].Name != "lake.telemetry" {
		t.Errorf("unexpected catalog items: %+v", items)
	}

	// 2. TOON output via --output toon
	stdout, stderr, code = executeCLIForTest(t, true, "query", "SELECT dept, headcount FROM emp", "--endpoint", ts.URL, "--output", "toon")
	if code != ui.ExitOK {
		t.Fatalf("expected exit 0 for TOON output, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "rows[1]{dept,headcount}:") || !strings.Contains(stdout, "  eng,42") {
		t.Errorf("expected TOON tabular output on stdout, got:\n%s", stdout)
	}
	// Info log "Casting query" must be routed to stderr, keeping stdout pure TOON
	if strings.Contains(stdout, "Casting query") {
		t.Errorf("informational message leaked into stdout during TOON output:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Casting query") {
		t.Errorf("expected informational message on stderr, got:\n%s", stderr)
	}

	// 3. YAML output via -o yaml
	stdout, stderr, code = executeCLIForTest(t, true, "version", "-o", "yaml")
	if code != ui.ExitOK {
		t.Fatalf("expected exit 0 for YAML output, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "version:") || !strings.Contains(stdout, "commit:") {
		t.Errorf("expected YAML version fields, got:\n%s", stdout)
	}
}

func TestNonInteractiveLoginAndExitCodes(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// 1. Non-interactive auth login without credentials must fail immediately with ExitUsageError (2)
	_, stderr, code := executeCLIForTest(t, true, "auth", "login", "--non-interactive", "--json")
	if code != ui.ExitUsageError {
		t.Fatalf("expected ExitUsageError (%d), got %d (stderr: %s)", ui.ExitUsageError, code, stderr)
	}
	var errResp ui.StructuredError
	if err := json.Unmarshal([]byte(stderr), &errResp); err != nil {
		t.Fatalf("expected structured JSON error on stderr, got %q: %v", stderr, err)
	}
	if errResp.Code != ui.ExitUsageError {
		t.Errorf("expected structured error code %d, got %d", ui.ExitUsageError, errResp.Code)
	}

	// 2. Context use for non-existent context returns ExitNotFoundError (4)
	_, stderr, code = executeCLIForTest(t, true, "context", "use", "nonexistent-ctx", "--json")
	if code != ui.ExitNotFoundError {
		t.Fatalf("expected ExitNotFoundError (%d), got %d (stderr: %s)", ui.ExitNotFoundError, code, stderr)
	}

	// 3. Auth login with invalid credentials against server returns ExitAuthError (3)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"invalid credentials"}`))
	}))
	defer ts.Close()

	_, stderr, code = executeCLIForTest(t, true, "auth", "login", "--non-interactive", "--email", "agent@lake.io", "--password", "wrong", "--endpoint", ts.URL, "--json")
	if code != ui.ExitAuthError {
		t.Fatalf("expected ExitAuthError (%d), got %d (stderr: %s)", ui.ExitAuthError, code, stderr)
	}
}
