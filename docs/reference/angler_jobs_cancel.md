## angler jobs cancel

Cancel an in-flight pipeline run

### Synopsis

Terminates or marks failed an active or queued DAG execution run.

```
angler jobs cancel <job_id> <run_id> [flags]
```

### Options

```
  -f, --force   Force cancel without confirmation prompt
  -h, --help    help for cancel
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
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

