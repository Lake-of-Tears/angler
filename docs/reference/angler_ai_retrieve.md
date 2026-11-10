## angler ai retrieve

Execute RBAC-enforced vector similarity chunk retrieval without LLM synthesis

```
angler ai retrieve <query> [flags]
```

### Options

```
      --catalog string          Filter chunks to a specific catalog
  -h, --help                    help for retrieve
      --schema string           Filter chunks to a specific schema
      --score-threshold float   Minimum cosine similarity score threshold (0.0..1.0)
      --table string            Filter chunks to a specific table
      --top-k int               Maximum chunks to return (1..50) (default 5)
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

