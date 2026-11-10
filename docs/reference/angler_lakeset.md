## angler lakeset

Native Zero-Login Lakeset BI Studio, DuckDB Semantic Sync & Embedded Analytics (alias: bi)

### Synopsis

Inspect Lakeset BI Studio health, browse governed workspace and shared catalog datasets
with semantic column descriptions, trigger automated DuckDB + MinIO S3 dataset synchronization
in Superset, mint RLS-scoped Guest Tokens, execute live DuckDB S3 Chart Builder aggregations,
and open zero-login deep-links into Lakeset Studio.

Aliases:
  angler lakeset
  angler bi

Examples:
  angler lakeset status
  angler lakeset datasets --tier gold --search revenue
  angler lakeset sync --catalog main --force
  angler lakeset guest-token --dataset main.gold.revenue_summary --ttl 3600
  angler lakeset explore --dataset main.gold.revenue_summary --dimension region --metric amount --agg SUM
  angler lakeset open --dataset main.gold.revenue_summary

```
angler lakeset [flags]
```

### Options

```
  -h, --help   help for lakeset
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
* [angler lakeset datasets](angler_lakeset_datasets.md)	 - List RBAC-authorized workspace and shared catalog datasets in Lakeset BI Studio
* [angler lakeset explore](angler_lakeset_explore.md)	 - Execute live DuckDB S3 Chart Builder aggregations and SQL preview
* [angler lakeset guest-token](angler_lakeset_guest-token.md)	 - Mint a short-lived Superset Guest Token (JWT) with workspace RLS rules
* [angler lakeset open](angler_lakeset_open.md)	 - Open or print zero-login deep-links into the Lakeset BI Studio web UI
* [angler lakeset status](angler_lakeset_status.md)	 - Display Lakeset BI Studio health, DuckDB Lakehouse status, and dataset counts
* [angler lakeset sync](angler_lakeset_sync.md)	 - Register DuckDB Lakehouse database in Superset and sync catalog datasets

