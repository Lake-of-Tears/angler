## angler jobs run

Immediately trigger a pipeline run

### Synopsis

Triggers on-demand execution of the specified pipeline.
When --wait is passed, blocks until completion, polls status, and displays the task breakdown upon finish.

```
angler jobs run <job_id> [flags]
```

### Options

```
      --conf string              JSON parameters to pass to pipeline run
  -h, --help                     help for run
      --poll-interval duration   Polling interval when waiting (default 3s)
      --timeout duration         Timeout when waiting for completion (default 10m0s)
      --wait                     Wait for run completion before exiting
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

