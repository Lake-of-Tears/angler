# angler 🎣

**The official CLI for Lake of Tears — cast queries, launch clusters, inspect catalogs, and reel in data**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/Lake-of-Tears/angler/actions/workflows/ci.yaml/badge.svg)](https://github.com/Lake-of-Tears/angler/actions/workflows/ci.yaml)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/Lake-of-Tears/angler?color=00B4D8)](https://github.com/Lake-of-Tears/angler/releases)

---

## What is Angler?

`angler` is the official command-line interface for the **Lake of Tears** data lakehouse. Designed for data engineers, analysts, and platform operators, `angler` allows you to manage local and cloud cluster deployments, execute high-speed SQL queries across S3 and DuckDB, browse Parquet data catalogs, and cast natural language AI queries right from your terminal.

---

## Installation

### macOS & Linux (Homebrew)

Install directly from the official Lake of Tears tap:

```bash
# Tap and install
brew tap Lake-of-Tears/tap
brew install angler

# Or in a single command
brew install Lake-of-Tears/tap/angler
```

Shell completions for **Bash**, **Zsh**, and **Fish** are automatically configured during installation. See the [Homebrew Distribution Pipeline & Roadmap](docs/distribution/homebrew.md) for architecture details and upstream Homebrew Core tracking.

### Windows (Scoop)

Install using the developer-friendly Scoop bucket:

```powershell
scoop bucket add lake-of-tears https://github.com/Lake-of-Tears/scoop-bucket
scoop install angler
```

### Windows (Chocolatey)

Install using Chocolatey package manager:

```powershell
choco install angler
```

### Linux (Debian, Ubuntu, RedHat, Fedora)

Download native `.deb` or `.rpm` packages directly from [GitHub Releases](https://github.com/Lake-of-Tears/angler/releases):

```bash
# Debian / Ubuntu (x86_64 or ARM64)
sudo dpkg -i angler_*_linux_amd64.deb

# RedHat / Fedora / AlmaLinux
sudo rpm -i angler_*_linux_amd64.rpm
```

### Pre-compiled Standalone Binaries

Download pre-compiled binaries for macOS (Apple Silicon / Intel), Windows (x64 / ARM64), and Linux (x64 / ARM64) directly from the [GitHub Releases](https://github.com/Lake-of-Tears/angler/releases) page:

```bash
# Example: macOS Apple Silicon
curl -sSL https://github.com/Lake-of-Tears/angler/releases/latest/download/angler_darwin_arm64.tar.gz | tar -xz
sudo mv angler /usr/local/bin/
```

### Build from Source

```bash
go install github.com/Lake-of-Tears/angler@latest
```

---

## Quickstart

### 1. Authenticate or Configure Endpoint

Connect to your Lake of Tears instance:

```bash
# Log in with a Personal Access Token (Recommended)
angler auth login --token lot_pat_abcdef0123456789...

# Or pipe token via standard input in CI/CD runners
echo "$LOT_TOKEN" | angler auth login --token-stdin

# Interactive login (prompting for PAT or email & password)
angler auth login

# Or switch context profiles
angler context set production --endpoint https://lake.example.com
angler context use production
```

### 2. Verify Connectivity

```bash
angler status
# or alias:
angler ping
```

### 3. Cast a Query (SQL Statement Execution)

Run DuckDB queries directly over remote Parquet datasets using the unified SQL suite:

```bash
# Direct SQL execution (or use alias: angler cast / angler query)
angler sql "SELECT * FROM lake.telemetry LIMIT 10"

# Execute from file with NDJSON or CSV streaming output
angler sql exec -f queries/report.sql --format csv
angler sql exec -f queries/metrics.sql --format ndjson

# Explain query execution plans
angler sql explain "SELECT department, count(*) FROM employees GROUP BY 1"

# Generate SQL from natural language prompts
angler sql generate "Find top 10 users by spend last month" --run

# Diagnose and fix broken SQL queries
angler sql fix "SELECT empid FROM emps" --error "column empid not found"

# View recent statement history
angler sql history --limit 10
```

### 4. Transmit a Sonar Pulse (AI RAG)

Ask questions grounded in your data using Gemini vector search:

```bash
angler sonar "Which region had the largest revenue drop in Q3?"
```

### 5. Inspect Catalog (`angler catalog`)

Browse and manage Unity Catalog-style data catalogs, schemas, tables, soft-delete trash, TOON projections, cross-workspace sharing, and anomalies:

```bash
# List all registered catalogs (alias: angler net)
angler catalog ls

# Inspect and update catalog details and schemas
angler catalog get main
angler catalog update main --description "Primary Lakehouse Catalog"

# Manage database schemas within a catalog
angler catalog schemas list main
angler catalog schemas get main default
angler catalog schemas create main gold
angler catalog schemas update main gold --description "Curated executive metrics"

# Inspect and update tables, columns, and partition patterns
angler catalog tables list main default
angler catalog tables get main default users
angler catalog tables update main default users --description "Active user accounts"

# Export token-efficient TOON (Token-Oriented Object Notation) schema or table projections
angler catalog toon main default
angler catalog toon main default users --raw

# Soft-Delete Trash & Undrop Recovery
angler catalog trash list
angler catalog undrop <asset_id>
angler catalog trash restore <asset_id> --strategy rename --new-name users_v2
angler catalog trash purge <asset_id> --force
angler catalog trash purge-expired --force

# User-Scoped Catalog Directory & Cross-Workspace Sharing
angler catalog directory
angler catalog access list main
angler catalog access request finance_lake --message "Need Q4 revenue tables"
angler catalog access approve main <access_id>
angler catalog access reject main <access_id>
angler catalog access revoke main <access_id> --force

# Export full catalog manifest projection JSON
angler catalog manifest

# Run automated anomaly detection across catalog tables
angler catalog anomalies detect --table users
```

### 6. Filesystem & Object Storage (`angler fs`)

Browse and manage files in Lake of Tears object stores (S3/MinIO) and workspace volumes:

```bash
# List directory contents
angler fs ls gold/telemetry

# Stream / print file contents (with optional offset and length)
angler fs cat configs/app.json --length 1024

# Copy between local and remote lakehouse storage (lot://, s3://, dbfs://)
angler fs cp ./data.csv lot://bronze/data.csv
angler fs cp lot://gold/report.parquet ./report.parquet

# Inspect object metadata and Parquet schemas
angler fs stat gold/telemetry/data.parquet

# Upload, download, and remove objects
angler fs upload ./model.onnx models/
angler fs download gold/summary.csv ./summary.csv
angler fs rm bronze/staging/ --recursive --force

# View workspace storage metrics
angler fs stats
```

### 7. Workspace Notebooks & Directories (`angler workspace`)

Interact with Lake of Tears notebooks, code directories, and interactive analytical sessions programmatically:

```bash
# List workspace items (alias: angler ws ls)
angler workspace list
angler workspace list analysis/ --recursive

# Import notebooks or scripts (converts .py, .sql, .md into cells)
angler workspace import ./model_training.py analysis/model_training.ipynb --format SOURCE

# Export notebooks to local disk (.ipynb or extracted .py script)
angler workspace export analysis/model_training.ipynb ./model_training.ipynb
angler workspace export analysis/model_training.ipynb - --format SOURCE

# Create directories and delete items
angler workspace mkdirs analysis/v2/experiments
angler workspace delete analysis/legacy_test.ipynb --force

# Headless notebook execution
angler workspace run analysis/model_training.ipynb --kernel python3

# Search workspace items and inspect historical run archives
angler workspace search "duckdb"
angler workspace runs analysis/model_training.ipynb
```

### 8. Pipelines, Jobs & Workflow DAGs (`angler jobs`)

Manage data pipelines, scheduled notebook runs, dbt tasks, and Airflow workflow DAGs:

```bash
# List all active and scheduled pipelines (alias: angler pipelines)
angler jobs list
angler jobs list --all

# Inspect pipeline configuration, DAG dependency graph, and schedule
angler jobs get telemetry_etl

# Trigger immediate execution (with optional --wait blocking and progress streaming)
angler jobs run telemetry_etl
angler jobs run telemetry_etl --wait --timeout 5m

# List past execution runs and inspect task breakdown
angler jobs runs list telemetry_etl
angler jobs runs get telemetry_etl manual__20261108T160000Z__user

# Fetch stdout/stderr task execution logs
angler jobs logs telemetry_etl manual__20261108T160000Z__user extract

# Cancel an in-flight run
angler jobs cancel telemetry_etl manual__20261108T160000Z__user --force

# Pause or unpause scheduled execution
angler jobs pause telemetry_etl
angler jobs unpause telemetry_etl

# Configure and trigger native pipeline inbound webhooks
angler jobs webhook enable telemetry_etl
angler jobs webhook url telemetry_etl
angler jobs webhook trigger telemetry_etl --payload '{"source":"s3","batch":42}'

# Update pipeline cron schedule, schedule a notebook, or view execution timeline
angler jobs schedule telemetry_etl --cron "0 * * * *"
angler jobs schedule-notebook --path analysis/etl.ipynb --cron "@daily" --name daily_etl
angler jobs timeline --date 2026-11-10

# Send HMAC-signed raw webhook events with downstream pipeline triggering
angler webhooks secret get
angler webhooks send --payload '{"event":"order_created"}' --trigger-pipeline telemetry_etl --hmac
```

### 9. Genie AI, Vector Retrieval & Semantic Gateway (`angler ai`)

Converse with Lake of Tears Genie AI grounded in catalog tables and schemas, execute permission-aware vector retrieval, query OpenAI-compatible virtual models and vector stores, and enrich catalog metadata:

```bash
# Natural language data queries and semantic reasoning (alias: angler genie)
angler ai chat "Which tables have customer transaction data?"
angler ai chat "Count total orders in 2026" --no-stream

# Stream reasoning steps and SQL generation in real-time
angler ai chat --stream "Analyze revenue variance between Q2 and Q3"

# Permission-aware vector chunk retrieval with granular catalog/schema/table filters
angler ai retrieve "enterprise churn risk factors" --catalog main --schema gold --top-k 5

# List and search OpenAI-compatible RBAC vector stores
angler ai vector-stores list
angler ai vector-stores search vs_default_main "quarterly revenue"

# List virtual AI models (lot-genie, lot-rag, lot-hybrid) and generate vector embeddings
angler ai models
angler ai embeddings "Lakehouse semantic architecture" --model text-embedding-004

# Generate a catalog table specification from a natural language prompt or enrich table descriptions
angler ai table-from-prompt "Track daily customer churn scores by cohort" --schema gold
angler ai enrich churn_predictions --catalog main --schema gold

# Inspect active workspace grounding context, schemas, and registered SQL AI functions
angler ai context

# List active table vector embeddings and indexes across workspace catalogs
angler ai vector-index list
angler ai vector-index search "high value customers" --catalog main --top-k 5

# Trigger vector index synchronization for a table
angler ai vector-index sync main retail orders

# Trusted proxy user impersonation (superadmin service tokens)
angler ai retrieve "confidential notes" --on-behalf-of analyst@lake.io
```

### 10. Lakeset BI Studio & Embedded Analytics (`angler lakeset`)

Manage the native zero-login Lakeset BI Studio (`/lakeset`), synchronize catalog tables and semantic descriptions to DuckDB + Superset, mint RLS-scoped guest tokens, and run interactive Chart Builder queries:

```bash
# Inspect Lakeset BI Studio status and DuckDB Lakehouse engine health (alias: angler bi)
angler lakeset status

# List RBAC-authorized Lakeset datasets across Medallion tiers
angler lakeset datasets --tier gold --search revenue

# Synchronize workspace catalog tables and semantic column descriptions to Lakeset
angler lakeset sync --catalog main --force

# Mint an RLS-scoped JWT Guest Token for embedded Superset dashboards/charts
angler lakeset guest-token --dataset main.gold.revenue_summary --raw

# Run a live DuckDB S3 Chart Builder aggregation query from the terminal
angler lakeset explore --dataset main.gold.revenue_summary --dimension region --metric amount --agg SUM

# Open or print a zero-login deep-link into Lakeset BI Studio
angler lakeset open --dataset main.gold.revenue_summary --tab explore
```

### 11. Secret Scopes & Enterprise Vault Providers (`angler secrets`, `angler admin vaults`)

Manage workspace secret scopes, encrypted secret keys, role-based ACLs, and external vault backends (PostgreSQL Encrypted Vault, HashiCorp Vault, Google Cloud Secret Manager):

```bash
# List and create workspace secret scopes (alias: angler secret)
angler secrets scopes list
angler secrets scopes create prod-vault --backend hashicorp_vault

# Store, list, and retrieve secrets (masked by default in table output; --raw for shell piping)
angler secrets put prod-vault api_key --string-value "sk-live-..."
angler secrets list prod-vault
angler secrets get prod-vault api_key
export API_KEY=$(angler secrets get prod-vault api_key --raw)

# Manage fine-grained Secret Scope ACLs (READ, WRITE, MANAGE)
angler secrets acls list prod-vault
angler secrets acls set prod-vault --principal write --principal-type role --permission WRITE

# Configure and test cluster-wide vault providers (superadmin)
angler admin vaults list
angler admin vaults test hashicorp_vault
angler admin vaults scopes
```

### 12. BI Dashboards & Visualizations (`angler dashboards`)

Inspect dashboard layouts, deploy new canvas specifications, and extract live aggregated chart data:

```bash
# List all dashboards in the workspace (aliases: angler dashboard, angler dash)
angler dashboards list

# Inspect dashboard canvas layout, metadata, and widget configurations
angler dashboards get dash-123

# Create a dashboard from built-in template or JSON/YAML specification
angler dashboards create --title "Executive KPI Overview" --template executive
angler dashboards create --spec ./dashboards/sales.json

# Display workspace dashboard metrics and BI engine status
angler dashboards stats

# List widgets on a dashboard canvas
angler dashboards widgets list dash-123

# Fetch live aggregated metric points from a widget query
angler dashboards widgets get-data dash-123 w_kpi_1 --time-range last_30_days

# Delete a dashboard
angler dashboards delete dash-123 --force
```

### 13. IAM, Workspaces, Compute & Administration (`angler users`, `angler workspaces`, `angler compute`, `angler admin`)

Manage user accounts, multi-tenant workspace environments, access memberships, Kubernetes platform services, enterprise AI providers, and cluster-wide runtime settings:

```bash
# User Management (superadmin)
angler users list
angler users get u-123
angler users update u-123 --role superadmin --name "Lead Engineer"

# Multi-Tenant Workspace Provisioning & Members
angler workspaces list
angler workspaces get ws-prod
angler workspaces create --name "Risk Analytics" --slug risk-analytics
angler workspaces members list ws-prod
angler workspaces members add ws-prod --email analyst@lot.io --role editor
angler workspaces members remove ws-prod u-123 --force

# Kubernetes Platform Service Lifecycle Controls (superadmin)
angler compute services
angler compute restart superset
angler compute stop jupyter
angler compute start jupyter
# Or via admin alias:
angler admin services list
angler admin services restart airflow

# Multi-Provider AI Administration (Gemini, Vertex AI, OpenAI, Anthropic, Ollama, OpenAI-Compat)
angler admin ai-providers list
angler admin ai-providers test vertex
angler admin ai-providers discover-models vertex

# System Administration & Session Auditing
angler admin settings get
angler admin settings update --session-timeout 120 --gemini-model gemini-2.0-flash
angler admin sessions list
angler admin sessions revoke sess-123 --force
```

### 14. Model Context Protocol (MCP) Server & Remote Bridge (`angler mcp`)

Run Lake of Tears as a local stdio MCP server, bridge stdio directly to the remote `/api/mcp` HTTP endpoint, inspect RBAC-scoped catalog resources, or configure MCP clients (Claude Desktop, Google Antigravity, Cursor):

```bash
# Inspect remote /api/mcp server status, tools, and RBAC catalog resources
angler mcp status
angler mcp tools
angler mcp resources
angler mcp read-resource lot://catalogs/default/main

# Start local MCP server over standard I/O (stdio) or bridge to remote /api/mcp
angler mcp serve
angler mcp serve --remote

# Auto-configure Claude Desktop, Google Antigravity, or Cursor (stdio, sse, or http)
angler mcp install --client claude
angler mcp install --client antigravity --transport http
angler mcp install --client cursor --transport sse --print
```

**Exposed MCP Tools & Resources:**
- **Canonical Lake of Tears Tools:** `list_workspaces_and_catalogs`, `describe_table`, `semantic_search_datalake`, `query_datalake_sql`, `ask_genie`
- **Angler Extended Tools:** `angler_sql_query`, `angler_catalog_inspect`, `angler_fs_browse`, `angler_workspace_read_notebook`, `angler_jobs_trigger`, `angler_ai_genie`, `angler_lakeset_datasets`, `angler_lakeset_explore`, `angler_secrets_list`
- **Catalog Resources:** `lot://catalogs/{workspace_slug}/{catalog_slug}`

---

## Command Reference

| Command | Lake Alias | Description |
| :--- | :--- | :--- |
| `angler bundle` | `angler lab` | Develop, validate, deploy, and manage Lake Asset Bundles (LABs) |
| `angler lakeset` | `angler bi` | Zero-login Lakeset BI Studio, DuckDB semantic sync, guest tokens, and Chart Builder |
| `angler secrets` | `angler secret` | Workspace secret scopes, encrypted key CRUD, and fine-grained ACLs |
| `angler webhooks` | `angler webhook` | Workspace webhook secrets and HMAC-signed event ingestion with pipeline triggers |
| `angler status` | `angler ping` | Check cluster connectivity and service health |
| `angler sql` | `angler query`, `angler cast` | Unified SQL execution, query plans, history, and AI assistance |
| `angler fs` | `angler storage` | Filesystem & storage management modeled after Databricks FS / Volumes |
| `angler workspace` | `angler ws` | Manage notebooks, scripts, directories, and headless execution |
| `angler jobs` | `angler pipelines` | Manage data pipelines, webhooks, schedules, timeline, DAG runs, and task logs |
| `angler dashboards` | `angler dash`, `angler dashboard` | Native BI dashboards, canvas widget configurations, and aggregated metrics |
| `angler ai` | `angler genie` | Genie conversational AI, vector retrieval, virtual models, embeddings, and enrichment |
| `angler ask` | `angler sonar` | Natural language question answering via AI RAG |
| `angler catalog` | `angler net` | Browse and manage catalogs, schemas, tables, soft-delete trash/undrop, TOON export, and sharing |
| `angler compute` | `angler clusters` | Inspect cluster status, topology, and control Kubernetes platform service lifecycles (`restart`, `stop`, `start`) |
| `angler tool-defs` | `angler schema` | Export LLM tool/function schemas (Gemini, OpenAI, OpenAPI 3.1) |
| `angler mcp` | — | Model Context Protocol (MCP) stdio server, remote `/api/mcp` bridge, and client installer |
| `angler up` | `angler launch` | Spin up local Lake of Tears Docker stack |
| `angler down` | `angler dock` | Stop and safely dock running containers |
| `angler auth` | — | Log in, log out, and check authentication status |
| `angler tokens` | `angler pat` | Provision, inspect, list, and revoke Personal Access Tokens |
| `angler users` | `angler user` | User accounts, global roles, active status, and credentials |
| `angler workspaces` | `angler ws-admin` | Multi-tenant workspace provisioning and IAM memberships |
| `angler admin` | — | System settings, multi-provider AI config, external vaults, and session auditing |
| `angler context` | — | Manage multiple environment profiles (`~/.angler/config.yaml`) |
| `angler version` | — | Display build version, commit, and date |
| `angler completion` | — | Generate shell autocompletions (bash, zsh, fish, powershell) |

---

## Lake Asset Bundles (LABs)

`angler bundle` brings declarative Infrastructure-as-Code to your data lakehouse with AI-first Token-Oriented Object Notation (`lake.toon`) and YAML compatibility, supporting Python virtualenv task `requirements`, `@webhook`-triggered pipelines, live drift detection, and AI Prompt-to-Bundle generation.

```bash
# Initialize a new project with starter templates (python-etl, webhook-etl, rag-search, medallion)
angler bundle init my-analytics --template medallion
angler bundle init my-webhook-etl --template webhook-etl --format yaml

# Generate a complete Lake Asset Bundle from a natural language prompt
angler bundle generate "Build a customer 360 medallion pipeline with webhook ingestion" --out lake.toon
angler bundle generate "Biomedical PubMed RAG lakehouse with vector search" --format yaml --out lake.yml

# Validate manifest syntax, DAG cycles, task requirements, and @webhook schedules offline or remotely
angler bundle validate --local
angler bundle validate --target dev

# Simulate deployment with diff reconciliation plan
angler bundle deploy --target prod --dry-run

# Deploy bundle assets, Python scripts (scripts/, python/, src/), SQL, tables, and Airflow DAGs
angler bundle deploy --target dev

# Inspect deployed bundle metadata, resource inventory, and live state drift report
angler bundle get my-analytics --target dev
angler bundle inspect my-analytics --manifest

# Trigger ad-hoc pipeline execution with runtime JSON configuration and wait for completion
angler bundle run orders_etl --target dev --conf '{"batch_date":"2026-11-10"}' --wait

# List active deployed bundles
angler bundle ls

# Teardown and unprovision bundle resources
angler bundle destroy --target dev --purge-storage -y
```

---

## Personal Access Tokens (`angler tokens`)

Manage scoped, long-lived Personal Access Tokens (PATs) for CI/CD runners, automated workflows, and autonomous agents:

```bash
# Generate a new token with default 30-day lifetime in the active workspace
angler tokens create --name "github-actions-deploy"

# Generate a 90-day token with write role for a specific workspace
angler tokens create --name "airflow-sync" --workspace default --role write --lifetime-days 90

# Machine-readable token generation (returns JSON payload containing raw lot_pat_... secret)
angler tokens create --name "agent-token" --json

# List active tokens registered to your account
angler tokens list

# List all tokens including revoked and expired
angler tokens list --all

# Revoke a token immediately
angler tokens revoke 99999999-8888-7777-6666-555555555555 --yes
```

---

## AI Agent & Machine-to-Machine Interface

`angler` is built as a first-class interface for AI agents (Gemini, Claude, OpenAI, LangChain, CrewAI, Antigravity) and CI/CD automation.

### 1. Global Structured Output (`json`, `yaml`, `toon`, `table`)

Pass `--json` or `--output` (`-o`) to any command:

```bash
# JSON output
angler catalog ls --json

# Token-Oriented Object Notation (TOON) for token-efficient LLM consumption
angler query "SELECT * FROM lake.telemetry LIMIT 5" --output toon

# YAML output
angler status --output yaml
```

When `--json` / `--output json|yaml|toon` is specified **or when `stdout` is not a TTY** (e.g., invoked by an AI agent subprocess or piped to `jq`):
- Interactive banners, spinners, and ANSI escape sequences are suppressed.
- Clean, valid structured data is written to `stdout`.
- Informational logs, warnings, and structured error payloads are routed to `stderr`.

### 2. LLM Tool Calling Schema Export (`angler tool-defs`)

Dynamically introspect `angler`'s command tree to generate ready-to-use tool definitions:

```bash
# Google Gemini function declarations
angler tool-defs --format gemini

# OpenAI / Anthropic tool definitions
angler tool-defs --format openai

# OpenAPI 3.1.0 specification
angler schema --format openapi

# Include global flags (--context, --endpoint, --token, etc.) in tool schemas
angler tool-defs --format openai --include-global-flags
```

### 3. Non-Interactive Execution & Exit Codes

Bypass all terminal prompts with `--non-interactive`, `-y` (`--yes`), or `--force`:

```bash
angler auth login --non-interactive --email agent@example.com --password "$ANGLER_PASSWORD"
```

| Exit Code | Error Type | Meaning |
| :--- | :--- | :--- |
| `0` | `ok` | Command succeeded |
| `1` | `runtime_error` | General or unexpected runtime error |
| `2` | `usage_error` | Invalid syntax, flags, or missing required arguments |
| `3` | `auth_error` | Authentication or permission failure (`HTTP 401/403`) |
| `4` | `not_found` | Target resource or context profile not found (`HTTP 404`) |
| `5` | `connection_error` | Lake of Tears endpoint unreachable |

---

## Configuration & Environments

`angler` stores profile configurations in `~/.angler/config.yaml`:

```yaml
current_context: local
contexts:
  local:
    endpoint: http://localhost:8000
    timeout: 30
  production:
    endpoint: https://lake.myorg.internal
    token: eyJhbGciOi...
    workspace: analytics
    timeout: 60
```

### Multi-Tenant Workspace Scoping

Scope all queries, catalog browsing, and commands to a specific tenant workspace:

```bash
# Global persistent flag
angler catalog list --workspace analytics
angler query "SELECT * FROM gold.users" -w marketing

# Or configure workspace directly on a context profile
angler context set production --endpoint https://lake.myorg.internal --workspace analytics
angler context use production
```

### Environment Variables

For headless CI/CD environments, `angler` respects environment variables:

| Variable | Description |
| :--- | :--- |
| `ANGLER_ENDPOINT` | Override the active API endpoint |
| `ANGLER_TOKEN` | API bearer token or service account key |
| `ANGLER_WORKSPACE` / `LOT_WORKSPACE` | Target workspace slug, name, or UUID |
| `ANGLER_WORKSPACE_ID` / `LOT_WORKSPACE_ID` | Explicit target workspace ID |
| `ANGLER_ON_BEHALF_OF` / `LOT_ON_BEHALF_OF` | Trusted proxy user email for superadmin impersonation |
| `ANGLER_CONTEXT` | Select active profile context name |
| `ANGLER_EMAIL` | Account email for non-interactive `angler auth login` |
| `ANGLER_PASSWORD` | Account password for non-interactive `angler auth login` |

---

## Shell Autocompletion & Documentation Generator

Generate interactive tab completions and complete CLI markdown reference documentation:

```bash
# Generate markdown documentation tree and shell completion scripts
angler docs ./docs/reference --completions-dir ./completions

# Zsh
angler completion zsh > "${fpath[1]}/_angler"

# Bash
angler completion bash > /etc/bash_completion.d/angler

# Fish
angler completion fish > ~/.config/fish/completions/angler.fish

# PowerShell
angler completion powershell | Out-String | Invoke-Expression
```

---

## End-to-End Integration Testing

Run the automated E2E test suite against an in-memory mock backend:

```bash
go test -v -race ./test/e2e/...
```

---

## License

This project is licensed under the [MIT License](LICENSE).
