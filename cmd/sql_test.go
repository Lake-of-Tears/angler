package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetSQLTestFlags() {
	sqlStatement = ""
	sqlFile = ""
	sqlOutputFormat = "table"
	outputFormat = "table"
	sqlWarehouse = "default"
	sqlCatalog = "main"
	sqlSchema = "default"
	sqlRowLimit = 1000
	sqlParams = ""
	sqlExplainFlag = false
	sqlModel = ""
	sqlCurrentSQL = ""
	sqlAutoExecute = false
	sqlErrorMsg = ""
	sqlHistoryLimit = 20
	sqlHistoryStatus = ""
	jsonOutput = false
	outputFlag = ""
}

func executeSQLCommand(t *testing.T, endpointURL string, args ...string) (string, string, int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut := ui.OutWriter
	origErr := ui.ErrWriter
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	stdoutTTY := false
	ui.SetTTYOverrides(&stdoutTTY, &stdoutTTY)

	resetSQLTestFlags()

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

func TestSQLCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundSQL, foundQuery bool
	for _, c := range commands {
		if c.Name() == "sql" {
			foundSQL = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			expected := []string{"exec", "explain", "generate", "fix", "history", "get", "cancel"}
			for _, exp := range expected {
				if !subMap[exp] {
					t.Errorf("expected subcommand %q under sqlCmd", exp)
				}
			}
		}
		if c.Name() == "query" {
			foundQuery = true
		}
	}

	if !foundSQL {
		t.Errorf("expected 'sql' command to be registered on rootCmd")
	}
	if !foundQuery {
		t.Errorf("expected 'query' command to be registered on rootCmd")
	}
}

func TestSQLExecStatement(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/sql/statements" && r.Method == http.MethodPost {
			var req client.StatementExecutionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to parse statement request: %v", err)
			}

			if req.Statement != "SELECT 42 AS answer" {
				t.Errorf("unexpected statement: %s", req.Statement)
			}
			if req.WarehouseID != "default" {
				t.Errorf("expected warehouse default, got %s", req.WarehouseID)
			}

			execMs := 15.2
			resp := client.StatementExecutionResponse{
				StatementID: "stmt-abc-123",
				Status: client.StatementStatus{
					State: "SUCCEEDED",
				},
				Manifest: &client.StatementManifest{
					Format: "JSON_ARRAY",
					Schema: client.StatementSchemaInfo{
						ColumnCount: 1,
						Columns: []client.StatementColumnInfo{
							{Name: "answer", TypeText: "INTEGER", Position: 0},
						},
					},
					TotalRowCount: 1,
				},
				Result: &client.StatementResultChunk{
					ChunkIndex: 0,
					RowCount:   1,
					DataArray: [][]interface{}{
						{42},
					},
				},
				ExecutionMs: &execMs,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	// 1. Test standard sql exec
	stdout, stderr, code := executeSQLCommand(t, ts.URL, "sql", "exec", "SELECT 42 AS answer")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "stmt-abc-123") {
		t.Errorf("expected output to contain statement ID stmt-abc-123, got:\n%s", stdout)
	}

	// 2. Test direct sql command invocation (backwards compat)
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "SELECT 42 AS answer")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "stmt-abc-123") {
		t.Errorf("expected direct sql execution output, got:\n%s", stdout)
	}

	// 3. Test query alias (backwards compat)
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "query", "SELECT 42 AS answer")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "stmt-abc-123") {
		t.Errorf("expected query alias output, got:\n%s", stdout)
	}
}

func TestSQLExecFormats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/sql/statements" && r.Method == http.MethodPost {
			execMs := 8.4
			resp := client.StatementExecutionResponse{
				StatementID: "stmt-fmt-456",
				Status: client.StatementStatus{
					State: "SUCCEEDED",
				},
				Manifest: &client.StatementManifest{
					Format: "JSON_ARRAY",
					Schema: client.StatementSchemaInfo{
						ColumnCount: 2,
						Columns: []client.StatementColumnInfo{
							{Name: "id", TypeText: "INTEGER", Position: 0},
							{Name: "city", TypeText: "VARCHAR", Position: 1},
						},
					},
					TotalRowCount: 2,
				},
				Result: &client.StatementResultChunk{
					ChunkIndex: 0,
					RowCount:   2,
					DataArray: [][]interface{}{
						{1, "Ardmore"},
						{2, "Henderson"},
					},
				},
				ExecutionMs: &execMs,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	// Test CSV format
	stdout, stderr, code := executeSQLCommand(t, ts.URL, "sql", "exec", "SELECT id, city FROM locations", "--format", "csv")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "id,city") || !strings.Contains(stdout, "1,Ardmore") {
		t.Errorf("expected CSV header and rows, got:\n%s", stdout)
	}

	// Test NDJSON format
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "exec", "SELECT id, city FROM locations", "--format", "ndjson")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d:\n%s", len(lines), stdout)
	}
	var rec map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("failed to decode NDJSON line: %v", err)
	}
	if rec["city"] != "Ardmore" {
		t.Errorf("expected city Ardmore, got %v", rec["city"])
	}
}

func TestSQLExecFileAndParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/sql/statements" && r.Method == http.MethodPost {
			var req client.StatementExecutionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			if !strings.Contains(req.Statement, "WHERE age > ?") {
				t.Errorf("expected query from file, got %s", req.Statement)
			}
			if len(req.Parameters) == 0 {
				t.Errorf("expected parameters in request")
			}

			resp := client.StatementExecutionResponse{
				StatementID: "stmt-params-789",
				Status: client.StatementStatus{
					State: "SUCCEEDED",
				},
				Result: &client.StatementResultChunk{
					RowCount: 0,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	tmpDir := t.TempDir()
	sqlFilePath := filepath.Join(tmpDir, "filter.sql")
	if err := os.WriteFile(sqlFilePath, []byte("SELECT * FROM members WHERE age > ?"), 0644); err != nil {
		t.Fatalf("failed to write temp sql file: %v", err)
	}

	_, stderr, code := executeSQLCommand(t, ts.URL, "sql", "exec", "-f", sqlFilePath, "--params", "[21]")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
}

func TestSQLExplainAndGenerateAndFix(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ai/sql/explain":
			json.NewEncoder(w).Encode(client.SqlExplainResponse{
				OK:               true,
				Summary:          "Scans orders partitioned by date",
				Steps:            []string{"Scan orders table", "Aggregate by department"},
				TablesReferenced: []string{"orders"},
			})
		case "/api/ai/sql/generate":
			json.NewEncoder(w).Encode(client.SqlGenerateResponse{
				OK:          true,
				SQL:         "SELECT sum(amount) FROM sales",
				Explanation: "Calculates total sales revenue",
				TablesUsed:  []string{"sales"},
			})
		case "/api/ai/sql/fix":
			json.NewEncoder(w).Encode(client.SqlFixResponse{
				OK:          true,
				FixedSQL:    "SELECT product_id FROM products",
				Explanation: "Fixed typo in column name",
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

	// 1. Test explain
	stdout, stderr, code := executeSQLCommand(t, ts.URL, "sql", "explain", "SELECT sum(amount) FROM orders")
	if code != ui.ExitOK {
		t.Fatalf("explain failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Scans orders partitioned by date") {
		t.Errorf("expected summary in output, got:\n%s", stdout)
	}

	// 2. Test generate
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "generate", "Calculate total sales")
	if code != ui.ExitOK {
		t.Fatalf("generate failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "SELECT sum(amount) FROM sales") {
		t.Errorf("expected generated SQL in output, got:\n%s", stdout)
	}

	// 3. Test fix
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "fix", "SELECT pid FROM products", "-e", "column pid not found")
	if code != ui.ExitOK {
		t.Fatalf("fix failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "SELECT product_id FROM products") {
		t.Errorf("expected repaired SQL in output, got:\n%s", stdout)
	}
}

func TestSQLHistoryAndGetAndCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/sql/statements" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]client.StatementExecutionResponse{
				{
					StatementID: "stmt-h-1",
					Status:      client.StatementStatus{State: "SUCCEEDED"},
					Catalog:     "main",
					Schema:      "default",
				},
			})
		case r.URL.Path == "/api/sql/statements/stmt-h-1" && r.Method == http.MethodGet:
			execMs := 4.2
			json.NewEncoder(w).Encode(client.StatementExecutionResponse{
				StatementID: "stmt-h-1",
				Status:      client.StatementStatus{State: "SUCCEEDED"},
				Catalog:     "main",
				Schema:      "default",
				ExecutionMs: &execMs,
			})
		case r.URL.Path == "/api/sql/statements/stmt-h-1/cancel" && r.Method == http.MethodPost:
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

	// Test history
	stdout, stderr, code := executeSQLCommand(t, ts.URL, "sql", "history", "--limit", "5")
	if code != ui.ExitOK {
		t.Fatalf("history failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "stmt-h-1") {
		t.Errorf("expected stmt-h-1 in history, got:\n%s", stdout)
	}

	// Test get
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "get", "stmt-h-1")
	if code != ui.ExitOK {
		t.Fatalf("get failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "stmt-h-1") {
		t.Errorf("expected statement details, got:\n%s", stdout)
	}

	// Test cancel
	stdout, stderr, code = executeSQLCommand(t, ts.URL, "sql", "cancel", "stmt-h-1")
	if code != ui.ExitOK {
		t.Fatalf("cancel failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "canceled") && !strings.Contains(stdout, "canceled") {
		t.Errorf("expected canceled confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}
}
