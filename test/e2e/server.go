package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
)

// MockBackend provides a comprehensive in-memory mock of the Lake of Tears REST API
// designed specifically for end-to-end integration testing of the Angler CLI.
type MockBackend struct {
	Server    *httptest.Server
	URL       string
	mu        sync.RWMutex
	files     map[string][]byte
	tokens    []client.PersonalAccessToken
	lastQuery string
}

func NewMockBackend() *MockBackend {
	mb := &MockBackend{
		files: make(map[string][]byte),
		tokens: []client.PersonalAccessToken{
			{
				ID:          "tok-initial",
				Name:        "seed-pat",
				TokenPrefix: "lot_pat_seed...",
				CreatedAt:   time.Now(),
			},
		},
	}

	mux := http.NewServeMux()

	// 1. Auth endpoints
	mux.HandleFunc("/api/auth/login", mb.handleLogin)
	mux.HandleFunc("/api/auth/me", mb.handleCurrentUser)

	// 2. PAT Token endpoints
	mux.HandleFunc("/api/tokens", mb.handleTokens)
	mux.HandleFunc("/api/auth/tokens", mb.handleTokens)

	// 3. Storage endpoints
	mux.HandleFunc("/api/storage/upload", mb.handleStorageUpload)
	mux.HandleFunc("/api/storage/browse", mb.handleStorageBrowse)
	mux.HandleFunc("/api/storage/objects/", mb.handleStorageObject)

	// 4. SQL statements endpoints
	mux.HandleFunc("/api/sql/statements", mb.handleSqlStatements)

	// 5. Catalog endpoints
	mux.HandleFunc("/api/catalogs", mb.handleCatalogs)
	mux.HandleFunc("/api/catalogs/", mb.handleCatalogs)
	mux.HandleFunc("/api/catalogs/manifest", mb.handleCatalogManifest)

	// 6. Workspaces
	mux.HandleFunc("/api/workspaces", mb.handleWorkspaces)

	// 7. Lakeset BI Studio endpoints (#39)
	mux.HandleFunc("/api/v1/lakeset/status", mb.handleLakesetStatus)
	mux.HandleFunc("/api/v1/lakeset/datasets", mb.handleLakesetDatasets)
	mux.HandleFunc("/api/v1/lakeset/sync", mb.handleLakesetSync)
	mux.HandleFunc("/api/v1/lakeset/guest-token", mb.handleLakesetGuestToken)
	mux.HandleFunc("/api/v1/lakeset/explore-data", mb.handleLakesetExploreData)

	// 8. Secrets & Admin Vaults endpoints (#40)
	mux.HandleFunc("/api/secrets/scopes", mb.handleSecretsScopes)
	mux.HandleFunc("/api/secrets/scopes/", mb.handleSecretsScopes)
	mux.HandleFunc("/api/admin/vaults", mb.handleAdminVaults)

	// 9. Webhooks & Pipeline Webhooks (#41)
	mux.HandleFunc("/api/v1/webhooks/", mb.handleWebhooks)
	mux.HandleFunc("/api/v1/pipelines/", mb.handlePipelinesV1)

	// 10. AI Gateway, Retrieval & Admin AI Providers (#42)
	mux.HandleFunc("/api/v1/retrieve", mb.handleAIRetrieve)
	mux.HandleFunc("/api/v1/vector_stores", mb.handleVectorStores)
	mux.HandleFunc("/api/v1/models", mb.handleAIModels)
	mux.HandleFunc("/api/v1/embeddings", mb.handleAIEmbeddings)
	mux.HandleFunc("/api/admin/ai-provider", mb.handleAdminAIProvider)

	// 11. Remote MCP Server (#43)
	mux.HandleFunc("/api/mcp", mb.handleMCP)

	mb.Server = httptest.NewServer(mux)
	mb.URL = mb.Server.URL
	return mb
}

func (mb *MockBackend) Close() {
	if mb.Server != nil {
		mb.Server.Close()
	}
}

func (mb *MockBackend) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req client.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "admin@thelakeoftears.com" && req.Password == "secret123" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.LoginResponse{
			Token: "lot_jwt_e2e_session_token",
			User: struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			}{
				Email: req.Email,
				Role:  "admin",
			},
		})
		return
	}

	http.Error(w, `{"error": "invalid email or password"}`, http.StatusUnauthorized)
}

