## angler mcp

Model Context Protocol (MCP) Server and Developer Environment Integration

### Synopsis

Run Lake of Tears as a Model Context Protocol (MCP) server over standard I/O (stdio),
or automatically configure AI developer environments (Claude Desktop, Antigravity, Cursor).

Enables LLM agents and assistant environments to execute analytical DuckDB queries,
inspect lakehouse catalogs and schemas, browse object store volumes, read notebooks,
trigger pipelines, and query Genie AI.

Examples:
  angler mcp serve
  angler mcp install --client claude
  angler mcp install --client antigravity
  angler mcp install --client cursor

### Options

```
  -h, --help   help for mcp
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler mcp install](angler_mcp_install.md)	 - Automatically configure Claude Desktop, Antigravity, or Cursor for Angler MCP
* [angler mcp serve](angler_mcp_serve.md)	 - Start native Model Context Protocol (MCP) server over standard I/O (stdio)

