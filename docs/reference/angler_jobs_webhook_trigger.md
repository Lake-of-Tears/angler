## angler jobs webhook trigger

Trigger a pipeline via its native inbound webhook endpoint

```
angler jobs webhook trigger <job_id> [flags]
```

### Options

```
  -h, --help                  help for trigger
      --hmac                  Sign payload with HMAC-SHA256 (X-Hub-Signature-256)
  -d, --payload string        JSON payload string to send to webhook
  -f, --payload-file string   Path to JSON payload file
      --secret string         Workspace webhook secret token (auto-resolved if omitted)
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

* [angler jobs webhook](angler_jobs_webhook.md)	 - Enable, disable, and trigger native inbound pipeline webhooks

