## angler mcp

Model Context Protocol (MCP) Server, Remote Bridge, and Client Integration

### Synopsis

Run Lake of Tears as a Model Context Protocol (MCP) server over standard I/O (stdio),
bridge stdio to the remote /api/mcp HTTP endpoint, inspect remote MCP tools and catalog
resources, or configure MCP clients (Claude Desktop, Antigravity, Cursor).

Examples:
  angler mcp status
  angler mcp tools
  angler mcp resources
  angler mcp read-resource lot://catalogs/default/main
  angler mcp serve
  angler mcp serve --remote
  angler mcp install --client claude
  angler mcp install --client antigravity --transport http
  angler mcp install --client cursor --transport sse --print

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
      --on-behalf-of string   impersonate target user email via trusted superadmin service token (X-On-Behalf-Of)
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler mcp install](angler_mcp_install.md)	 - Configure MCP clients (Claude Desktop, Antigravity, Cursor) for stdio, SSE, or Streamable HTTP
* [angler mcp read-resource](angler_mcp_read-resource.md)	 - Read an MCP catalog resource URI (e.g. lot://catalogs/default/main)
* [angler mcp resources](angler_mcp_resources.md)	 - List RBAC-authorized catalog resources (lot://catalogs/*) from /api/mcp
* [angler mcp serve](angler_mcp_serve.md)	 - Start native Model Context Protocol (MCP) server over standard I/O (stdio)
* [angler mcp status](angler_mcp_status.md)	 - Inspect the remote Lake of Tears /api/mcp server status, protocol versions, and endpoints
* [angler mcp tools](angler_mcp_tools.md)	 - List MCP tools exposed by the remote Lake of Tears /api/mcp server

