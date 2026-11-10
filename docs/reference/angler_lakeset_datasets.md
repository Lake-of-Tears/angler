## angler lakeset datasets

List RBAC-authorized workspace and shared catalog datasets in Lakeset BI Studio

```
angler lakeset datasets [flags]
```

### Options

```
      --catalog string   Filter datasets by catalog slug or name
  -h, --help             help for datasets
      --schema string    Filter datasets by schema slug or name
  -q, --search string    Search dataset names, table descriptions, and columns
      --tier string      Filter datasets by medallion tier (bronze, silver, gold)
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

* [angler lakeset](angler_lakeset.md)	 - Native Zero-Login Lakeset BI Studio, DuckDB Semantic Sync & Embedded Analytics (alias: bi)

