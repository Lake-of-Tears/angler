## angler workspace

Workspace notebook and directory management modeled after Databricks Workspace

### Synopsis

Manage Lake of Tears notebooks, code directories, and interactive analytical sessions.

Modeled after Databricks Workspace, enabling human data engineers and LLM agents
to inspect directory trees, import/export Jupyter (.ipynb) and source files (.py, .sql, .md),
create directories, trigger headless executions, and inspect archived runs.

Aliases:
  angler workspace
  angler ws

Examples:
  angler workspace list
  angler workspace list analysis/ --recursive
  angler workspace import ./model_training.py analysis/model_training.ipynb --format SOURCE
  angler workspace export analysis/model_training.ipynb ./model_training.ipynb
  angler workspace delete analysis/legacy_experiment.ipynb --force
  angler workspace mkdirs analysis/v2/experiments
  angler workspace run analysis/model_training.ipynb --kernel python3
  angler workspace search "duckdb"
  angler workspace runs analysis/model_training.ipynb

```
angler workspace [flags]
```

### Options

```
  -h, --help   help for workspace
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler workspace delete](angler_workspace_delete.md)	 - Remove a notebook or workspace directory
* [angler workspace export](angler_workspace_export.md)	 - Export a workspace notebook to local disk in Jupyter or Python script format
* [angler workspace import](angler_workspace_import.md)	 - Upload and register a notebook or script into the workspace
* [angler workspace list](angler_workspace_list.md)	 - List items (notebooks, folders, source files) within the workspace directory tree
* [angler workspace mkdirs](angler_workspace_mkdirs.md)	 - Create directory structures within the workspace
* [angler workspace run](angler_workspace_run.md)	 - Trigger headless execution of a workspace notebook
* [angler workspace runs](angler_workspace_runs.md)	 - List historical archived runs and artifact captures for a notebook
* [angler workspace search](angler_workspace_search.md)	 - Search for notebooks and files across workspace directories
* [angler workspace sessions](angler_workspace_sessions.md)	 - Inspect and control active Jupyter kernel execution sessions

