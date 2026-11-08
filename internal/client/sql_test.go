package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestExecuteStatement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sql/statements" {
			t.Fatalf("expected path /api/sql/statements, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req StatementExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Statement != "SELECT 1" {
			t.Errorf("expected statement 'SELECT 1', got %s", req.Statement)
		}

		execMs := 12.5
		resp := StatementExecutionResponse{
			StatementID: "stmt-123",
			Status: StatementStatus{
				State: "SUCCEEDED",
			},
			Manifest: &StatementManifest{
				Format: "JSON_ARRAY",
				Schema: StatementSchemaInfo{
					ColumnCount: 1,
					Columns: []StatementColumnInfo{
						{Name: "col1", TypeText: "INTEGER", Position: 0},
					},
				},
				TotalRowCount: 1,
			},
			Result: &StatementResultChunk{
				ChunkIndex: 0,
				RowCount:   1,
				DataArray: [][]interface{}{
					{1},
				},
			},
			ExecutionMs: &execMs,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(config.Context{
		Endpoint:  server.URL,
		Workspace: "analytics",
	})

	resp, err := c.ExecuteStatement(StatementExecutionRequest{
		Statement: "SELECT 1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatementID != "stmt-123" {
		t.Errorf("expected statement_id 'stmt-123', got %s", resp.StatementID)
	}
	if resp.Status.State != "SUCCEEDED" {
		t.Errorf("expected state SUCCEEDED, got %s", resp.Status.State)
	}
	if len(resp.Result.DataArray) != 1 {
		t.Errorf("expected 1 row, got %d", len(resp.Result.DataArray))
	}
}

func TestListStatements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sql/statements" {
			t.Fatalf("expected /api/sql/statements, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "20" {
			t.Errorf("expected limit=20, got %s", r.URL.Query().Get("limit"))
		}

		items := []StatementExecutionResponse{
			{
				StatementID: "stmt-001",
				Status:      StatementStatus{State: "SUCCEEDED"},
			},
			{
				StatementID: "stmt-002",
				Status:      StatementStatus{State: "FAILED"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer server.Close()

	c := NewClient(config.Context{Endpoint: server.URL})
	list, err := c.ListStatements(20, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
}

func TestAISQLEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ai/sql/explain":
			json.NewEncoder(w).Encode(SqlExplainResponse{
				OK:      true,
				Summary: "Performs full table scan",
				Steps:   []string{"1. Scan telemetry", "2. Filter errors"},
			})
		case "/api/ai/sql/generate":
			json.NewEncoder(w).Encode(SqlGenerateResponse{
				OK:          true,
				SQL:         "SELECT count(*) FROM users",
				Explanation: "Counts all rows in users table",
			})
		case "/api/ai/sql/fix":
			json.NewEncoder(w).Encode(SqlFixResponse{
				OK:          true,
				FixedSQL:    "SELECT id, name FROM users",
				Explanation: "Replaced non-existent column 'username' with 'name'",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(config.Context{Endpoint: server.URL})

	exp, err := c.ExplainSQL(SqlExplainRequest{SQL: "SELECT * FROM telemetry"})
	if err != nil || !exp.OK {
		t.Fatalf("ExplainSQL failed: %v", err)
	}
	if exp.Summary != "Performs full table scan" {
		t.Errorf("unexpected summary: %s", exp.Summary)
	}

	gen, err := c.GenerateSQL(SqlGenerateRequest{Prompt: "Count all users"})
	if err != nil || !gen.OK {
		t.Fatalf("GenerateSQL failed: %v", err)
	}
	if gen.SQL != "SELECT count(*) FROM users" {
		t.Errorf("unexpected generated sql: %s", gen.SQL)
	}

	fix, err := c.FixSQL(SqlFixRequest{SQL: "SELECT username FROM users", Error: "column username not found"})
	if err != nil || !fix.OK {
		t.Fatalf("FixSQL failed: %v", err)
	}
	if fix.FixedSQL != "SELECT id, name FROM users" {
		t.Errorf("unexpected fixed sql: %s", fix.FixedSQL)
	}
}
