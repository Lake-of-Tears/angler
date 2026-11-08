## angler users update

Modify user role, active status, or display name

```
angler users update <user_id> [flags]
```

### Options

```
      --active        Account active status (default true)
  -h, --help          help for update
      --name string   User display name
      --role string   Role: 'superadmin' or 'user'
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

* [angler users](angler_users.md)	 - Lake of Tears User and Identity Management (IAM)

