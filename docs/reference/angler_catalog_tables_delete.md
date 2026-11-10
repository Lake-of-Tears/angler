## angler catalog tables delete

Drop a table from a catalog schema

```
angler catalog tables delete <catalog_id> <schema_id> <table_id> [flags]
```

### Options

```
  -f, --force   Force delete without confirmation
  -h, --help    help for delete
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

* [angler catalog tables](angler_catalog_tables.md)	 - Manage and inspect catalog tables

