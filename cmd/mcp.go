package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) Server and Developer Environment Integration",
	Long: `Run Lake of Tears as a Model Context Protocol (MCP) server over standard I/O (stdio),
or automatically configure AI developer environments (Claude Desktop, Antigravity, Cursor).

Enables LLM agents and assistant environments to execute analytical DuckDB queries,
inspect lakehouse catalogs and schemas, browse object store volumes, read notebooks,
trigger pipelines, and query Genie AI.

Examples:
  angler mcp serve
  angler mcp install --client claude
  angler mcp install --client antigravity
  angler mcp install --client cursor`,
}

// buildMCPServer configures and registers all tools exposed by the Angler MCP server.
func buildMCPServer(c *client.Client) *server.MCPServer {
	s := server.NewMCPServer("angler", "1.0.0", server.WithToolCapabilities(true))

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

	return s
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start native Model Context Protocol (MCP) server over standard I/O (stdio)",
	Long: `Starts a stdio-based MCP server providing LLM tools for Lake of Tears.
Designed to be invoked by Claude Desktop, Antigravity, Cursor, Continue, or custom agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		mcpServer := buildMCPServer(c)
		return server.ServeStdio(mcpServer)
	},
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Automatically configure Claude Desktop, Antigravity, or Cursor for Angler MCP",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientTarget := strings.ToLower(strings.TrimSpace(mcpClient))
		if clientTarget == "" {
			clientTarget = "claude"
		}

		// Resolve executable binary path
		binPath, err := os.Executable()
		if err != nil {
			binPath = "angler"
		} else {
			binPath, _ = filepath.EvalSymlinks(binPath)
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

		mcpServers["angler"] = map[string]any{
			"command": binPath,
			"args":    []string{"mcp", "serve"},
		}
		configMap["mcpServers"] = mcpServers

		updatedData, err := json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format config json: %w", err)
		}

		if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to write config to %s: %w", configPath, err)
		}

		ui.PrintSuccess("Angler MCP server successfully configured for %s!", clientTarget)
		fmt.Fprintf(ui.OutWriter, "Config file: %s\n", configPath)
		fmt.Fprintf(ui.OutWriter, "Command:     %s mcp serve\n", binPath)
		return nil
	},
}

func init() {
	mcpInstallCmd.Flags().StringVar(&mcpClient, "client", "claude", "Target client: 'claude', 'antigravity', or 'cursor'")
	mcpInstallCmd.Flags().StringVar(&mcpConfigPath, "config-path", "", "Explicit path to client configuration JSON file")

	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpInstallCmd)

	rootCmd.AddCommand(mcpCmd)
}
