## angler lakeset guest-token

Mint a short-lived Superset Guest Token (JWT) with workspace RLS rules

```
angler lakeset guest-token [flags]
```

### Options

```
      --dashboard-id string   Optional target dashboard UUID/ID
      --dataset string        Optional target dataset full name (catalog.schema.table)
  -h, --help                  help for guest-token
      --raw                   Print raw JWT guest token string to stdout
      --ttl int               Token lifetime in seconds (60..86400) (default 3600)
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

