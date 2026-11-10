## angler catalog tables update

Update table description or S3 path pattern

```
angler catalog tables update <catalog_id> <schema_id> <table_id> [flags]
```

### Options

```
  -d, --description string   Updated table description
  -h, --help                 help for update
      --s3-path string       Updated S3 path pattern
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

* [angler catalog tables](angler_catalog_tables.md)	 - Manage and inspect catalog tables

