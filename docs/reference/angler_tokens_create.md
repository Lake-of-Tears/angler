## angler tokens create

Generate a new Personal Access Token

### Synopsis

Generate a new Personal Access Token (PAT) with scoped workspace permissions and lifetime.

The raw secret token (lot_pat_...) is displayed once upon creation and cannot be retrieved later.
Tokens are cryptographically hashed on the Lake of Tears server.

Examples:
  # Generate token with default 30-day lifetime in active workspace
  angler tokens create --name "ci-deployer"

  # Generate 90-day token with write role for a specific workspace
  angler tokens create --name "airflow-sync" --lifetime-days 90 --workspace default --role write

  # Machine-readable output returning token payload in JSON
  angler tokens create --name "agent-token" --json

```
angler tokens create [flags]
```

### Options

```
      --comment string      alias for --name
  -h, --help                help for create
  -d, --lifetime-days int   token lifespan in days (1 to 3650) (default 30)
  -n, --name string         name or description of the token
  -r, --role string         workspace access role ('read', 'write', 'admin') (default "read")
  -s, --scopes strings      optional access scopes
  -w, --workspace string    workspace name, slug, or ID (defaults to active/default workspace)
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

* [angler tokens](angler_tokens.md)	 - Manage Personal Access Tokens (PATs) for Lake of Tears