func (mb *MockBackend) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client.User{
		ID:       "usr-admin-1",
		Email:    "admin@thelakeoftears.com",
		Role:     "admin",
		IsActive: true,
	})
}

func (mb *MockBackend) handleTokens(w http.ResponseWriter, r *http.Request) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(mb.tokens)
	case http.MethodPost:
		var req client.TokenCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid token create payload", http.StatusBadRequest)
			return
		}
		newPAT := client.PersonalAccessToken{
			ID:          fmt.Sprintf("tok-%d", time.Now().UnixNano()),
			Name:        req.Name,
			TokenPrefix: "lot_pat_e2e...",
			Scopes:      req.Scopes,
			CreatedAt:   time.Now(),
		}
		mb.tokens = append(mb.tokens, newPAT)

		_ = json.NewEncoder(w).Encode(client.TokenCreateResponse{
			ID:          newPAT.ID,
			Name:        newPAT.Name,
			Token:       "lot_pat_e2e_abcdef0123456789",
			TokenPrefix: newPAT.TokenPrefix,
			Scopes:      newPAT.Scopes,
			CreatedAt:   newPAT.CreatedAt,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (mb *MockBackend) handleStorageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file form parameter", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prefix := r.FormValue("prefix")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	fullKey := prefix + header.Filename

	mb.mu.Lock()
	mb.files[fullKey] = data
	mb.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client.StorageUploadResponse{
		Key:         fullKey,
		RelKey:      header.Filename,
		Size:        int64(len(data)),
		ETag:        `"d41d8cd98f00b204e9800998ecf8427e"`,
		ContentType: "text/plain",
		Status:      "uploaded",
	})
}

