## angler jobs runs list

List past execution runs for a pipeline

```
angler jobs runs list <job_id> [flags]
```

### Options

```
  -h, --help        help for list
  -l, --limit int   Maximum number of past runs to retrieve (default 25)
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

* [angler jobs runs](angler_jobs_runs.md)	 - List, inspect, and manage historical pipeline runs

