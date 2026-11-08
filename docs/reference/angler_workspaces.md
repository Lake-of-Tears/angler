## angler workspaces

Lake of Tears Multi-Tenant Workspace Provisioning and IAM Members

### Synopsis

Manage Lake of Tears multi-tenant workspace environments, access memberships,
and tenant boundaries.

Aliases:
  angler workspaces
  angler ws-admin

Examples:
  angler workspaces list
  angler workspaces get ws-prod
  angler workspaces create --name "Risk Analytics" --slug risk-analytics
  angler workspaces members list ws-prod
  angler workspaces members add ws-prod --email analyst@lot.io --role editor
  angler workspaces members remove ws-prod u-123

### Options

```
  -h, --help   help for workspaces
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
* [angler workspaces create](angler_workspaces_create.md)	 - Provision a new analytical workspace
* [angler workspaces get](angler_workspaces_get.md)	 - Inspect workspace details, storage boundaries, and configuration
* [angler workspaces list](angler_workspaces_list.md)	 - List all workspaces accessible to the current context
* [angler workspaces members](angler_workspaces_members.md)	 - Manage workspace user memberships and role assignments

