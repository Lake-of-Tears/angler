## angler sql

Unified SQL statement execution, history, and AI assistance (aliases: query, cast)

### Synopsis

Execute SQL statements against DuckDB and Lake of Tears lakehouse storage,
explain query execution plans, generate SQL from natural language prompts,
and inspect or repair failing queries.

Aliases:
  angler sql      - Databricks / ANSI SQL command suite
  angler query    - Standard analytical query command
  angler cast     - Nautical Lake of Tears query command

Examples:
  angler sql "SELECT * FROM lake.telemetry LIMIT 10"
  angler sql exec -f queries/report.sql --format csv
  angler sql explain "SELECT count(*) FROM users"
  angler sql generate "Find top 10 users by spend last month" --execute
  angler sql fix "SELECT empid FROM emps" --error "column empid not found"
  angler sql history --limit 10

```
angler sql [SQL] [flags]
```

### Options

```
      --catalog string     Catalog scope (default "main")
  -e, --execute string     SQL query statement to execute
      --explain            Explain execution plan prior to running
  -f, --file string        Path to file containing SQL query
      --format string      Output format: table, json, yaml, toon, csv, ndjson (default "table")
  -h, --help               help for sql
      --limit int          Maximum number of rows to return (default 1000)
      --params string      JSON parameters array or object
      --schema string      Schema scope (default "default")
      --warehouse string   Execution warehouse ID (default "default")
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --force                 bypass confirmation prompts
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler sql cancel](angler_sql_cancel.md)	 - Cancel an in-flight statement execution
* [angler sql exec](angler_sql_exec.md)	 - Execute a SQL query or DDL/DML statement
* [angler sql explain](angler_sql_explain.md)	 - Explain query execution plan and optimization steps
* [angler sql fix](angler_sql_fix.md)	 - Diagnose and repair a failing SQL query
* [angler sql generate](angler_sql_generate.md)	 - Generate executable SQL from a natural language prompt
* [angler sql get](angler_sql_get.md)	 - Get status and details of a statement execution
* [angler sql history](angler_sql_history.md)	 - List recent SQL statement executions

