## angler sql generate

Generate executable SQL from a natural language prompt

### Synopsis

Translate a natural language question into executable DuckDB SQL grounded in your catalog schemas.
Optionally execute the generated query immediately with -x/--execute.

```
angler sql generate <prompt> [flags]
```

### Options

```
      --current-sql string   Current SQL context to refine
  -h, --help                 help for generate
      --model string         AI model override
  -x, --run                  Execute the generated SQL statement automatically
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

