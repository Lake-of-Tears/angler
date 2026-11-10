## angler sql explain

Explain query execution plan and optimization steps

### Synopsis

Analyze and explain a DuckDB SQL statement step-by-step using Lake of Tears AI engine.

```
angler sql explain [SQL] [flags]
```

### Options

```
  -e, --execute string   SQL query statement to explain
  -f, --file string      Path to file containing SQL query
  -h, --help             help for explain
      --model string     AI model override for plan explanation
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

