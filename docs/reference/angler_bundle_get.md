## angler bundle get

Inspect a deployed Lake Asset Bundle and run live state drift detection

### Synopsis

Retrieve detailed deployment metadata, resource inventory counts, provisioned UUID map,
and a live state drift report comparing the bundle manifest against active workspace resources.

Examples:
  angler bundle get
  angler bundle get customer-analytics -T dev
  angler bundle inspect 96fd3d6c-2c6e-4b81-8198-453ea28a085a --manifest
  angler bundle drift my-lakehouse --no-drift

```
angler bundle get [bundle_id_or_name] [flags]
```

### Options

```
      --bundle-id string   UUID of the deployed bundle
  -h, --help               help for get
      --manifest           Print stored raw manifest snapshot
      --no-drift           Skip live drift detection check
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

