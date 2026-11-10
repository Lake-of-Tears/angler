## angler bundle init

Scaffold a new Lake Asset Bundle project

### Synopsis

Initialize a new Lake Asset Bundle with starter catalogs, schemas, pipelines, and manifests.

Available templates:
  • python-etl (default): Starter DuckDB pipeline and managed tables
  • rag-search: Lakehouse with vector embeddings and semantic search
  • medallion: Full Bronze, Silver, Gold tiered architecture with transformation stages

```
angler bundle init [name] [flags]
```

### Options

```
      --format string     Manifest format: toon or yaml (default "toon")
  -h, --help              help for init
      --template string   Template (python-etl, rag-search, medallion) (default "python-etl")
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
  -f, --file string           Path to bundle manifest file (lake.toon, lake.yml)
      --force                 bypass confirmation prompts
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
      --on-behalf-of string   impersonate target user email via trusted superadmin service token (X-On-Behalf-Of)
  -o, --output string         output format: table, json, yaml, toon
  -T, --target string         Target environment (e.g. dev, staging, prod)
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler bundle](angler_bundle.md)	 - Develop, validate, deploy, and manage Lake Asset Bundles (LABs)

