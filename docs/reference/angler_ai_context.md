## angler ai context

Inspect active system data context, grounding prompt, and registered UDFs

### Synopsis

Fetches schemas, datasets, and SQL AI function definitions exposed to the AI engine.

```
angler ai context [flags]
```

### Options

```
  -h, --help   help for context
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

* [angler ai](angler_ai.md)	 - Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)

