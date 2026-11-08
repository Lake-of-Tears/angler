## angler fs stat

Retrieve file metadata and Parquet schema inspection

### Synopsis

Inspect an S3 object key to view file size, headers, ETag, last modified date, and Parquet column metrics.

```
angler fs stat <path> [flags]
```

### Options

```
  -h, --help   help for stat
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

