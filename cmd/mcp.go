package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var (
	mcpClient     string
	mcpConfigPath string
	mcpTransport  string
	mcpRemote     bool
	mcpPrint      bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) Server, Remote Bridge, and Client Integration",
	Long: `Run Lake of Tears as a Model Context Protocol (MCP) server over standard I/O (stdio),
bridge stdio to the remote /api/mcp HTTP endpoint, inspect remote MCP tools and catalog
resources, or configure MCP clients (Claude Desktop, Antigravity, Cursor).

Examples:
  angler mcp status
  angler mcp tools
  angler mcp resources
  angler mcp read-resource lot://catalogs/default/main
  angler mcp serve
  angler mcp serve --remote
  angler mcp install --client claude
  angler mcp install --client antigravity --transport http
  angler mcp install --client cursor --transport sse --print`,
}

// buildMCPServer configures and registers all tools and resources exposed by the Angler MCP server.
func buildMCPServer(c *client.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"angler",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false),
	)

	// ============================================================================
	// Legacy / Native Angler Tools (angler_*)
	// ============================================================================

	// 1. angler_sql_query
	sqlTool := mcp.NewTool("angler_sql_query",
		mcp.WithDescription("Execute analytical DuckDB SQL queries over Lake of Tears lakehouse tables, views, and Parquet storage"),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The SQL statement to execute")),
		mcp.WithBoolean("read_only", mcp.Description("Enforce read-only analytical queries (default: true)")),
	)
	s.AddTool(sqlTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sqlQuery, err := request.RequireString("sql")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		res, err := c.ExecuteStatement(client.StatementExecutionRequest{
			Statement: sqlQuery,
			Format:    "JSON_OBJECT",
			RowLimit:  500,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("SQL execution error: %v", err)), nil
		}

		if res.Status.State == "FAILED" && res.Status.Error != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Query failed [%s]: %s", res.Status.Error.ErrorCode, res.Status.Error.Message)), nil
		}

		outBytes, _ := json.MarshalIndent(res.Result, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 2. angler_catalog_inspect
	catalogTool := mcp.NewTool("angler_catalog_inspect",
		mcp.WithDescription("Inspect Lake of Tears Unity-style data catalogs, schemas, tables, and column metadata"),
		mcp.WithString("catalog", mcp.Description("Optional catalog ID or name")),
		mcp.WithString("schema", mcp.Description("Optional schema ID or name (requires catalog)")),
		mcp.WithString("table", mcp.Description("Optional table ID or name (requires catalog and schema)")),
	)
	s.AddTool(catalogTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		catID := request.GetString("catalog", "")
		schID := request.GetString("schema", "")
		tblID := request.GetString("table", "")

		if catID != "" && schID != "" && tblID != "" {
			tbl, err := c.GetTable(catID, schID, tblID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get table: %v", err)), nil
			}
			outBytes, _ := json.MarshalIndent(tbl, "", "  ")
			return mcp.NewToolResultText(string(outBytes)), nil
		}

		if catID != "" && schID != "" {
			tables, err := c.ListTables(catID, schID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list tables: %v", err)), nil
			}
			outBytes, _ := json.MarshalIndent(tables, "", "  ")
			return mcp.NewToolResultText(string(outBytes)), nil
		}

		if catID != "" {
			schemas, err := c.ListSchemas(catID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list schemas: %v", err)), nil
			}
			outBytes, _ := json.MarshalIndent(schemas, "", "  ")
			return mcp.NewToolResultText(string(outBytes)), nil
		}

		catalogs, err := c.ListCatalogs()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list catalogs: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(catalogs, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 3. angler_fs_browse
	fsTool := mcp.NewTool("angler_fs_browse",
		mcp.WithDescription("Browse files, directories, and object storage paths in Lake of Tears volumes (S3/MinIO)"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Object storage volume path (e.g. gold/telemetry)")),
	)
	s.AddTool(fsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		browseRes, err := c.BrowseStorage(path, 100, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to browse storage: %v", err)), nil
		}

		outBytes, _ := json.MarshalIndent(browseRes, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 4. angler_workspace_read_notebook
	notebookTool := mcp.NewTool("angler_workspace_read_notebook",
		mcp.WithDescription("Fetch notebook cells, scripts, or files from the Lake of Tears workspace directory"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to notebook in workspace (e.g. analysis/report.ipynb)")),
	)
	s.AddTool(notebookTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		nb, err := c.GetNotebook(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read notebook: %v", err)), nil
		}

		outBytes, _ := json.MarshalIndent(nb, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 5. angler_jobs_trigger
	jobsTool := mcp.NewTool("angler_jobs_trigger",
		mcp.WithDescription("Trigger immediate execution of a scheduled data pipeline or Airflow DAG"),
		mcp.WithString("job_id", mcp.Required(), mcp.Description("Pipeline or DAG identifier (e.g. telemetry_etl)")),
		mcp.WithString("conf", mcp.Description("Optional JSON configuration payload for the pipeline run")),
	)
	s.AddTool(jobsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		jobID, err := request.RequireString("job_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var confMap map[string]any
		confStr := request.GetString("conf", "")
		if confStr != "" {
			_ = json.Unmarshal([]byte(confStr), &confMap)
		}

		runResp, err := c.TriggerPipeline(jobID, confMap)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger pipeline: %v", err)), nil
		}

		outBytes, _ := json.MarshalIndent(runResp, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 6. angler_ai_genie
	genieTool := mcp.NewTool("angler_ai_genie",
		mcp.WithDescription("Query Lake of Tears Genie AI grounded in catalog tables, schemas, and semantic embeddings"),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("Natural language question or request")),
		mcp.WithString("mode", mcp.Description("Query mode: auto, sql, rag, clarify (default: auto)")),
	)
	s.AddTool(genieTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt, err := request.RequireString("prompt")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		mode := request.GetString("mode", "auto")

		chatResp, err := c.GenieChat(client.GenieChatRequest{
			Question: prompt,
			Mode:     mode,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Genie AI error: %v", err)), nil
		}

		outBytes, _ := json.MarshalIndent(chatResp, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// ============================================================================
	// Canonical Lake of Tears /api/mcp Tools
	// ============================================================================

	// 7. list_workspaces_and_catalogs
	listWsAndCatsTool := mcp.NewTool("list_workspaces_and_catalogs",
		mcp.WithDescription("List all Lake of Tears workspaces, catalogs, schemas, and tables that the authenticated user is authorized to access"),
		mcp.WithString("workspace_slug", mcp.Description("Optional workspace slug to filter results")),
	)
	s.AddTool(listWsAndCatsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wsSlug := request.GetString("workspace_slug", "")
		// Try remote /api/mcp tool call first; fall back to REST catalog listing
		if remoteRes, err := c.MCPCallTool("list_workspaces_and_catalogs", map[string]any{"workspace_slug": wsSlug}); err == nil && len(remoteRes.Content) > 0 {
			if remoteRes.IsError {
				return mcp.NewToolResultError(remoteRes.Content[0].Text), nil
			}
			return mcp.NewToolResultText(remoteRes.Content[0].Text), nil
		}
		cats, err := c.ListCatalogs()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list catalogs: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(map[string]any{"ok": true, "catalogs": cats}, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 8. describe_table
	describeTableTool := mcp.NewTool("describe_table",
		mcp.WithDescription("Describe a specific Lake of Tears catalog table, returning its column definitions, semantic descriptions, and vector index status"),
		mcp.WithString("catalog", mcp.Required(), mcp.Description("Catalog slug or name")),
		mcp.WithString("schema", mcp.Required(), mcp.Description("Schema slug or name")),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table slug or name")),
		mcp.WithString("workspace_slug", mcp.Description("Optional workspace slug")),
		mcp.WithBoolean("include_sample", mcp.Description("Include up to 5 sample rows")),
	)
	s.AddTool(describeTableTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cat, err := request.RequireString("catalog")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sch, err := request.RequireString("schema")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tbl, err := request.RequireString("table")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		wsSlug := request.GetString("workspace_slug", "")
		includeSample := request.GetBool("include_sample", false)

		if remoteRes, err := c.MCPCallTool("describe_table", map[string]any{
			"catalog":        cat,
			"schema":         sch,
			"table":          tbl,
			"workspace_slug": wsSlug,
			"include_sample": includeSample,
		}); err == nil && len(remoteRes.Content) > 0 {
			if remoteRes.IsError {
				return mcp.NewToolResultError(remoteRes.Content[0].Text), nil
			}
			return mcp.NewToolResultText(remoteRes.Content[0].Text), nil
		}

		tableObj, err := c.GetTable(cat, sch, tbl)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe table: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(tableObj, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 9. semantic_search_datalake
	semanticSearchTool := mcp.NewTool("semantic_search_datalake",
		mcp.WithDescription("Execute permission-aware semantic vector similarity search across authorized Lake of Tears catalog tables and workspace embeddings"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
		mcp.WithNumber("top_k", mcp.Description("Maximum number of chunks to return (default 5)")),
		mcp.WithString("workspace_slug", mcp.Description("Optional workspace slug filter")),
		mcp.WithString("catalog", mcp.Description("Optional catalog filter")),
		mcp.WithString("schema", mcp.Description("Optional schema filter")),
		mcp.WithString("table", mcp.Description("Optional table filter")),
	)
	s.AddTool(semanticSearchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		topK := request.GetInt("top_k", 5)
		res, err := c.RetrieveVectorChunks(client.VectorRetrieveRequest{
			Query:         q,
			TopK:          topK,
			WorkspaceSlug: request.GetString("workspace_slug", ""),
			Catalog:       request.GetString("catalog", ""),
			SchemaName:    request.GetString("schema", ""),
			Table:         request.GetString("table", ""),
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Semantic search error: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 10. query_datalake_sql
	queryDatalakeSQLTool := mcp.NewTool("query_datalake_sql",
		mcp.WithDescription("Execute a read-only DuckDB SQL query against authorized Lake of Tears catalog tables and views"),
		mcp.WithString("sql", mcp.Required(), mcp.Description("Read-only DuckDB SQL SELECT query")),
		mcp.WithString("workspace_slug", mcp.Description("Optional workspace slug context")),
		mcp.WithNumber("max_rows", mcp.Description("Maximum number of rows to return (default 100)")),
	)
	s.AddTool(queryDatalakeSQLTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sqlQuery, err := request.RequireString("sql")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxRows := request.GetInt("max_rows", 100)
		res, err := c.ExecuteStatement(client.StatementExecutionRequest{
			Statement: sqlQuery,
			Format:    "JSON_OBJECT",
			RowLimit:  maxRows,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("SQL execution error: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(res.Result, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 11. ask_genie
	askGenieTool := mcp.NewTool("ask_genie",
		mcp.WithDescription("Ask the Lake of Tears Genie AI analyst a natural language question (Text-to-SQL synthesis + DuckDB execution + narrative insight, or semantic RAG)"),
		mcp.WithString("question", mcp.Required(), mcp.Description("Natural language analytical question")),
		mcp.WithString("mode", mcp.Description("Execution mode: genie, rag, or hybrid (default: genie)")),
		mcp.WithString("workspace_slug", mcp.Description("Optional workspace slug to scope the query")),
	)
	s.AddTool(askGenieTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := request.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		mode := request.GetString("mode", "genie")
		chatResp, err := c.GenieChat(client.GenieChatRequest{
			Question: question,
			Mode:     mode,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Genie AI error: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(chatResp, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// ============================================================================
	// Lakeset BI & Secret Scopes Tools
	// ============================================================================

	// 12. angler_lakeset_datasets
	lakesetDatasetsTool := mcp.NewTool("angler_lakeset_datasets",
		mcp.WithDescription("List RBAC-authorized Lakeset BI Studio datasets and DuckDB semantic views"),
		mcp.WithString("catalog", mcp.Description("Optional catalog slug filter")),
		mcp.WithString("tier", mcp.Description("Optional Medallion tier filter: bronze, silver, or gold")),
	)
	s.AddTool(lakesetDatasetsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.ListLakesetDatasets(client.LakesetDatasetsFilter{
			Catalog: request.GetString("catalog", ""),
			Tier:    request.GetString("tier", ""),
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list Lakeset datasets: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 13. angler_lakeset_explore
	lakesetExploreTool := mcp.NewTool("angler_lakeset_explore",
		mcp.WithDescription("Execute an interactive Lakeset BI chart aggregation query over a DuckDB dataset"),
		mcp.WithString("dataset", mcp.Required(), mcp.Description("Dataset full name (e.g. main.gold.revenue)")),
		mcp.WithString("dimension", mcp.Description("Grouping dimension column")),
		mcp.WithString("metric", mcp.Description("Metric column to aggregate")),
		mcp.WithString("aggregation", mcp.Description("Aggregation function: COUNT, SUM, AVG, MIN, MAX (default: COUNT)")),
		mcp.WithString("chart_type", mcp.Description("Chart type: bar, line, area, pie, table (default: bar)")),
	)
	s.AddTool(lakesetExploreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ds, err := request.RequireString("dataset")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.ExploreLakesetData(client.LakesetExploreRequest{
			Dataset:   ds,
			ChartType: request.GetString("chart_type", "bar"),
			Dimension: request.GetString("dimension", ""),
			Metric:    request.GetString("metric", ""),
			MetricAgg: request.GetString("aggregation", "COUNT"),
			Limit:     100,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Lakeset explore error: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// 14. angler_secrets_list
	secretsListTool := mcp.NewTool("angler_secrets_list",
		mcp.WithDescription("List secret scopes or secret key metadata (without exposing raw secret values) in the active workspace"),
		mcp.WithString("scope", mcp.Description("Optional scope name to list keys within that scope")),
	)
	s.AddTool(secretsListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scope := request.GetString("scope", "")
		if scope != "" {
			keys, err := c.ListSecretKeys(scope)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list secret keys: %v", err)), nil
			}
			outBytes, _ := json.MarshalIndent(keys, "", "  ")
			return mcp.NewToolResultText(string(outBytes)), nil
		}
		scopes, err := c.ListSecretScopes()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list secret scopes: %v", err)), nil
		}
		outBytes, _ := json.MarshalIndent(scopes, "", "  ")
		return mcp.NewToolResultText(string(outBytes)), nil
	})

	// ============================================================================
	// MCP Resource Template: lot://catalogs/{workspace_slug}/{catalog_slug}
	// ============================================================================
	tmpl := mcp.NewResourceTemplate(
		"lot://catalogs/{workspace_slug}/{catalog_slug}",
		"Lake of Tears Catalog Resource",
		mcp.WithTemplateDescription("RBAC-scoped Lake of Tears catalog metadata and schema/table definitions"),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.AddResourceTemplate(tmpl, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		uri := request.Params.URI
		contents, err := c.MCPReadResource(uri)
		if err != nil {
			return nil, err
		}
		var out []mcp.ResourceContents
		for _, item := range contents {
			out = append(out, mcp.TextResourceContents{
				URI:      item.URI,
				MIMEType: item.MimeType,
				Text:     item.Text,
			})
		}
		return out, nil
	})

	return s
}

// serveRemoteMCPBridge forwards line-delimited JSON-RPC 2.0 requests from in to the remote /api/mcp endpoint and writes responses to out.
func serveRemoteMCPBridge(c *client.Client, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Allow up to 10MB JSON-RPC messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		respBytes, statusCode, err := c.MCPForwardRaw(line)
		if err != nil {
			errResp := map[string]any{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": map[string]any{
					"code":    -32603,
					"message": fmt.Sprintf("remote MCP bridge transport error: %v", err),
				},
			}
			b, _ := json.Marshal(errResp)
			_, _ = fmt.Fprintf(out, "%s\n", string(b))
			continue
		}
		if statusCode == 202 || len(bytes.TrimSpace(respBytes)) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s\n", strings.TrimSpace(string(respBytes)))
	}
	return scanner.Err()
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start native Model Context Protocol (MCP) server over standard I/O (stdio)",
	Long: `Starts a stdio-based MCP server providing LLM tools and catalog resources for Lake of Tears.
Pass --remote to bridge stdio JSON-RPC 2.0 messages directly to the remote /api/mcp endpoint.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		if mcpRemote {
			return serveRemoteMCPBridge(c, os.Stdin, os.Stdout)
		}
		mcpServer := buildMCPServer(c)
		return server.ServeStdio(mcpServer)
	},
}

var mcpStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"info"},
	Short:   "Inspect the remote Lake of Tears /api/mcp server status, protocol versions, and endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		st, err := c.GetMCPStatus()
		if err != nil {
			return err
		}

		return ui.RenderOutput(st, func() {
			headers := []string{"PROPERTY", "VALUE"}
			rows := [][]string{
				{"Server Name", st.Name},
				{"Version", st.Version},
				{"Protocol Versions", strings.Join(st.ProtocolVersions, ", ")},
				{"Streamable HTTP", st.Endpoints["streamable_http"]},
				{"SSE Endpoint", st.Endpoints["sse"]},
				{"Messages Endpoint", st.Endpoints["messages"]},
				{"Registered Tools", fmt.Sprintf("%d (%s)", len(st.Tools), strings.Join(st.Tools, ", "))},
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		})
	},
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List MCP tools exposed by the remote Lake of Tears /api/mcp server",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		tools, err := c.MCPListTools()
		if err != nil {
			return err
		}

		return ui.RenderOutput(tools, func() {
			headers := []string{"TOOL NAME", "DESCRIPTION"}
			var rows [][]string
			for _, t := range tools {
				desc := t.Description
				if len(desc) > 85 {
					desc = desc[:82] + "..."
				}
				rows = append(rows, []string{t.Name, desc})
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		})
	},
}

var mcpResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List RBAC-authorized catalog resources (lot://catalogs/*) from /api/mcp",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resources, err := c.MCPListResources()
		if err != nil {
			return err
		}

		return ui.RenderOutput(resources, func() {
			headers := []string{"URI", "NAME", "MIME TYPE", "DESCRIPTION"}
			var rows [][]string
			for _, r := range resources {
				rows = append(rows, []string{r.URI, r.Name, r.MimeType, r.Description})
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		})
	},
}

var mcpReadResourceCmd = &cobra.Command{
	Use:   "read-resource <uri>",
	Short: "Read an MCP catalog resource URI (e.g. lot://catalogs/default/main)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		contents, err := c.MCPReadResource(args[0])
		if err != nil {
			return err
		}

		return ui.RenderOutput(contents, func() {
			for _, item := range contents {
				fmt.Fprintln(ui.OutWriter, item.Text)
			}
		})
	},
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Configure MCP clients (Claude Desktop, Antigravity, Cursor) for stdio, SSE, or Streamable HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientTarget := strings.ToLower(strings.TrimSpace(mcpClient))
		if clientTarget == "" {
			clientTarget = "claude"
		}
		transport := strings.ToLower(strings.TrimSpace(mcpTransport))
		if transport == "" {
			transport = "stdio"
		}
		if transport != "stdio" && transport != "sse" && transport != "http" {
			return fmt.Errorf("unsupported --transport %q (supported: stdio, sse, http)", transport)
		}

		c := getClient()

		// Resolve executable binary path
		binPath, err := os.Executable()
		if err != nil {
			binPath = "angler"
		} else {
			binPath, _ = filepath.EvalSymlinks(binPath)
		}

		var serverEntry map[string]any
		switch transport {
		case "stdio":
			serveArgs := []string{"mcp", "serve"}
			if mcpRemote {
				serveArgs = append(serveArgs, "--remote")
			}
			serverEntry = map[string]any{
				"command": binPath,
				"args":    serveArgs,
			}
		case "sse":
			entry := map[string]any{
				"type": "sse",
				"url":  strings.TrimRight(c.Endpoint, "/") + "/api/mcp/sse",
			}
			if c.Token != "" {
				entry["headers"] = map[string]string{
					"Authorization": "Bearer " + c.Token,
				}
			}
			serverEntry = entry
		case "http":
			entry := map[string]any{
				"type": "streamable-http",
				"url":  strings.TrimRight(c.Endpoint, "/") + "/api/mcp",
			}
			if c.Token != "" {
				entry["headers"] = map[string]string{
					"Authorization": "Bearer " + c.Token,
				}
			}
			serverEntry = entry
		}

		if mcpPrint {
			snippet := map[string]any{
				"mcpServers": map[string]any{
					"angler": serverEntry,
				},
			}
			b, _ := json.MarshalIndent(snippet, "", "  ")
			fmt.Fprintln(ui.OutWriter, string(b))
			return nil
		}

		configPath := mcpConfigPath
		if configPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to determine user home directory: %w", err)
			}

			switch clientTarget {
			case "claude":
				switch runtime.GOOS {
				case "darwin":
					configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
				case "windows":
					appData := os.Getenv("APPDATA")
					if appData == "" {
						appData = filepath.Join(homeDir, "AppData", "Roaming")
					}
					configPath = filepath.Join(appData, "Claude", "claude_desktop_config.json")
				default:
					configPath = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
				}
			case "antigravity":
				configPath = filepath.Join(homeDir, ".gemini", "antigravity-cli", "mcp", "angler.json")
			case "cursor":
				configPath = filepath.Join(homeDir, ".cursor", "mcp.json")
			default:
				return fmt.Errorf("unsupported client %q (supported: claude, antigravity, cursor)", clientTarget)
			}
		}

		// Create parent directory if needed
		parentDir := filepath.Dir(configPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory %s: %w", parentDir, err)
		}

		// Load existing or initialize config structure
		var configMap map[string]any
		data, err := os.ReadFile(configPath)
		if err == nil {
			_ = json.Unmarshal(data, &configMap)
		}
		if configMap == nil {
			configMap = make(map[string]any)
		}

		mcpServers, ok := configMap["mcpServers"].(map[string]any)
		if !ok || mcpServers == nil {
			mcpServers = make(map[string]any)
		}

		mcpServers["angler"] = serverEntry
		configMap["mcpServers"] = mcpServers

		updatedData, err := json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format config json: %w", err)
		}

		if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to write config to %s: %w", configPath, err)
		}

		ui.PrintSuccess("Angler MCP server (%s) successfully configured for %s!", transport, clientTarget)
		fmt.Fprintf(ui.OutWriter, "Config file: %s\n", configPath)
		if transport == "stdio" {
			fmt.Fprintf(ui.OutWriter, "Command:     %s mcp serve\n", binPath)
		} else {
			fmt.Fprintf(ui.OutWriter, "Endpoint:    %v\n", serverEntry["url"])
		}
		return nil
	},
}

func init() {
	mcpServeCmd.Flags().BoolVar(&mcpRemote, "remote", false, "Bridge stdio JSON-RPC messages directly to the remote /api/mcp endpoint")

	mcpInstallCmd.Flags().StringVar(&mcpClient, "client", "claude", "Target client: 'claude', 'antigravity', or 'cursor'")
	mcpInstallCmd.Flags().StringVar(&mcpConfigPath, "config-path", "", "Explicit path to client configuration JSON file")
	mcpInstallCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "MCP transport mode: 'stdio', 'sse', or 'http'")
	mcpInstallCmd.Flags().BoolVar(&mcpRemote, "remote", false, "Use 'angler mcp serve --remote' when transport is stdio")
	mcpInstallCmd.Flags().BoolVar(&mcpPrint, "print", false, "Print generated MCP configuration JSON to stdout without writing to disk")

	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpStatusCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpResourcesCmd)
	mcpCmd.AddCommand(mcpReadResourceCmd)
	mcpCmd.AddCommand(mcpInstallCmd)

	rootCmd.AddCommand(mcpCmd)
}
