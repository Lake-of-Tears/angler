package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestMCPClientMethods(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":             "lake-of-tears-mcp",
				"version":          "1.0.0",
				"protocolVersions": []string{"2025-03-26", "2024-11-05"},
				"endpoints": map[string]string{
					"streamable_http": "/api/mcp",
					"sse":             "/api/mcp/sse",
				},
				"tools": []string{"list_workspaces_and_catalogs", "describe_table"},
			})
			return
		}

		var rpcReq MCPJSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch rpcReq.Method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}, "resources": map[string]any{}},
					"serverInfo":      map[string]any{"name": "lake-of-tears-mcp", "version": "1.0.0"},
				},
			})
		case "ping":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result":  map[string]any{},
			})
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "list_workspaces_and_catalogs",
							"description": "List authorized workspaces and catalogs",
							"inputSchema": map[string]any{"type": "object"},
						},
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
							"description": "Catalog main in workspace default",
							"mimeType":    "application/json",
						},
					},
				},
			})
		case "resources/read":
			uri, _ := rpcReq.Params["uri"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]any{
					"contents": []map[string]any{
						{
							"uri":      uri,
							"mimeType": "application/json",
							"text":     `{"name":"main","slug":"main"}`,
						},
					},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(config.Context{
		Endpoint:   srv.URL,
		Token:      "tok-mcp",
		Workspace:  "default",
		OnBehalfOf: "analyst@lake.io",
	})

	st, err := c.GetMCPStatus()
	if err != nil {
		t.Fatalf("GetMCPStatus failed: %v", err)
	}
	if st.Name != "lake-of-tears-mcp" || len(st.Tools) != 2 {
		t.Fatalf("unexpected MCP status: %+v", st)
	}

	initRes, err := c.MCPInitialize("")
	if err != nil {
		t.Fatalf("MCPInitialize failed: %v", err)
	}
	if initRes.ServerInfo.Name != "lake-of-tears-mcp" {
		t.Fatalf("unexpected initialize server name: %+v", initRes)
	}

	if err := c.MCPPing(); err != nil {
		t.Fatalf("MCPPing failed: %v", err)
	}

	tools, err := c.MCPListTools()
	if err != nil {
		t.Fatalf("MCPListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "list_workspaces_and_catalogs" {
		t.Fatalf("unexpected MCP tools: %+v", tools)
	}

	resources, err := c.MCPListResources()
	if err != nil {
		t.Fatalf("MCPListResources failed: %v", err)
	}
	if len(resources) != 1 || resources[0].URI != "lot://catalogs/default/main" {
		t.Fatalf("unexpected MCP resources: %+v", resources)
	}

	contents, err := c.MCPReadResource("lot://catalogs/default/main")
	if err != nil {
		t.Fatalf("MCPReadResource failed: %v", err)
	}
	if len(contents) != 1 || contents[0].URI != "lot://catalogs/default/main" {
		t.Fatalf("unexpected MCP resource contents: %+v", contents)
	}

	rawResp, code, err := c.MCPForwardRaw([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if err != nil {
		t.Fatalf("MCPForwardRaw failed: %v", err)
	}
	if code != http.StatusAccepted || len(rawResp) != 0 {
		t.Fatalf("expected 202 empty body for notification, got %d %q", code, string(rawResp))
	}
}
