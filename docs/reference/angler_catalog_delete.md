## angler catalog delete

Delete or purge a catalog

```
angler catalog delete <catalog_id> [flags]
```

### Options

```
  -f, --force   Force delete without confirmation
  -h, --help    help for delete
      --purge   Permanently purge catalog and bypass soft delete
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler catalog](angler_catalog.md)	 - Unity Catalog-style metadata discovery and management (alias: net)

