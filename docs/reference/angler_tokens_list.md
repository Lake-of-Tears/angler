## angler tokens list

List active Personal Access Tokens registered to your account

### Synopsis

List all Personal Access Tokens registered to the authenticated user.

Displays token ID, name, prefix, workspace associations, expiration, and last used timestamp.

Examples:
  angler tokens list
  angler tokens list --all
  angler tokens list --json

```
angler tokens list [flags]
```

### Options

```
  -a, --all    include revoked and expired tokens
  -h, --help   help for list
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

* [angler tokens](angler_tokens.md)	 - Manage Personal Access Tokens (PATs) for Lake of Tears

