## angler sql fix

Diagnose and repair a failing SQL query

### Synopsis

Analyze a broken SQL statement alongside its error message and propose an AI-repaired query grounded in catalog schemas.
Optionally execute the repaired query immediately with -x/--execute.

```
angler sql fix [SQL] [flags]
```

### Options

```
  -e, --error string       Database error message from failure
  -f, --file string        Path to file containing SQL statement
  -h, --help               help for fix
      --model string       AI model override
  -x, --run                Execute the repaired SQL statement automatically
  -s, --statement string   Failing SQL statement
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --force                 bypass confirmation prompts
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler sql](angler_sql.md)	 - Unified SQL statement execution, history, and AI assistance (aliases: query, cast)

