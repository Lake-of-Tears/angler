## angler bundle run

Trigger ad-hoc execution of a bundle pipeline or DAG

### Synopsis

Execute an Airflow pipeline defined within the deployed bundle.
Pass --conf to inject runtime JSON parameters into the DAG run, and --wait to poll until completion.

Examples:
  angler bundle run orders_etl -T dev
  angler bundle run orders_etl -T dev --conf '{"batch_date":"2026-11-10"}' --wait
  angler bundle run --bundle-id 96fd3d6c-2c6e-4b81-8198-453ea28a085a

```
angler bundle run [pipeline] [flags]
```

### Options

```
      --bundle-id string         UUID of the deployed bundle
      --conf string              Runtime configuration JSON payload passed to the pipeline DAG run
  -h, --help                     help for run
      --poll-interval duration   Polling interval when waiting for pipeline completion (default 3s)
      --timeout duration         Timeout when waiting for pipeline completion (default 10m0s)
      --wait                     Wait for triggered pipeline run to complete before exiting
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

