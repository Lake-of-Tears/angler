## angler catalog trash

Manage soft-deleted catalogs, schemas, and tables in the workspace trash

```
angler catalog trash [flags]
```

### Options

```
  -h, --help   help for trash
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
* [angler catalog trash list](angler_catalog_trash_list.md)	 - List soft-deleted catalogs, schemas, and tables in the workspace trash
* [angler catalog trash purge](angler_catalog_trash_purge.md)	 - Permanently delete a specific trashed asset
* [angler catalog trash purge-expired](angler_catalog_trash_purge-expired.md)	 - Permanently purge all trashed assets past their retention TTL
* [angler catalog trash restore](angler_catalog_trash_restore.md)	 - Restore a soft-deleted catalog, schema, or table from trash

