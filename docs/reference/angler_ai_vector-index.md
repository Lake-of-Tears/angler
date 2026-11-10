## angler ai vector-index

Manage and search table vector indexes and semantic embeddings

### Options

```
  -h, --help   help for vector-index
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
* [angler ai vector-index list](angler_ai_vector-index_list.md)	 - List active table vector embeddings and indexes across workspace catalogs
* [angler ai vector-index search](angler_ai_vector-index_search.md)	 - Execute semantic vector search against registered documentation and tables
* [angler ai vector-index sync](angler_ai_vector-index_sync.md)	 - Trigger incremental or full vector index synchronization for a table

