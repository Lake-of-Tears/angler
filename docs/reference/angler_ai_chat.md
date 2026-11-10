## angler ai chat

Chat with Lake of Tears Genie AI grounded in catalog tables and metadata

### Synopsis

Executes a single question or initiates an interactive chat session with Genie AI.
Grounded in workspace catalogs, Parquet schemas, and SQL AI functions.

```
angler ai chat [prompt] [flags]
```

### Options

```
  -h, --help           help for chat
  -m, --mode string    Query mode: auto, sql, rag, clarify (default "auto")
      --model string   LLM model override (e.g. gemini-2.5-flash)
      --no-stream      Disable streaming and return complete response
      --stream         Stream response steps in real time
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

