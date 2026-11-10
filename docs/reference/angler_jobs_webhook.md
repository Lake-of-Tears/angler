## angler jobs webhook

Enable, disable, and trigger native inbound pipeline webhooks

### Options

```
  -h, --help   help for webhook
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

* [angler jobs](angler_jobs.md)	 - Data pipelines, scheduled jobs, and workflow DAGs management (alias: pipelines)
* [angler jobs webhook disable](angler_jobs_webhook_disable.md)	 - Disable native inbound webhook trigger for a pipeline
* [angler jobs webhook enable](angler_jobs_webhook_enable.md)	 - Enable native inbound webhook trigger for a pipeline and print its endpoint URL
* [angler jobs webhook trigger](angler_jobs_webhook_trigger.md)	 - Trigger a pipeline via its native inbound webhook endpoint

