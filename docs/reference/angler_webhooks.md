## angler webhooks

Workspace Webhook Ingestion, HMAC Signing, Secret Rotation & Downstream Pipeline Triggering

### Synopsis

Manage workspace webhook secret tokens and stream real-time JSON event payloads
into MinIO Parquet raw storage with optional atomic downstream pipeline triggering.

Examples:
  angler webhooks secret get
  angler webhooks secret rotate --force
  angler webhooks send --payload '{"event":"order_placed","amount":99.5}'
  angler webhooks send -f events.json --trigger-pipeline bronze_to_silver_etl --hmac

```
angler webhooks [flags]
```

### Options

```
  -h, --help   help for webhooks
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler webhooks secret](angler_webhooks_secret.md)	 - Retrieve or rotate the active workspace's webhook secret token
* [angler webhooks send](angler_webhooks_send.md)	 - Send an event payload to workspace webhook ingestion (MinIO Parquet buffer + optional pipeline trigger)

