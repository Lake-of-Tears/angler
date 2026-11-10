## angler

The official CLI for Lake of Tears data lakehouse

### Synopsis

🎣 angler: Cast lines into your Lake of Tears data lakehouse.
Query DuckDB over S3, manage cluster deployments, browse data catalogs, 
and run AI semantic searches directly from your terminal.

### Options

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --force                 bypass confirmation prompts
  -h, --help                  help for angler
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
      --on-behalf-of string   impersonate target user email via trusted superadmin service token (X-On-Behalf-Of)
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler admin](angler_admin.md)	 - Lake of Tears System Administration, Settings, and Session Auditing
* [angler ai](angler_ai.md)	 - Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)
* [angler ask](angler_ask.md)	 - Ask questions in natural language with AI search (alias: sonar)
* [angler auth](angler_auth.md)	 - Manage authentication with Lake of Tears
* [angler bundle](angler_bundle.md)	 - Develop, validate, deploy, and manage Lake Asset Bundles (LABs)
* [angler catalog](angler_catalog.md)	 - Unity Catalog-style metadata discovery and management (alias: net)
* [angler completion](angler_completion.md)	 - Generate the autocompletion script for the specified shell
* [angler compute](angler_compute.md)	 - Manage and inspect Lake of Tears cluster compute engines and topology
* [angler context](angler_context.md)	 - Manage connection environments and profiles
* [angler dashboards](angler_dashboards.md)	 - Lake of Tears BI Dashboards, Canvas Layouts, and Visual Widgets
* [angler docs](angler_docs.md)	 - Generate CLI reference documentation and shell autocompletions
* [angler down](angler_down.md)	 - Stop and dock the Lake of Tears stack (alias: dock)
* [angler fs](angler_fs.md)	 - Filesystem and storage management modeled after Databricks FS (alias: storage)
* [angler jobs](angler_jobs.md)	 - Data pipelines, scheduled jobs, and workflow DAGs management (alias: pipelines)
* [angler lakeset](angler_lakeset.md)	 - Native Zero-Login Lakeset BI Studio, DuckDB Semantic Sync & Embedded Analytics (alias: bi)
* [angler mcp](angler_mcp.md)	 - Model Context Protocol (MCP) Server, Remote Bridge, and Client Integration
* [angler query](angler_query.md)	 - Execute SQL query against the data lake (alias: cast)
* [angler secrets](angler_secrets.md)	 - Enterprise Secret Scopes, Encrypted Keys, and Scope ACLs Management
* [angler sql](angler_sql.md)	 - Unified SQL statement execution, history, and AI assistance (aliases: query, cast)
* [angler status](angler_status.md)	 - Check the health and status of the Lake of Tears instance
* [angler tokens](angler_tokens.md)	 - Manage Personal Access Tokens (PATs) for Lake of Tears
* [angler tool-defs](angler_tool-defs.md)	 - Export LLM tool calling schemas (Gemini, OpenAI, OpenAPI 3.1)
* [angler up](angler_up.md)	 - Launch local Lake of Tears stack (alias: launch)
* [angler users](angler_users.md)	 - Lake of Tears User and Identity Management (IAM)
* [angler version](angler_version.md)	 - Print the angler version
* [angler webhooks](angler_webhooks.md)	 - Workspace Webhook Ingestion, HMAC Signing, Secret Rotation & Downstream Pipeline Triggering
* [angler workspace](angler_workspace.md)	 - Workspace notebook and directory management modeled after Databricks Workspace
* [angler workspaces](angler_workspaces.md)	 - Lake of Tears Multi-Tenant Workspace Provisioning and IAM Members

