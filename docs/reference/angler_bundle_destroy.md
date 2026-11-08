## angler bundle destroy

Tear down and unprovision bundle resources in the target workspace

### Synopsis

Permanently deletes provisioned tables, pipelines, and dashboards managed by the bundle.
Core medallion schemas (bronze, silver, gold) remain protected.

Examples:
  angler bundle destroy -t dev -y
  angler bundle destroy -t dev --purge-storage -y

```
angler bundle destroy [flags]
```

### Options

```
      --bundle-id string   UUID of the deployed bundle
  -h, --help               help for destroy
      --purge-storage      Also permanently purge raw Parquet files and vector embeddings in object storage
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
  -o, --output string         output format: table, json, yaml, toon
  -T, --target string         Target environment (e.g. dev, staging, prod)
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler bundle](angler_bundle.md)	 - Develop, validate, deploy, and manage Lake Asset Bundles (LABs)

