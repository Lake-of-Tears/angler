## angler mcp read-resource

Read an MCP catalog resource URI (e.g. lot://catalogs/default/main)

```
angler mcp read-resource <uri> [flags]
```

### Options

```
  -h, --help   help for read-resource
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

