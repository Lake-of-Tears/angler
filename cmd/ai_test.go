package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetAIFlags() {
	aiStream = false
	aiNoStream = false
	aiMode = "auto"
	aiModel = ""
	aiKey = ""
	jsonOutput = false
	outputFlag = ""
}

func TestAICommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundAI bool
	for _, c := range commands {
		if c.Name() == "ai" {
			foundAI = true
			if len(c.Aliases) == 0 || c.Aliases[0] != "genie" {
				t.Errorf("expected alias 'genie' on ai command, got %v", c.Aliases)
			}
			expectedSubcommands := map[string]bool{
				"chat":         false,
				"context":      false,
				"vector-index": false,
				"test-key":     false,
			}
			for _, sub := range c.Commands() {
				if _, ok := expectedSubcommands[sub.Name()]; ok {
					expectedSubcommands[sub.Name()] = true
				}
				if sub.Name() == "vector-index" {
					subSubs := map[string]bool{
						"list":   false,
						"search": false,
						"sync":   false,
					}
					for _, ss := range sub.Commands() {
						if _, ok := subSubs[ss.Name()]; ok {
							subSubs[ss.Name()] = true
						}
					}
					for sName, found := range subSubs {
						if !found {
							t.Errorf("expected subcommand %q under ai vector-index", sName)
						}
					}
				}
			}
			for sub, found := range expectedSubcommands {
				if !found {
					t.Errorf("expected subcommand %q registered under ai", sub)
				}
			}
		}
	}
	if !foundAI {
		t.Fatal("ai command not registered on rootCmd")
	}
}

func setupAIMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Chat endpoint: /api/ai/chat
	mux.HandleFunc("/api/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req client.GenieChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		resp := client.GenieChatResponse{
			Answer:  fmt.Sprintf("Found 142 total orders matching %q", req.Question),
			SQL:     "SELECT COUNT(*) FROM main.retail.orders;",
			Columns: []string{"count"},
			Data: []map[string]any{
				{"count": 142},
			},
			GroundedDocs: []string{"orders table schema", "retail catalog documentation"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Chat stream endpoint: /api/ai/chat/stream
	mux.HandleFunc("/api/ai/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		step1 := `{"message":"Analyzing database catalogs"}`
		step2 := `{"message":"Synthesizing SQL query"}`
		res := `{"answer":"Found 142 orders","sql":"SELECT count(*) FROM orders","data":[{"count":142}],"columns":["count"]}`

		fmt.Fprintf(w, "event: step\ndata: %s\n\n", step1)
		fmt.Fprintf(w, "event: step\ndata: %s\n\n", step2)
		fmt.Fprintf(w, "event: result\ndata: %s\n\n", res)
	})

	// Context endpoint: /api/ai/context
	mux.HandleFunc("/api/ai/context", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.AIContextResponse{
			WorkspaceName: "Engineering Analytics",
			WorkspaceSlug: "eng-analytics",
			Role:          "admin",
			Catalogs: []map[string]any{
				{
					"name": "lake_dev",
					"schemas": []any{
						map[string]any{"name": "gold"},
						map[string]any{"name": "silver"},
					},
				},
			},
			DefaultDatasets: []map[string]any{
				{"name": "Daily Active Users", "table": "lake_dev.gold.daily_active_users"},
			},
			SQLAIFunctions: []map[string]any{
				{"name": "ai_classify", "description": "Classifies sentiment or intent of text"},
				{"name": "ai_extract", "description": "Extracts structured JSON entities from unstructured text"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Catalogs list: /api/catalogs
	mux.HandleFunc("/api/catalogs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cats := []client.Catalog{
			{ID: "cat-1", Name: "lake_dev"},
		}
		_ = json.NewEncoder(w).Encode(cats)
	})

	// Schemas list: /api/catalogs/cat-1/schemas
	mux.HandleFunc("/api/catalogs/cat-1/schemas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		schemas := []client.CatalogSchema{
			{ID: "sch-1", Name: "gold", CatalogID: "cat-1"},
		}
		_ = json.NewEncoder(w).Encode(schemas)
	})

	// Tables list: /api/catalogs/cat-1/schemas/sch-1/tables
	mux.HandleFunc("/api/catalogs/cat-1/schemas/sch-1/tables", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tables := []client.CatalogTable{
			{
				ID:       "tbl-1",
				Name:     "customer_reviews",
				SchemaID: "sch-1",
				VectorIndex: map[string]interface{}{
					"source_column":  "review_text",
					"total_vectors":  float64(8520),
					"status":         "READY",
					"s3_vector_path": "s3://lakehouse/indexes/cat-1/sch-1/tbl-1.index",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(tables)
	})

	// Vector index sync: /api/catalogs/cat-1/schemas/sch-1/tables/tbl-1/vector-index/sync
	mux.HandleFunc("/api/catalogs/cat-1/schemas/sch-1/tables/tbl-1/vector-index/sync", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "sync_initiated"})
	})

	// Test Gemini key: /api/admin/settings/test-gemini-key
	mux.HandleFunc("/api/admin/settings/test-gemini-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["api_key"] == "invalid-key" {
			_ = json.NewEncoder(w).Encode(client.TestGeminiKeyResponse{
				OK:      false,
				Model:   body["model"],
				Message: "API key invalid or unauthorized by provider",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(client.TestGeminiKeyResponse{
			OK:      true,
			Model:   body["model"],
			Message: "Gemini connection verified successfully",
		})
	})

	return httptest.NewServer(mux)
}

