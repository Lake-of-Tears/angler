## angler admin sessions

Audit and manage active user and API sessions

### Options

```
  -h, --help   help for sessions
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

* [angler admin](angler_admin.md)	 - Lake of Tears System Administration, Settings, and Session Auditing
* [angler admin sessions list](angler_admin_sessions_list.md)	 - List active user sessions across the cluster
* [angler admin sessions revoke](angler_admin_sessions_revoke.md)	 - Revoke an active user or service session

