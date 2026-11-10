package cmd

import (
	"bytes"
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
	mcpTransport = "stdio"
	mcpRemote = false
	mcpPrint = false
	jsonOutput = false
	outputFlag = ""
}

func TestMCPCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundMCP bool
	for _, c := range commands {
		if c.Name() == "mcp" {
			foundMCP = true
			subs := map[string]bool{
				"serve":         false,
				"status":        false,
				"tools":         false,
				"resources":     false,
				"read-resource": false,
				"install":       false,
			}
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

	// Vector retrieve endpoint: /api/v1/retrieve
	mux.HandleFunc("/api/v1/retrieve", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"query": "revenue",
			"total": 1,
			"results": []map[string]any{
				{"chunk_id": "chk-1", "content": "Q4 revenue summary", "score": 0.95},
			},
		})
	})

	// Remote /api/mcp endpoint
	mux.HandleFunc("/api/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":             "lake-of-tears-mcp",
				"version":          "1.0.0",
				"protocolVersions": []string{"2025-03-26", "2024-11-05"},
				"endpoints": map[string]string{
					"streamable_http": "/api/mcp",
					"sse":             "/api/mcp/sse",
					"messages":        "/api/mcp/messages",
				},
				"tools": []string{
					"list_workspaces_and_catalogs",
					"describe_table",
					"semantic_search_datalake",
					"query_datalake_sql",
					"ask_genie",
				},
			})
			return
		}

		var rpcReq client.MCPJSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&rpcReq)
		switch rpcReq.Method {
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "list_workspaces_and_catalogs", "description": "List authorized workspaces and catalogs"},
						{"name": "describe_table", "description": "Describe a catalog table"},
					},
				},
			})
		case "resources/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"resources": []map[string]any{
						{
							"uri":         "lot://catalogs/default/main",
							"name":        "default/main",
							"mimeType":    "application/json",
							"description": "Catalog main in workspace default",
						},
					},
				},
			})
		case "resources/read":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"contents": []map[string]any{
						{
							"uri":      "lot://catalogs/default/main",
							"mimeType": "application/json",
							"text":     `{"name":"main","slug":"main","schemas":[]}`,
						},
					},
				},
			})
		case "ping":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result":  map[string]any{},
			})
		default:
			w.WriteHeader(http.StatusAccepted)
		}
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

	// Verify both legacy angler_* tools and canonical /api/mcp tools are registered
	tools := s.ListTools()
	expectedTools := []string{
		"angler_sql_query",
		"angler_catalog_inspect",
		"angler_fs_browse",
		"angler_workspace_read_notebook",
		"angler_jobs_trigger",
		"angler_ai_genie",
		"list_workspaces_and_catalogs",
		"describe_table",
		"semantic_search_datalake",
		"query_datalake_sql",
		"ask_genie",
		"angler_lakeset_datasets",
		"angler_lakeset_explore",
		"angler_secrets_list",
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

	// 2. Canonical Tool: semantic_search_datalake
	reqSem := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "semantic_search_datalake",
			Arguments: map[string]any{
				"query": "revenue",
				"top_k": 3,
			},
		},
	}
	resSem, err := s.GetTool("semantic_search_datalake").Handler(ctx, reqSem)
	if err != nil {
		t.Fatalf("semantic_search_datalake error: %v", err)
	}
	textSem := resSem.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textSem, "Q4 revenue summary") {
		t.Errorf("expected Q4 revenue summary in semantic search response: %s", textSem)
	}

	// 3. Canonical Tool: ask_genie
	reqAsk := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask_genie",
			Arguments: map[string]any{
				"question": "How did revenue change?",
			},
		},
	}
	resAsk, err := s.GetTool("ask_genie").Handler(ctx, reqAsk)
	if err != nil {
		t.Fatalf("ask_genie error: %v", err)
	}
	textAsk := resAsk.Content[0].(mcp.TextContent).Text
	if !strings.Contains(textAsk, "14%") {
		t.Errorf("expected 14%% in ask_genie response: %s", textAsk)
	}
}

func TestMCPRemoteSubcommandsAndBridge(t *testing.T) {
	ts := setupMCPMockServer(t)
	defer ts.Close()

	// 1. status
	resetMCPFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "--endpoint", ts.URL, "mcp", "status")
	if code != ui.ExitOK {
		t.Fatalf("mcp status failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "lake-of-tears-mcp") {
		t.Fatalf("expected lake-of-tears-mcp in status output: %s", stdout)
	}

	// 2. tools
	resetMCPFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", ts.URL, "mcp", "tools")
	if code != ui.ExitOK {
		t.Fatalf("mcp tools failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "list_workspaces_and_catalogs") {
		t.Fatalf("expected list_workspaces_and_catalogs in tools output: %s", stdout)
	}

	// 3. resources
	resetMCPFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", ts.URL, "mcp", "resources")
	if code != ui.ExitOK {
		t.Fatalf("mcp resources failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "lot://catalogs/default/main") {
		t.Fatalf("expected lot://catalogs/default/main in resources output: %s", stdout)
	}

	// 4. read-resource
	resetMCPFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", ts.URL, "mcp", "read-resource", "lot://catalogs/default/main")
	if code != ui.ExitOK {
		t.Fatalf("mcp read-resource failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, `"slug":"main"`) {
		t.Fatalf("expected catalog JSON in read-resource output: %s", stdout)
	}

	// 5. Remote stdio-to-HTTP bridge
	c := client.NewClient(config.Context{Endpoint: ts.URL, Token: "tok"})
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":99,"method":"ping"}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	var out bytes.Buffer
	if err := serveRemoteMCPBridge(c, in, &out); err != nil {
		t.Fatalf("serveRemoteMCPBridge failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], `"id":99`) {
		t.Fatalf("expected 1 response line for ping (notification ignored), got: %q", out.String())
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

	// Test --transport http --print
	resetMCPFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", "https://lake.example.com", "--token", "tok-abc", "mcp", "install", "--client", "antigravity", "--transport", "http", "--print")
	if code != ui.ExitOK {
		t.Fatalf("mcp install --transport http --print failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "https://lake.example.com/api/mcp") || !strings.Contains(stdout, "streamable-http") {
		t.Fatalf("expected streamable-http config in --print output, got: %s", stdout)
	}
}
