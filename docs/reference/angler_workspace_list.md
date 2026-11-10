## angler workspace list

List items (notebooks, folders, source files) within the workspace directory tree

### Synopsis

Lists notebooks, subdirectories, and source files within the workspace directory tree. Supports recursive traversal and JSON output.

```
angler workspace list [path] [flags]
```

### Options

```
      --format string   Output format: table, json (default "table")
  -h, --help            help for list
  -r, --recursive       Recursively list directory items
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

