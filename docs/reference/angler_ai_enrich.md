## angler ai enrich

Auto-generate semantic business descriptions and column comments for a catalog, schema, or table

```
angler ai enrich [name] [flags]
```

### Options

```
      --catalog string   Catalog name context
  -h, --help             help for enrich
      --model string     Optional LLM model override
      --schema string    Schema name context
      --table string     Table name (optional if passed as positional argument)
      --type string      Asset type: table, schema, or catalog (default "table")
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

* [angler ai](angler_ai.md)	 - Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)

