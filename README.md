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
brew tap Lake-of-Tears/tap
brew trust lake-of-tears/tap
brew install angler
```

Or in a single command:

```bash
brew install Lake-of-Tears/tap/angler
```

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

Browse and manage Unity Catalog-style data catalogs, schemas, tables, and anomalies:

```bash
# List all registered catalogs (alias: angler net)
angler catalog ls

# Inspect catalog details and schemas
angler catalog get main

# Manage database schemas within a catalog
angler catalog schemas list main
angler catalog schemas get main default
angler catalog schemas create main gold

# Inspect tables, columns, and partition patterns
angler catalog tables list main default
angler catalog tables get main default users

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
```

### 9. Genie AI & Semantic Gateway (`angler ai`)

Converse with Lake of Tears Genie AI grounded in catalog tables and schemas, stream execution steps, inspect active semantic grounding contexts, and manage table vector indexes:

```bash
# Natural language data queries and semantic reasoning (alias: angler genie)
angler ai chat "Which tables have customer transaction data?"
angler ai chat "Count total orders in 2026" --no-stream

# Stream reasoning steps and SQL generation in real-time
angler ai chat --stream "Analyze revenue variance between Q2 and Q3"

# Inspect active workspace grounding context, schemas, and registered SQL AI functions
angler ai context

# List active table vector embeddings and indexes across workspace catalogs
angler ai vector-index list
angler ai vector-index search "high value customers"

# Trigger vector index synchronization for a table
angler ai vector-index sync main retail orders

# Verify configured LLM credentials against the backend gateway
angler ai test-key --model gemini-2.0-flash
```

### 10. BI Dashboards & Visualizations (`angler dashboards`)

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

### 11. IAM, Workspaces & Administration (`angler users`, `angler workspaces`, `angler admin`)

Manage user accounts, multi-tenant workspace environments, access memberships, and cluster-wide runtime settings:

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

# System Administration & Session Auditing
angler admin settings get
angler admin settings update --session-timeout 120 --gemini-model gemini-2.0-flash
angler admin sessions list
angler admin sessions revoke sess-123 --force
```

---

## Command Reference

| Command | Lake Alias | Description |
| :--- | :--- | :--- |
| `angler bundle` | `angler lab` | Develop, validate, deploy, and manage Lake Asset Bundles (LABs) |
| `angler status` | `angler ping` | Check cluster connectivity and service health |
| `angler sql` | `angler query`, `angler cast` | Unified SQL execution, query plans, history, and AI assistance |
| `angler fs` | `angler storage` | Filesystem & storage management modeled after Databricks FS / Volumes |
| `angler workspace` | `angler ws` | Manage notebooks, scripts, directories, and headless execution |
| `angler jobs` | `angler pipelines` | Manage data pipelines, scheduled jobs, DAG runs, and task logs |
| `angler dashboards` | `angler dash`, `angler dashboard` | Native BI dashboards, canvas widget configurations, and aggregated metrics |
| `angler ai` | `angler genie` | Genie conversational AI, semantic gateway, vector indexes, and LLM verification |
| `angler ask` | `angler sonar` | Natural language question answering via AI RAG |
| `angler catalog` | `angler net` | Browse Parquet tables, schemas, and partition info |
| `angler tool-defs` | `angler schema` | Export LLM tool/function schemas (Gemini, OpenAI, OpenAPI 3.1) |
| `angler up` | `angler launch` | Spin up local Lake of Tears Docker stack |
| `angler down` | `angler dock` | Stop and safely dock running containers |
| `angler auth` | — | Log in, log out, and check authentication status |
| `angler tokens` | `angler pat` | Provision, inspect, list, and revoke Personal Access Tokens |
| `angler users` | `angler user` | User accounts, global roles, active status, and credentials |
| `angler workspaces` | `angler ws-admin` | Multi-tenant workspace provisioning and IAM memberships |
| `angler admin` | — | System-wide settings, session auditing, and security controls |
| `angler context` | — | Manage multiple environment profiles (`~/.angler/config.yaml`) |
| `angler version` | — | Display build version, commit, and date |
| `angler completion` | — | Generate shell autocompletions (bash, zsh, fish, powershell) |

---

## Lake Asset Bundles (LABs)

`angler bundle` brings declarative Infrastructure-as-Code to your data lakehouse with AI-first Token-Oriented Object Notation (`lake.toon`) and YAML compatibility.

```bash
# Initialize a new project with starter templates
angler bundle init my-analytics --template medallion

# Validate manifest syntax offline or against remote server
angler bundle validate --local
angler bundle validate --target dev

# Simulate deployment with diff reconciliation plan
angler bundle deploy --target prod --dry-run

# Deploy bundle assets, managed tables, and Airflow DAGs
angler bundle deploy --target dev

# Trigger ad-hoc pipeline execution
angler bundle run orders_etl --target dev

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
| `ANGLER_CONTEXT` | Select active profile context name |
| `ANGLER_EMAIL` | Account email for non-interactive `angler auth login` |
| `ANGLER_PASSWORD` | Account password for non-interactive `angler auth login` |

---

## Shell Autocompletion

Enable interactive tab completion in your shell:

```bash
# Zsh
angler completion zsh > "${fpath[1]}/_angler"

# Bash
angler completion bash > /etc/bash_completion.d/angler

# Fish
angler completion fish > ~/.config/fish/completions/angler.fish
```

---

## License

This project is licensed under the [MIT License](LICENSE).
