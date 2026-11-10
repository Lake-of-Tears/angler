## angler dashboards

Lake of Tears BI Dashboards, Canvas Layouts, and Visual Widgets

### Synopsis

Manage Lake of Tears native BI dashboards, canvas widget configurations,
and live aggregated analytical datasets.

Modeled after Databricks Dashboards, allowing data engineers, business analysts,
and autonomous LLM agents to inspect visualization layouts, extract live widget
query metrics, and programmatically deploy dashboards.

Aliases:
  angler dashboards
  angler dashboard
  angler dash

Examples:
  angler dashboards list
  angler dashboards get dash-123
  angler dashboards create --title "Executive KPIs" --template executive
  angler dashboards create --spec ./dashboards/sales_overview.json
  angler dashboards delete dash-123 --force
  angler dashboards widgets list dash-123
  angler dashboards widgets get-data dash-123 w_kpi_1 --time-range last_30_days

### Options

```
  -h, --help   help for dashboards
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
* [angler dashboards create](angler_dashboards_create.md)	 - Deploy a new native BI dashboard from specification or template
* [angler dashboards delete](angler_dashboards_delete.md)	 - Delete a dashboard from the current workspace
* [angler dashboards get](angler_dashboards_get.md)	 - Retrieve complete dashboard details, widgets, and layout configuration
* [angler dashboards list](angler_dashboards_list.md)	 - List all registered BI dashboards in the active workspace
* [angler dashboards stats](angler_dashboards_stats.md)	 - Display aggregate dashboard and chart metrics for the workspace
* [angler dashboards widgets](angler_dashboards_widgets.md)	 - Inspect dashboard widget layout and execute live visualization queries

