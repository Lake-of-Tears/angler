## angler ai vector-stores

List and search RBAC-scoped OpenAI-compatible vector stores

```
angler ai vector-stores [flags]
```

### Options

```
  -h, --help   help for vector-stores
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
* [angler ai vector-stores list](angler_ai_vector-stores_list.md)	 - List caller-accessible workspaces and vector-indexed catalog tables as vector_store objects
* [angler ai vector-stores search](angler_ai_vector-stores_search.md)	 - Search a specific OpenAI-compatible vector store ID (all, <workspace>, or <ws>:<cat>.<sch>.<tbl>)

