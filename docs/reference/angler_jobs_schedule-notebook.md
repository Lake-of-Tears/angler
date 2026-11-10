## angler jobs schedule-notebook

Create or update a scheduled notebook pipeline DAG

```
angler jobs schedule-notebook <notebook_path> [flags]
```

### Options

```
      --cpu string        CPU resource limit (e.g. '2')
  -h, --help              help for schedule-notebook
      --memory string     Memory resource limit (e.g. '4Gi')
      --params string     JSON parameters dictionary for notebook execution
      --schedule string   Cron expression or preset schedule (required)
      --title string      Display title for the scheduled notebook pipeline
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