func TestAINewGatewayAndAdminCommands(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/retrieve", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"query": "churn risk",
			"total": 1,
			"results": []map[string]any{
				{
					"chunk_id": "chk-101",
					"content":  "Enterprise customer churn risk analysis",
					"score":    0.94,
					"metadata": map[string]any{
						"workspace_slug": "default",
						"catalog_slug":   "main",
						"schema_slug":    "gold",
						"table_slug":     "churn_notes",
					},
				},
			},
		})
	})

	mux.HandleFunc("/api/v1/vector_stores", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"id": "vs_default_main", "object": "vector_store", "name": "default/main", "status": "completed", "workspace_slug": "default", "catalog_slug": "main"},
			},
		})
	})

	mux.HandleFunc("/api/v1/vector_stores/vs_default_main/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":       "vector_store.search_results.page",
			"search_query": "churn",
			"data": []map[string]any{
				{
					"file_id":  "file_1",
					"filename": "main.gold.churn_notes",
					"score":    0.91,
					"content": []map[string]string{
						{"type": "text", "text": "Churn note snippet"},
					},
				},
			},
		})
	})

	mux.HandleFunc("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"id": "lot-genie", "object": "model", "owned_by": "lake-of-tears"},
				{"id": "lot-rag:default", "object": "model", "owned_by": "lake-of-tears"},
			},
		})
	})

	mux.HandleFunc("/api/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "text-embedding-004",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": []float64{0.1, 0.2, 0.3}},
			},
		})
	})

	mux.HandleFunc("/api/ai/table-from-prompt", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"table_name":  "churn_predictions",
			"description": "Predicted customer churn scores",
			"sql":         "CREATE TABLE gold.churn_predictions AS SELECT 1 AS id",
			"row_count":   25,
		})
	})

	mux.HandleFunc("/api/ai/generate-description", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"description": "Enriched semantic table description for executive reporting",
			"columns": map[string]string{
				"id": "Primary identifier",
			},
		})
	})

	mux.HandleFunc("/api/admin/ai-provider", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active_provider": "vertex",
			"providers": []map[string]any{
				{"id": "vertex", "label": "Google Cloud Vertex AI", "chat_model": "gemini-2.5-flash", "embed_model": "text-embedding-005"},
				{"id": "gemini", "label": "Google Gemini API", "chat_model": "gemini-2.5-flash", "embed_model": "text-embedding-004"},
			},
		})
	})

	mux.HandleFunc("/api/admin/vaults", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"provider": "internal", "display_name": "PostgreSQL Encrypted Vault", "is_configured": true},
			{"provider": "hashicorp_vault", "display_name": "HashiCorp Vault", "is_configured": true},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 1. ai retrieve
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "retrieve", "churn risk", "--top-k", "3")
	if code != ui.ExitOK {
		t.Fatalf("ai retrieve failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "churn_notes") {
		t.Fatalf("expected churn_notes in retrieve output: %s", stdout)
	}

	// 2. ai vector-stores list & search
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "vector-stores", "list")
	if code != ui.ExitOK || !strings.Contains(stdout, "vs_default_main") {
		t.Fatalf("ai vector-stores list failed (%d): %s / %s", code, stderr, stdout)
	}
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "vector-stores", "search", "vs_default_main", "churn")
	if code != ui.ExitOK || !strings.Contains(stdout, "Churn note snippet") {
		t.Fatalf("ai vector-stores search failed (%d): %s / %s", code, stderr, stdout)
	}

	// 3. ai models
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "models")
	if code != ui.ExitOK || !strings.Contains(stdout, "lot-genie") {
		t.Fatalf("ai models failed (%d): %s / %s", code, stderr, stdout)
	}

	// 4. ai embeddings
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "embeddings", "lakehouse vector test")
	if code != ui.ExitOK || !strings.Contains(stdout, "text-embedding-004") {
		t.Fatalf("ai embeddings failed (%d): %s / %s", code, stderr, stdout)
	}

	// 5. ai table-from-prompt
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "table-from-prompt", "Generate churn predictions", "--schema", "gold")
	if code != ui.ExitOK || !strings.Contains(stdout, "churn_predictions") {
		t.Fatalf("ai table-from-prompt failed (%d): %s / %s", code, stderr, stdout)
	}

	// 6. ai enrich
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "ai", "enrich", "--catalog", "main", "--schema", "gold", "--table", "churn_predictions")
	if code != ui.ExitOK || !strings.Contains(stdout, "Enriched semantic table description") {
		t.Fatalf("ai enrich failed (%d): %s / %s", code, stderr, stdout)
	}

	// 7. admin ai-providers list
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "admin", "ai-providers", "list")
	if code != ui.ExitOK || !strings.Contains(stdout, "Google Cloud Vertex AI") {
		t.Fatalf("admin ai-providers list failed (%d): %s / %s", code, stderr, stdout)
	}

	// 8. admin vaults list
	stdout, stderr, code = executeCLIForTest(t, true, "--endpoint", srv.URL, "admin", "vaults", "list")
	if code != ui.ExitOK || !strings.Contains(stdout, "hashicorp_vault") {
		t.Fatalf("admin vaults list failed (%d): %s / %s", code, stderr, stdout)
	}
}

