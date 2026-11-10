## angler webhooks send

Send an event payload to workspace webhook ingestion (MinIO Parquet buffer + optional pipeline trigger)

```
angler webhooks send [flags]
```

### Options

```
  -f, --file string               Path to JSON event payload file
  -h, --help                      help for send
      --hmac                      Sign payload with HMAC-SHA256 (X-Hub-Signature-256)
  -d, --payload string            JSON event payload string
      --secret string             Explicit workspace webhook secret (auto-resolved if omitted)
      --trigger-pipeline string   Downstream pipeline DAG ID to trigger immediately after buffering
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

