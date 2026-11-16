## angler catalog migrate

Batch migrate an entire catalog's tables between Parquet and Iceberg

### Synopsis

Batch migrates all Parquet tables in a catalog in-place zero-copy to Apache Iceberg.
Alternatively specify --to-parquet to roll back or convert all Iceberg tables to Parquet.

Examples:
  # Dry-run inspection of catalog-wide migration
  angler catalog migrate main --dry-run

  # Batch migrate all tables in catalog to Iceberg
  angler catalog migrate main -y

  # Batch rollback catalog to Parquet
  angler catalog migrate main --to-parquet -y

```
angler catalog migrate <catalog_id> [flags]
```

### Options

```
      --dry-run       Report plan without applying changes
  -h, --help          help for migrate
      --materialize   Export modified Iceberg snapshots to Parquet via DuckDB
      --skip-verify   Skip post-migration row count and schema verification
      --to-iceberg    Batch migrate Parquet tables to Iceberg (default)
      --to-parquet    Batch roll back Iceberg tables to Parquet
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