func TestAIChatSingleTurn(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	// 1. Text output with table
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, false, "ai", "chat", "Count total orders in 2026", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Found 142 total orders") {
		t.Errorf("expected answer in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "SELECT COUNT(*)") {
		t.Errorf("expected SQL in stdout: %s", stdout)
	}

	// 2. JSON output
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "ai", "chat", "Count total orders in 2026", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var resp client.GenieChatResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("failed to decode JSON chat response: %v\nstdout: %s", err, stdout)
	}
	if !strings.Contains(resp.Answer, "142") {
		t.Errorf("expected answer in JSON: %+v", resp)
	}
	if resp.SQL != "SELECT COUNT(*) FROM main.retail.orders;" {
		t.Errorf("unexpected SQL: %s", resp.SQL)
	}
}

func TestAIChatStream(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "ai", "chat", "Analyze daily trends", "--stream", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Analyzing database catalogs") {
		t.Errorf("expected step event in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Found 142 orders") {
		t.Errorf("expected final answer in stdout: %s", stdout)
	}
}

func TestAIContext(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	// 1. Text display
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "ai", "context", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Engineering Analytics") {
		t.Errorf("expected workspace name in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "lake_dev") {
		t.Errorf("expected catalog name: %s", stdout)
	}
	if !strings.Contains(stdout, "ai_classify") {
		t.Errorf("expected SQL AI function: %s", stdout)
	}

	// 2. JSON output
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "ai", "context", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var ctx client.AIContextResponse
	if err := json.Unmarshal([]byte(stdout), &ctx); err != nil {
		t.Fatalf("failed to decode AI context JSON: %v\nstdout: %s", err, stdout)
	}
	if ctx.WorkspaceSlug != "eng-analytics" || len(ctx.SQLAIFunctions) != 2 {
		t.Errorf("unexpected context payload: %+v", ctx)
	}
}

func TestAIVectorIndexCommands(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	// 1. List in table format
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "ai", "vector-index", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "customer_reviews") {
		t.Errorf("expected table name in vector index list: %s", stdout)
	}
	if !strings.Contains(stdout, "8520") {
		t.Errorf("expected vector count: %s", stdout)
	}

	// 2. List in JSON format
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "ai", "vector", "ls", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var summaries []client.VectorIndexSummary
	if err := json.Unmarshal([]byte(stdout), &summaries); err != nil {
		t.Fatalf("failed to parse vector index summaries: %v\nstdout: %s", err, stdout)
	}
	if len(summaries) != 1 || summaries[0].TableName != "customer_reviews" {
		t.Errorf("unexpected vector summaries: %+v", summaries)
	}

	// 3. Search
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "ai", "vector-index", "search", "best customer service", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Vector search result for") {
		t.Errorf("expected search header: %s", stdout)
	}
	if !strings.Contains(stdout, "orders table schema") {
		t.Errorf("expected grounded doc: %s", stdout)
	}

	// 4. Sync
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "ai", "vector-index", "sync", "cat-1", "sch-1", "tbl-1", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Vector index synchronization initiated") {
		t.Errorf("expected sync message: %s", stdout)
	}
}

func TestAITestKey(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	// 1. Success case
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "ai", "test-key", "--key", "valid-key", "--model", "gemini-2.0-flash", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "LLM Provider verification succeeded") {
		t.Errorf("expected success in stdout: %s", stdout)
	}

	// 2. Failure case
	resetAIFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "ai", "test-key", "--key", "invalid-key", "--model", "gemini-2.0-flash", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "LLM Provider verification failed") {
		t.Errorf("expected failure message in stderr: %s (stdout: %s)", stderr, stdout)
	}
	if !strings.Contains(stdout, "API key invalid or unauthorized") {
		t.Errorf("expected message details in stdout: %s", stdout)
	}
}

func TestAIAliases(t *testing.T) {
	ts := setupAIMockServer(t)
	defer ts.Close()

	// Test 'genie' alias for 'ai'
	resetAIFlags()
	stdout, stderr, code := executeCLIForTest(t, false, "genie", "chat", "Count orders", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Found 142 total orders") {
		t.Errorf("expected answer using 'genie' alias: %s", stdout)
	}
}
