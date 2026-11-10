## angler bundle generate

Generate a complete Lake Asset Bundle manifest from a natural language prompt

### Synopsis

Use the Lake of Tears AI Architect (with offline heuristic fallback) to synthesize
a complete, valid lake.toon or lake.yml Lake Asset Bundle manifest from a natural language description.

Examples:
  angler bundle generate "Build a customer 360 medallion pipeline with webhook ingestion"
  angler bundle generate "Biomedical PubMed RAG lakehouse with vector search" --format yaml --out lake.yml
  angler bundle generate "IoT sensor telemetry pipeline" --out lake.toon --deploy --dry-run

```
angler bundle generate [prompt] [flags]
```

### Options

```
      --deploy          Immediately deploy the generated bundle to the active workspace
      --dry-run         When used with --deploy, simulate deployment without persisting
      --format string   Generated manifest format: toon or yaml (default "toon")
  -h, --help            help for generate
      --out string      Write generated manifest directly to file path
      --prompt string   Natural language description of the lakehouse bundle to generate
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

