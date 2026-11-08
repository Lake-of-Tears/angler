## angler jobs runs

List, inspect, and manage historical pipeline runs

### Options

```
  -h, --help   help for runs
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

* [angler jobs](angler_jobs.md)	 - Data pipelines, scheduled jobs, and workflow DAGs management (alias: pipelines)
* [angler jobs runs cancel](angler_jobs_runs_cancel.md)	 - Cancel an in-flight pipeline run
* [angler jobs runs get](angler_jobs_runs_get.md)	 - Inspect granular execution details and task breakdown of a run
* [angler jobs runs list](angler_jobs_runs_list.md)	 - List past execution runs for a pipeline
* [angler jobs runs logs](angler_jobs_runs_logs.md)	 - Stream or fetch stdout/stderr task execution logs

