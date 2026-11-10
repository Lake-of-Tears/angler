## angler catalog schemas

Manage and inspect catalog database schemas

### Synopsis

List, create, inspect, and delete schemas within a catalog.

### Options

```
  -h, --help   help for schemas
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
* [angler catalog schemas create](angler_catalog_schemas_create.md)	 - Create a database schema in a catalog
* [angler catalog schemas delete](angler_catalog_schemas_delete.md)	 - Drop a schema from a catalog
* [angler catalog schemas get](angler_catalog_schemas_get.md)	 - Inspect schema details and its tables
* [angler catalog schemas list](angler_catalog_schemas_list.md)	 - List all schemas in a catalog
* [angler catalog schemas update](angler_catalog_schemas_update.md)	 - Update schema description

