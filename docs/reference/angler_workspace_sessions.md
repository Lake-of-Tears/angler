## angler workspace sessions

Inspect and control active Jupyter kernel execution sessions

### Options

```
  -h, --help   help for sessions
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

* [angler workspace](angler_workspace.md)	 - Workspace notebook and directory management modeled after Databricks Workspace
* [angler workspace sessions interrupt](angler_workspace_sessions_interrupt.md)	 - Interrupt a running kernel execution
* [angler workspace sessions restart](angler_workspace_sessions_restart.md)	 - Restart a running kernel instance
* [angler workspace sessions start](angler_workspace_sessions_start.md)	 - Start or attach to a kernel session for a notebook

