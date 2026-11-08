package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/mark3labs/mcp-go/mcp"
)

func resetMCPFlags() {
	mcpClient = "claude"
	mcpConfigPath = ""
	jsonOutput = false
	outputFlag = ""
}

func TestMCPCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundMCP bool
	for _, c := range commands {
		if c.Name() == "mcp" {
			foundMCP = true
			subs := map[string]bool{"serve": false, "install": false}
			for _, sub := range c.Commands() {
				if _, ok := subs[sub.Name()]; ok {
					subs[sub.Name()] = true
				}
			}
			for s, found := range subs {
				if !found {
					t.Errorf("expected subcommand %q under mcp", s)
				}
			}
		}
	}
	if !foundMCP {
		t.Fatal("mcp command not registered on rootCmd")
	}
}

func setupMCPMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	now := time.Now().UTC().Truncate(time.Second)

	// SQL statements endpoint: /api/sql/statements
	mux.HandleFunc("/api/sql/statements", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.StatementExecutionResponse{
			StatementID: "stmt-123",
			Status: client.StatementStatus{
				State: "SUCCEEDED",
			},
			Manifest: &client.StatementManifest{
				Format: "JSON_OBJECT",
				Schema: client.StatementSchemaInfo{
					ColumnCount: 2,
					Columns: []client.StatementColumnInfo{
						{Name: "id", TypeText: "INTEGER", Position: 0},
						{Name: "count", TypeText: "BIGINT", Position: 1},
					},
				},
				TotalRowCount: 1,
			},
			Result: &client.StatementResultChunk{
				RowCount: 1,
				DataObjects: []map[string]interface{}{
					{"id": 1, "count": 42},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Catalogs endpoint: /api/catalogs
	mux.HandleFunc("/api/catalogs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cats := []client.Catalog{
			{ID: "cat-1", Name: "main"},
		}
		_ = json.NewEncoder(w).Encode(cats)
	})

	// Storage browse endpoint: /api/storage/browse
	mux.HandleFunc("/api/storage/browse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.StorageBrowseResponse{
			Workspace: "default",
			Prefix:    "gold/telemetry",
			Files: []client.StorageFile{
				{
					Key:          "gold/telemetry/data.parquet",
					Size:         1048576,
					LastModified: &now,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Notebook get endpoint: /api/notebooks/
	mux.HandleFunc("/api/notebooks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.NotebookDetail{
			Path:         "analysis/report.ipynb",
			Name:         "report.ipynb",
			Created:      &now,
			LastModified: &now,
			Content: &client.NotebookContent{
				Cells: []client.NotebookCell{
					{CellType: "code", Source: "import duckdb\nprint('Hello Lake')"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Pipelines trigger endpoint: /api/pipelines/telemetry_etl/trigger
	mux.HandleFunc("/api/pipelines/telemetry_etl/trigger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineTriggerResponse{
			DagID:         "telemetry_etl",
			DagRunID:      "manual__20261108T180000Z",
			ExecutionDate: now.Format(time.RFC3339),
			State:         "queued",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// AI Genie chat endpoint: /api/ai/chat
	mux.HandleFunc("/api/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.GenieChatResponse{
			OK:     true,
			Answer: "Lakehouse revenue increased by 14% this quarter.",
			SQL:    "SELECT sum(revenue) FROM gold.sales",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestMCPServerToolsExecution(t *testing.T) {
	ts := setupMCPMockServer(t)
	defer ts.Close()

	c := client.NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "test-tok",
	})

	s := buildMCPServer(c)

	ctx := context.Background()

	// Verify all 6 tools are registered
	tools := s.ListTools()
	expectedTools := []string{
		"angler_sql_query",
		"angler_catalog_inspect",
		"angler_fs_browse",
		"angler_workspace_read_notebook",
		"angler_jobs_trigger",
		"angler_ai_genie",
	}
	for _, expected := range expectedTools {
		if _, ok := tools[expected]; !ok {
			t.Errorf("expected tool %q to be registered on MCP server", expected)
		}
	}

	// 1. Tool: angler_sql_query
	reqSQL := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_sql_query",
			Arguments: map[string]any{
				"sql": "SELECT 1, 42",
			},
		},
	}
	resSQL, err := s.GetTool("angler_sql_query").Handler(ctx, reqSQL)
	if err != nil {
		t.Fatalf("angler_sql_query error: %v", err)
	}
	textSQL := resSQL.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textSQL, "42") {
		t.Errorf("expected 42 in SQL tool response: %s", textSQL)
	}

	// 2. Tool: angler_catalog_inspect
	reqCat := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_catalog_inspect",
		},
	}
	resCat, err := s.GetTool("angler_catalog_inspect").Handler(ctx, reqCat)
	if err != nil {
		t.Fatalf("angler_catalog_inspect error: %v", err)
	}
	textCat := resCat.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textCat, "main") {
		t.Errorf("expected main in catalog tool response: %s", textCat)
	}

	// 3. Tool: angler_fs_browse
	reqFS := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_fs_browse",
			Arguments: map[string]any{
				"path": "gold/telemetry",
			},
		},
	}
	resFS, err := s.GetTool("angler_fs_browse").Handler(ctx, reqFS)
	if err != nil {
		t.Fatalf("angler_fs_browse error: %v", err)
	}
	textFS := resFS.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textFS, "data.parquet") {
		t.Errorf("expected data.parquet in FS tool response: %s", textFS)
	}

	// 4. Tool: angler_workspace_read_notebook
	reqNB := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_workspace_read_notebook",
			Arguments: map[string]any{
				"path": "analysis/report.ipynb",
			},
		},
	}
	resNB, err := s.GetTool("angler_workspace_read_notebook").Handler(ctx, reqNB)
	if err != nil {
		t.Fatalf("angler_workspace_read_notebook error: %v", err)
	}
	textNB := resNB.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textNB, "Hello Lake") {
		t.Errorf("expected notebook cell content: %s", textNB)
	}

	// 5. Tool: angler_jobs_trigger
	reqJob := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_jobs_trigger",
			Arguments: map[string]any{
				"job_id": "telemetry_etl",
			},
		},
	}
	resJob, err := s.GetTool("angler_jobs_trigger").Handler(ctx, reqJob)
	if err != nil {
		t.Fatalf("angler_jobs_trigger error: %v", err)
	}
	textJob := resJob.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textJob, "manual__20261108T180000Z") {
		t.Errorf("expected run ID in job trigger response: %s", textJob)
	}

	// 6. Tool: angler_ai_genie
	reqGenie := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "angler_ai_genie",
			Arguments: map[string]any{
				"prompt": "How did revenue change?",
			},
		},
	}
	resGenie, err := s.GetTool("angler_ai_genie").Handler(ctx, reqGenie)
	if err != nil {
		t.Fatalf("angler_ai_genie error: %v", err)
	}
	textGenie := resGenie.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textGenie, "14%") {
		t.Errorf("expected 14%% in genie response: %s", textGenie)
	}
}

func TestMCPInstallCLI(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	resetMCPFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "mcp", "install", "--client", "claude", "--config-path", cfgPath)
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "successfully configured") {
		t.Errorf("expected success message in stdout: %s", stdout)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read written config file: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse written JSON: %v", err)
	}

	mcpServers, ok := parsed["mcpServers"].(map[string]any)
	if !ok || mcpServers["angler"] == nil {
		t.Fatalf("angler server not found in mcpServers: %+v", parsed)
	}
}
