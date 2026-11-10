## angler catalog

Unity Catalog-style metadata discovery and management (alias: net)

### Synopsis

Inspect datasets, catalogs, database schemas, tables, and data anomalies in Lake of Tears.
Modeled after Databricks Unity Catalog (catalogs, schemas, tables).

Aliases:
  angler catalog  - Unity Catalog standard command suite
  angler net      - Nautical Lake of Tears dataset inspect alias

Examples:
  angler catalog list
  angler catalog get main
  angler catalog schemas list main
  angler catalog tables list main default
  angler catalog tables get main default users
  angler catalog manifest
  angler catalog anomalies detect --table users

```
angler catalog [flags]
```

### Options

```
  -h, --help   help for catalog
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler catalog anomalies](angler_catalog_anomalies.md)	 - Data and table anomaly detection suite
* [angler catalog create](angler_catalog_create.md)	 - Register a new catalog in the workspace
* [angler catalog delete](angler_catalog_delete.md)	 - Delete or purge a catalog
* [angler catalog get](angler_catalog_get.md)	 - Inspect catalog metadata, schemas, and workspace ownership
* [angler catalog ls](angler_catalog_ls.md)	 - List all registered catalogs in the active workspace
* [angler catalog manifest](angler_catalog_manifest.md)	 - Export catalog manifest projection JSON
* [angler catalog schemas](angler_catalog_schemas.md)	 - Manage and inspect catalog database schemas
* [angler catalog tables](angler_catalog_tables.md)	 - Manage and inspect catalog tables

