## angler fs ls

List files, directories, and partitions in object storage

### Synopsis

List objects and common prefixes in workspace storage. Supports recursive traversal and JSON output.

```
angler fs ls [path] [flags]
```

### Options

```
  -a, --absolute    Show absolute keys instead of relative filenames
  -h, --help        help for ls
  -l, --limit int   Maximum number of items to list (default 100)
  -r, --recursive   Recursively list directories
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

* [angler fs](angler_fs.md)	 - Filesystem and storage management modeled after Databricks FS (alias: storage)

