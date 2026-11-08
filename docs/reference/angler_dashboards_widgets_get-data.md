## angler dashboards widgets get-data

Execute widget analytical query and return live aggregated chart data

```
angler dashboards widgets get-data <dashboard_id> <widget_id> [flags]
```

### Options

```
  -h, --help                help for get-data
      --time-range string   Time window filter (e.g. 'last_30_days', 'last_7_days')
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --force                 bypass confirmation prompts
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler dashboards widgets](angler_dashboards_widgets.md)	 - Inspect dashboard widget layout and execute live visualization queries

