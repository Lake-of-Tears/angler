## angler workspaces members add

Assign a user to a workspace with a specific role

```
angler workspaces members add <workspace_id> [flags]
```

### Options

```
      --email string   User email address to assign
  -h, --help           help for add
      --role string    Workspace role: admin, editor, or viewer (default "viewer")
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

* [angler workspaces members](angler_workspaces_members.md)	 - Manage workspace user memberships and role assignments

