## angler ai

Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)

### Synopsis

Interact with Lake of Tears' built-in Genie semantic engine, conversational AI,
vector indexes, and LLM gateway.

Enables human engineers and autonomous LLM agents to converse with Genie AI grounded in
lakehouse catalogs and schemas, stream execution steps, inspect active grounding contexts,
and manage vector embeddings.

Aliases:
  angler ai
  angler genie

Examples:
  angler ai chat "Which tables have customer transaction data?"
  angler ai chat --no-stream "Count total orders in 2026"
  angler ai context
  angler ai vector-index list
  angler ai vector-index search "high value customers"
  angler ai test-key --model gemini-2.0-flash

```
angler ai [flags]
```

### Options

```
  -h, --help   help for ai
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler ai chat](angler_ai_chat.md)	 - Chat with Lake of Tears Genie AI grounded in catalog tables and metadata
* [angler ai context](angler_ai_context.md)	 - Inspect active system data context, grounding prompt, and registered UDFs
* [angler ai embeddings](angler_ai_embeddings.md)	 - Generate vector embeddings using the active Lake of Tears AI provider
* [angler ai enrich](angler_ai_enrich.md)	 - Auto-generate semantic business descriptions and column comments for a catalog, schema, or table
* [angler ai models](angler_ai_models.md)	 - List virtual OpenAI-compatible models tailored to caller RBAC permissions
* [angler ai retrieve](angler_ai_retrieve.md)	 - Execute RBAC-enforced vector similarity chunk retrieval without LLM synthesis
* [angler ai table-from-prompt](angler_ai_table-from-prompt.md)	 - Generate a TOON table specification, column definitions, and DuckDB DDL from natural language
* [angler ai test-key](angler_ai_test-key.md)	 - Validate configured LLM provider credentials (e.g. Gemini, OpenAI)
* [angler ai vector-index](angler_ai_vector-index.md)	 - Manage and search table vector indexes and semantic embeddings
* [angler ai vector-stores](angler_ai_vector-stores.md)	 - List and search RBAC-scoped OpenAI-compatible vector stores