func (mb *MockBackend) handleStorageBrowse(w http.ResponseWriter, r *http.Request) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	prefix := r.URL.Query().Get("prefix")
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	resp := client.StorageBrowseResponse{
		Bucket:  "lake-data",
		Prefix:  prefix,
		Files:   make([]client.StorageFile, 0),
		Folders: make([]client.StorageFolder, 0),
	}

	for k, v := range mb.files {
		if strings.HasPrefix(k, prefix) || prefix == "" {
			fileName := strings.TrimPrefix(k, prefix)
			now := time.Now()
			resp.Files = append(resp.Files, client.StorageFile{
				Key:           k,
				RelKey:        fileName,
				Name:          fileName,
				Size:          int64(len(v)),
				SizeFormatted: fmt.Sprintf("%d B", len(v)),
				LastModified:  &now,
				ETag:          `"mock-etag"`,
			})
		}
	}
	resp.TotalFiles = len(resp.Files)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (mb *MockBackend) handleStorageObject(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.TrimPrefix(r.URL.Path, "/api/storage/objects/")

	mb.mu.RLock()
	defer mb.mu.RUnlock()

	if strings.HasSuffix(cleanPath, "/metadata") {
		key := strings.TrimSuffix(cleanPath, "/metadata")
		data, exists := mb.files[key]
		if !exists {
			http.Error(w, "object not found", http.StatusNotFound)
			return
		}
		now := time.Now()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.StorageMetadataResponse{
			Key:          key,
			RelKey:       key,
			Size:         int64(len(data)),
			LastModified: &now,
			ContentType:  "text/plain",
			ETag:         `"mock-etag"`,
		})
		return
	}

	if strings.HasSuffix(cleanPath, "/stream") {
		key := strings.TrimSuffix(cleanPath, "/stream")
		data, exists := mb.files[key]
		if !exists {
			http.Error(w, "object not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	http.Error(w, "endpoint not found", http.StatusNotFound)
}

func (mb *MockBackend) handleSqlStatements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req client.StatementExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mb.mu.Lock()
	mb.lastQuery = req.Statement
	mb.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch req.Format {
	case "CSV":
		_ = json.NewEncoder(w).Encode(client.StatementExecutionResponse{
			StatementID: "stmt-e2e-1",
			Status:      client.StatementStatus{State: "SUCCEEDED"},
			Result: &client.StatementResultChunk{
				CSVData:  "num,name\n1,lake\n",
				RowCount: 1,
			},
		})
	case "JSON_OBJECT":
		_ = json.NewEncoder(w).Encode(client.StatementExecutionResponse{
			StatementID: "stmt-e2e-1",
			Status:      client.StatementStatus{State: "SUCCEEDED"},
			Result: &client.StatementResultChunk{
				DataObjects: []map[string]interface{}{
					{"num": 1, "name": "lake"},
				},
				RowCount: 1,
			},
		})
	default:
		_ = json.NewEncoder(w).Encode(client.StatementExecutionResponse{
			StatementID: "stmt-e2e-1",
			Status:      client.StatementStatus{State: "SUCCEEDED"},
			Manifest: &client.StatementManifest{
				Format: "JSON_ARRAY",
				Schema: client.StatementSchemaInfo{
					ColumnCount: 2,
					Columns: []client.StatementColumnInfo{
						{Name: "num", TypeText: "INTEGER", Position: 0},
						{Name: "name", TypeText: "VARCHAR", Position: 1},
					},
				},
				TotalRowCount: 1,
			},
			Result: &client.StatementResultChunk{
				DataArray: [][]interface{}{
					{1, "lake"},
				},
				RowCount: 1,
			},
		})
	}
}

func (mb *MockBackend) handleCatalogs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/catalogs")
	w.Header().Set("Content-Type", "application/json")

	if path == "" || path == "/" {
		_ = json.NewEncoder(w).Encode([]client.Catalog{
			{
				ID:          "cat-default",
				Name:        "default_catalog",
				Slug:        "default_catalog",
				Description: "Primary lakehouse catalog",
				IsDefault:   true,
			},
		})
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 {
		// /api/catalogs/:id
		_ = json.NewEncoder(w).Encode(client.Catalog{
			ID:          parts[0],
			Name:        "default_catalog",
			Slug:        "default_catalog",
			Description: "Primary lakehouse catalog",
			IsDefault:   true,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "schemas" {
		// /api/catalogs/:id/schemas
		_ = json.NewEncoder(w).Encode([]client.CatalogSchema{
			{
				ID:        "sch-analytics",
				CatalogID: parts[0],
				Name:      "analytics",
				Slug:      "analytics",
			},
		})
		return
	}

	if len(parts) == 4 && parts[1] == "schemas" && parts[3] == "tables" {
		// /api/catalogs/:id/schemas/:schema_id/tables
		_ = json.NewEncoder(w).Encode([]client.CatalogTable{
			{
				ID:       "tbl-events",
				SchemaID: parts[2],
				Name:     "events",
				Slug:     "events",
			},
		})
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (mb *MockBackend) handleCatalogManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]map[string]interface{}{
		{
			"catalog": "default_catalog",
			"schemas": []map[string]interface{}{
				{
					"schema": "analytics",
					"tables": []string{"events"},
				},
			},
		},
	})
}

func (mb *MockBackend) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]client.Workspace{
		{
			ID:     "ws-default",
			Name:   "default",
			Slug:   "default",
			Status: "active",
		},
	})
}

func (mb *MockBackend) handleLakesetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":                    true,
		"superset_online":       true,
		"database_name":         "Lake of Tears (DuckDB Lakehouse)",
		"engine":                "duckdb",
		"workspace_slug":        "default",
		"datasets_count":        2,
		"synced_datasets_count": 2,
		"dashboards_count":      1,
	})
}

func (mb *MockBackend) handleLakesetDatasets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":             true,
		"workspace_slug": "default",
		"total":          1,
		"datasets": []map[string]any{
			{
				"id":           "ds-events",
				"catalog_slug": "default_catalog",
				"schema_slug":  "analytics",
				"tier":         "gold",
				"table_name":   "events",
				"full_name":    "default_catalog.analytics.events",
				"sql_view":     "analytics.events",
				"column_count": 4,
				"synced":       true,
			},
		},
	})
}

func (mb *MockBackend) handleLakesetSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":                    true,
		"workspace_slug":        "default",
		"database_id":           1,
		"database_name":         "Lake of Tears (DuckDB Lakehouse)",
		"engine":                "duckdb",
		"synced_datasets_count": 1,
	})
}

func (mb *MockBackend) handleLakesetGuestToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":             true,
		"token":          "lot_guest_jwt_e2e_123",
		"expires_in":     3600,
		"workspace_slug": "default",
	})
}

