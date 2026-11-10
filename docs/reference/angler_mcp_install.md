## angler mcp install

Configure MCP clients (Claude Desktop, Antigravity, Cursor) for stdio, SSE, or Streamable HTTP

```
angler mcp install [flags]
```

### Options

```
      --client string        Target client: 'claude', 'antigravity', or 'cursor' (default "claude")
      --config-path string   Explicit path to client configuration JSON file
  -h, --help                 help for install
      --print                Print generated MCP configuration JSON to stdout without writing to disk
      --remote               Use 'angler mcp serve --remote' when transport is stdio
      --transport string     MCP transport mode: 'stdio', 'sse', or 'http' (default "stdio")
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

* [angler mcp](angler_mcp.md)	 - Model Context Protocol (MCP) Server, Remote Bridge, and Client Integration

