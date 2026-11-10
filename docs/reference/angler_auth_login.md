## angler auth login

Log in to your Lake of Tears account via Personal Access Token or credentials

### Synopsis

Log in to Lake of Tears using a Personal Access Token (PAT) or email and password.

Recommended:
  Use a Personal Access Token created in the Lake of Tears Web UI (Account Settings -> Personal Access Tokens).
  This avoids storing raw passwords and supports granular workspace-level access.

Examples:
  # Authenticate with a PAT (interactive prompt or flag)
  angler auth login --token lot_pat_abcdef0123456789...

  # Pipe token via stdin (ideal for CI/CD)
  echo "$LOT_TOKEN" | angler auth login --token-stdin

  # Legacy interactive login
  angler auth login

```
angler auth login [flags]
```

### Options

```
      --email string      account email address (or ANGLER_EMAIL)
  -h, --help              help for login
      --password string   account password (or ANGLER_PASSWORD)
  -t, --token string      authenticate using Personal Access Token (or ANGLER_TOKEN)
      --token-stdin       read Personal Access Token from standard input
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
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler auth](angler_auth.md)	 - Manage authentication with Lake of Tears