func (mb *MockBackend) handleLakesetExploreData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         true,
		"dataset":    "default_catalog.analytics.events",
		"chart_type": "bar",
		"sql":        "SELECT COUNT(*) FROM analytics.events",
		"data": map[string]any{
			"categories": []string{"all"},
			"series":     []map[string]any{{"name": "COUNT(*)", "data": []float64{42}}},
		},
	})
}

func (mb *MockBackend) handleSecretsScopes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/secrets/scopes")
	if path == "" || path == "/" {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":              "sc-e2e-1",
				"workspace_id":    "ws-default",
				"name":            "prod-secrets",
				"backend_type":    "internal",
				"user_permission": "MANAGE",
				"key_count":       1,
			},
		})
		return
	}
	if strings.HasSuffix(path, "/keys/db_password") {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":     "db_password",
			"value":   "e2e-super-secret-value",
			"version": 1,
		})
		return
	}
	_ = json.NewEncoder(w).Encode([]map[string]any{
		{"key": "db_password", "version": 1},
	})
}

func (mb *MockBackend) handleAdminVaults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]map[string]any{
		{"provider": "internal", "display_name": "PostgreSQL Encrypted Vault", "is_configured": true},
		{"provider": "hashicorp_vault", "display_name": "HashiCorp Vault", "is_configured": true},
	})
}

func (mb *MockBackend) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if strings.HasSuffix(r.URL.Path, "/secret") {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id":   "ws-default",
			"workspace_slug": "default",
			"webhook_secret": "whsec_e2e_secret",
			"webhook_url":    "/api/v1/ingest/webhooks/default",
		})
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":             "accepted",
		"event_id":           "evt-e2e-1",
		"workspace_slug":     "default",
		"triggered_pipeline": r.URL.Query().Get("trigger_pipeline"),
		"dag_run_id":         "webhook__e2e_run",
	})
}

func (mb *MockBackend) handlePipelinesV1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if strings.HasSuffix(r.URL.Path, "/webhook") {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "queued",
			"dag_id":     "telemetry_etl",
			"dag_run_id": "webhook__e2e_pipeline_run",
			"state":      "queued",
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"dag_id":          "telemetry_etl",
		"webhook_enabled": true,
		"webhook_url":     "/api/v1/pipelines/default/telemetry_etl/webhook",
	})
}

func (mb *MockBackend) handleAIRetrieve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"query": "telemetry",
		"total": 1,
		"results": []map[string]any{
			{"chunk_id": "chk-e2e-1", "content": "E2E telemetry chunk", "score": 0.96},
		},
	})
}

func (mb *MockBackend) handleVectorStores(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "vs_default_catalog", "object": "vector_store", "name": "default/default_catalog", "status": "completed"},
		},
	})
}

func (mb *MockBackend) handleAIModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "lot-genie", "object": "model", "owned_by": "lake-of-tears"},
			{"id": "lot-rag", "object": "model", "owned_by": "lake-of-tears"},
		},
	})
}

func (mb *MockBackend) handleAIEmbeddings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"model":  "text-embedding-004",
		"data": []map[string]any{
			{"object": "embedding", "index": 0, "embedding": []float64{0.11, 0.22, 0.33}},
		},
	})
}

func (mb *MockBackend) handleAdminAIProvider(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"active_provider": "vertex",
		"providers": []map[string]any{
			{"id": "vertex", "label": "Google Cloud Vertex AI", "chat_model": "gemini-2.5-flash", "embed_model": "text-embedding-005"},
		},
	})
}

func (mb *MockBackend) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":             "lake-of-tears-mcp",
			"version":          "1.0.0",
			"protocolVersions": []string{"2025-03-26", "2024-11-05"},
			"endpoints": map[string]string{
				"streamable_http": "/api/mcp",
				"sse":             "/api/mcp/sse",
			},
			"tools": []string{"list_workspaces_and_catalogs", "describe_table", "semantic_search_datalake", "query_datalake_sql", "ask_genie"},
		})
		return
	}
	var rpcReq client.MCPJSONRPCRequest
	_ = json.NewDecoder(r.Body).Decode(&rpcReq)
	switch rpcReq.Method {
	case "resources/list":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result": map[string]any{
				"resources": []map[string]any{
					{"uri": "lot://catalogs/default/default_catalog", "name": "default/default_catalog", "mimeType": "application/json"},
				},
			},
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result":  map[string]any{},
		})
	}
}
