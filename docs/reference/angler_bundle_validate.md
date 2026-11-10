## angler bundle validate

Validate bundle manifest syntax, variable bindings, and DAG cycles

### Synopsis

Validate a Lake Asset Bundle manifest against syntax requirements, variable interpolation,
pipeline DAG acyclicity, task virtualenv requirements, @webhook schedules, and remote workspace constraints.

Examples:
  angler bundle validate --local
  angler bundle validate -T dev
  angler bundle validate -f custom.toon -T prod

```
angler bundle validate [flags]
```

### Options

```
  -h, --help    help for validate
      --local   Validate manifest syntax offline without contacting server
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

