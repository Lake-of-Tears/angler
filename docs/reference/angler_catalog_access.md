## angler catalog access

Manage user-scoped and cross-workspace catalog access requests and shares

### Options

```
  -h, --help   help for access
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

* [angler catalog](angler_catalog.md)	 - Unity Catalog-style metadata discovery and management (alias: net)
* [angler catalog access approve](angler_catalog_access_approve.md)	 - Approve an inbound catalog access request
* [angler catalog access list](angler_catalog_access_list.md)	 - List inbound access requests and active user-scoped shares for a catalog
* [angler catalog access reject](angler_catalog_access_reject.md)	 - Reject an inbound catalog access request
* [angler catalog access request](angler_catalog_access_request.md)	 - Request access to an external catalog
* [angler catalog access revoke](angler_catalog_access_revoke.md)	 - Revoke a user-scoped catalog share

