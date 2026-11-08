## angler workspace export

Export a workspace notebook to local disk in Jupyter or Python script format

### Synopsis

Exports a workspace notebook to standard Jupyter (.ipynb) or Python script (.py) format.
If destination is '-' or omitted without redirection, writes directly or creates a local file.

```
angler workspace export <workspace-path> [dest] [flags]
```

### Options

```
      --dest string     Destination file path
      --format string   Export format: JUPYTER, SOURCE (default "JUPYTER")
  -h, --help            help for export
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

