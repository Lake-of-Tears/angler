## angler workspace run

Trigger headless execution of a workspace notebook

### Synopsis

Initiates execution of all cells in a workspace notebook.
Attaches to or allocates a Jupyter kernel session, inspects cell outputs, and returns execution status.

```
angler workspace run <workspace-path> [flags]
```

### Options

```
  -h, --help               help for run
  -k, --kernel string      Kernel name for execution (default "python3")
  -s, --save string        Local path to save executed notebook with output cells
      --timeout duration   Execution timeout (default 2m0s)
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

* [angler workspace](angler_workspace.md)	 - Workspace notebook and directory management modeled after Databricks Workspace

