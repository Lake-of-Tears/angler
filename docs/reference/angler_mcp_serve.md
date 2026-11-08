## angler mcp serve

Start native Model Context Protocol (MCP) server over standard I/O (stdio)

### Synopsis

Starts a stdio-based MCP server providing LLM tools for Lake of Tears.
Designed to be invoked by Claude Desktop, Antigravity, Cursor, Continue, or custom agents.

```
angler mcp serve [flags]
```

### Options

```
  -h, --help   help for serve
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

* [angler mcp](angler_mcp.md)	 - Model Context Protocol (MCP) Server and Developer Environment Integration

