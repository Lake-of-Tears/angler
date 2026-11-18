## angler bundle destroy

Tear down and unprovision bundle resources in the target workspace

### Synopsis

Deletes the tables, pipelines, and dashboards managed by the bundle. Tables are moved to
Trash and can be restored (UNDROP) until the Trash retention period ends; a table that another
pipeline still references is kept. Catalogs and schemas the bundle created itself are removed
once empty (soft delete, recoverable); catalogs and schemas that already existed are never
removed. Core medallion schemas (bronze, silver, gold) go only with a bundle-created catalog.
Use --keep-catalogs to keep every catalog and schema.

Resources that another active record of the same bundle also references (for example the DAG and
tables of a live target) are never deleted; they are reported as shared and kept.

Use --dry-run to list what would be deleted or kept, and --record-only to remove only the
bundle record while keeping every provisioned resource (e.g. to clear a stale record).

Examples:
  angler bundle destroy -T dev -y
  angler bundle destroy -T dev --purge-storage -y
  angler bundle destroy -T dev --keep-catalogs -y
  angler bundle destroy -T default --dry-run
  angler bundle destroy -T default --record-only -y

```
angler bundle destroy [flags]
```

### Options

```
      --bundle-id string   UUID of the deployed bundle
      --dry-run            List what would be deleted or kept (including shared resources) without changing anything
  -h, --help               help for destroy
      --keep-catalogs      Keep catalogs and schemas the bundle created, even when empty
      --purge-storage      Also permanently purge raw Parquet files and vector embeddings in object storage
      --record-only        Remove only the bundle record and keep all provisioned resources
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
      --toon                  output results in TOON format
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
      --yaml                  output results in YAML format
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler bundle](angler_bundle.md)	 - Develop, validate, deploy, and manage Lake Asset Bundles (LABs)

