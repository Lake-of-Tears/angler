## angler catalog tables

Manage and inspect catalog tables

### Synopsis

List, inspect, and drop tables within a catalog schema.

### Options

```
  -h, --help   help for tables
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

* [angler catalog](angler_catalog.md)	 - Unity Catalog-style metadata discovery and management (alias: net)
* [angler catalog tables delete](angler_catalog_tables_delete.md)	 - Drop a table from a catalog schema
* [angler catalog tables get](angler_catalog_tables_get.md)	 - Inspect table schema, column types, and storage pattern
* [angler catalog tables list](angler_catalog_tables_list.md)	 - List all tables in a catalog schema
* [angler catalog tables update](angler_catalog_tables_update.md)	 - Update table description or S3 path pattern

