## angler fs cp

Copy files between local filesystem and remote Lake of Tears storage

### Synopsis

Copy files between local paths and remote storage. Supports schemes like lot://, s3://, and dbfs://.

Examples:
  angler fs cp ./local.parquet lot://bronze/local.parquet
  angler fs cp lot://silver/data.parquet ./downloaded.parquet
  angler fs cp s3://bronze/raw.json lot://silver/raw.json

```
angler fs cp <source> <destination> [flags]
```

### Options

```
  -f, --force   Overwrite existing destination files without prompt
  -h, --help    help for cp
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
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

* [angler fs](angler_fs.md)	 - Filesystem and storage management modeled after Databricks FS (alias: storage)

