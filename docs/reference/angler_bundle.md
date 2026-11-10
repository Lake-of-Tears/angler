## angler bundle

Develop, validate, deploy, and manage Lake Asset Bundles (LABs)

### Synopsis

📦 Lake Asset Bundles (LABs) enable Infrastructure-as-Code for your lakehouse.
Manage catalogs, schemas, DuckDB pipelines, vector indexes, and dashboards
declaratively using AI-first TOON (lake.toon) or YAML manifests.

Examples:
  angler bundle init my-lakehouse --template medallion
  angler bundle validate --local
  angler bundle deploy -t prod --dry-run
  angler bundle deploy -t dev
  angler bundle run orders_etl -t dev
  angler bundle destroy -t dev --purge-storage -y
  angler bundle ls -t dev

### Options

```
  -f, --file string     Path to bundle manifest file (lake.toon, lake.yml)
  -h, --help            help for bundle
  -T, --target string   Target environment (e.g. dev, staging, prod)
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
* [angler bundle deploy](angler_bundle_deploy.md)	 - Deploy Lake Asset Bundle resources and pipelines to the active lakehouse
* [angler bundle destroy](angler_bundle_destroy.md)	 - Tear down and unprovision bundle resources in the target workspace
* [angler bundle init](angler_bundle_init.md)	 - Scaffold a new Lake Asset Bundle project
* [angler bundle ls](angler_bundle_ls.md)	 - List deployed Lake Asset Bundles in the active workspace
* [angler bundle run](angler_bundle_run.md)	 - Trigger ad-hoc execution of a bundle pipeline or DAG
* [angler bundle validate](angler_bundle_validate.md)	 - Validate bundle manifest syntax, variable bindings, and DAG cycles

