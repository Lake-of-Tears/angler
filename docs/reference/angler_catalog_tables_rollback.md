## angler catalog tables rollback

Roll back an Apache Iceberg table to Parquet (zero-copy or materialized)

### Synopsis

Roll back an Apache Iceberg table to Parquet format.
If the table is untouched since zero-copy migration, restores the original Parquet format zero-copy.
If the table has new commits, use --materialize to export the current snapshot to clean Parquet files.

Examples:
  # Dry run inspection
  angler catalog tables rollback main default users --dry-run

  # Zero-copy rollback for untouched table
  angler catalog tables rollback main default users -y

  # Materialized conversion for mutated table
  angler catalog tables rollback main default users --materialize -y

```
angler catalog tables rollback <catalog_id> <schema_id> <table_id> [flags]
```

### Options

```
      --dry-run       Report plan without applying changes
  -h, --help          help for rollback
      --materialize   Export modified Iceberg snapshot to Parquet via DuckDB
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

* [angler catalog tables](angler_catalog_tables.md)	 - Manage and inspect catalog tables

