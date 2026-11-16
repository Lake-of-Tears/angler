## angler catalog rollback

Batch roll back an entire catalog's Iceberg tables to Parquet

### Synopsis

Batch rolls back all Apache Iceberg tables in a catalog to Parquet format.
Tables untouched since migration are reverted zero-copy; mutated tables can be converted with --materialize.

Examples:
  # Dry run
  angler catalog rollback main --dry-run

  # Batch rollback
  angler catalog rollback main -y

```
angler catalog rollback <catalog_id> [flags]
```

### Options

```
      --dry-run       Report plan without applying changes
  -h, --help          help for rollback
      --materialize   Export modified Iceberg snapshots to Parquet via DuckDB
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
      --toon                  output results in TOON format
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
      --yaml                  output results in YAML format
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler catalog](angler_catalog.md)	 - Unity Catalog-style metadata discovery and management (alias: net)

