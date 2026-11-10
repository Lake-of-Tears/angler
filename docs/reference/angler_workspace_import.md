## angler workspace import

Upload and register a notebook or script into the workspace

### Synopsis

Uploads and registers a local Jupyter (.ipynb), Python (.py), SQL (.sql), or Markdown (.md)
file into the Lake of Tears workspace directory.

Supports format conversions:
  --format JUPYTER : expects standard Jupyter notebook JSON (.ipynb).
  --format SOURCE  : parses scripts into cells with '# %%' cell boundaries or single code cells.
  --format AUTO    : automatically detects format based on file extension.

```
angler workspace import <local-file> <workspace-path> [flags]
```

### Options

```
      --format string   Format: AUTO, JUPYTER, SOURCE (default "AUTO")
  -h, --help            help for import
      --overwrite       Overwrite existing workspace item
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

