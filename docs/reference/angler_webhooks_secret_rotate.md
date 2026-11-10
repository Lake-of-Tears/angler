## angler webhooks secret rotate

Regenerate the workspace's webhook secret token

```
angler webhooks secret rotate [flags]
```

### Options

```
  -f, --force   Rotate secret without confirmation prompt
  -h, --help    help for rotate
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
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

* [angler webhooks secret](angler_webhooks_secret.md)	 - Retrieve or rotate the active workspace's webhook secret token

