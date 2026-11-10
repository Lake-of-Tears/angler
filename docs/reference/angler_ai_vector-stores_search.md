## angler ai vector-stores search

Search a specific OpenAI-compatible vector store ID (all, <workspace>, or <ws>:<cat>.<sch>.<tbl>)

```
angler ai vector-stores search <vector_store_id> <query> [flags]
```

### Options

```
  -h, --help                    help for search
      --score-threshold float   Minimum cosine similarity score threshold (0.0..1.0)
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

* [angler ai vector-stores](angler_ai_vector-stores.md)	 - List and search RBAC-scoped OpenAI-compatible vector stores

