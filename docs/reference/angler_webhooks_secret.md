## angler webhooks secret

Retrieve or rotate the active workspace's webhook secret token

```
angler webhooks secret [flags]
```

### Options

```
  -h, --help   help for secret
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

* [angler webhooks](angler_webhooks.md)	 - Workspace Webhook Ingestion, HMAC Signing, Secret Rotation & Downstream Pipeline Triggering
* [angler webhooks secret get](angler_webhooks_secret_get.md)	 - Retrieve the active workspace's webhook secret token and raw ingestion URL
* [angler webhooks secret rotate](angler_webhooks_secret_rotate.md)	 - Regenerate the workspace's webhook secret token

