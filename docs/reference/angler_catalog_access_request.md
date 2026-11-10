## angler catalog access request

Request access to an external catalog

```
angler catalog access request <catalog_id> [flags]
```

### Options

```
  -h, --help               help for request
  -m, --message string     Reason message for requesting catalog access (default "Requested via Angler CLI")
      --mode string        Requested access mode (read or write) (default "read")
  -w, --workspace string   Requesting workspace ID
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
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler catalog access](angler_catalog_access.md)	 - Manage user-scoped and cross-workspace catalog access requests and shares

