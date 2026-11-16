## angler catalog tables migrate

Migrate a table in-place zero-copy to Apache Iceberg (or Parquet via --to-parquet)

### Synopsis

Converts a Parquet table in-place zero-copy to Apache Iceberg format.
Optionally roll back to Parquet format by specifying --to-parquet.

Examples:
  # Dry-run inspection of Parquet -> Iceberg migration
  angler catalog tables migrate main default users --dry-run

  # Execute migration in-place zero-copy
  angler catalog tables migrate main default users -y

  # Rollback untouched table zero-copy to Parquet
  angler catalog tables migrate main default users --to-parquet -y

  # Materialize modified Iceberg table snapshot to clean Parquet export
  angler catalog tables migrate main default users --to-parquet --materialize -y

```
angler catalog tables migrate <catalog_id> <schema_id> <table_id> [flags]
```

### Options

```
      --dry-run       Report plan without applying changes
  -h, --help          help for migrate
      --materialize   Export modified Iceberg snapshot to Parquet via DuckDB
      --skip-verify   Skip post-migration row count and schema verification
      --to-parquet    Roll back or convert Iceberg table to Parquet
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

