## angler sql exec

Execute a SQL query or DDL/DML statement

### Synopsis

Execute analytical queries (SELECT) or DDL/DML statements (CREATE TABLE, INSERT, ALTER)
against the Lake of Tears execution engine.

Supports reading from command arguments, flags (-e), or files (-f).

```
angler sql exec [SQL] [flags]
```

### Options

```
      --catalog string     Catalog scope (default "main")
  -e, --execute string     SQL query statement to execute
      --explain            Explain execution plan prior to running
  -f, --file string        Path to file containing SQL query
      --format string      Output format: table, json, yaml, toon, csv, ndjson (default "table")
  -h, --help               help for exec
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

* [angler sql](angler_sql.md)	 - Unified SQL statement execution, history, and AI assistance (aliases: query, cast)

