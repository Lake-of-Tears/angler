## angler lakeset explore

Execute live DuckDB S3 Chart Builder aggregations and SQL preview

```
angler lakeset explore [flags]
```

### Options

```
      --agg string          Aggregation function (COUNT, SUM, AVG, MIN, MAX) (default "count")
      --chart-type string   Chart aggregation format (bar, line, area, pie, kpi, table) (default "bar")
      --dataset string      Target governed dataset (catalog.schema.table or schema.table)
      --dimension string    Grouping dimension column
  -h, --help                help for explore
      --limit int           Maximum series categories/rows (1..500) (default 50)
      --metric string       Metric column (default "count")
      --sql string          Custom analytical DuckDB SQL query override
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

