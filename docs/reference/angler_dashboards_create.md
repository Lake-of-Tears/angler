## angler dashboards create

Deploy a new native BI dashboard from specification or template

### Synopsis

Creates a new dashboard in the active workspace.
Optionally load a complete specification from a JSON/YAML file, or initialize
from built-in templates ('blank', 'executive', 'traffic').

```
angler dashboards create [flags]
```

### Options

```
      --description string   Dashboard description
  -h, --help                 help for create
      --published            Publish status (default true) (default true)
      --spec string          Path to dashboard JSON or YAML spec file (or '-' for stdin)
      --template string      Starter template: blank, executive, traffic (default "blank")
      --title string         Dashboard title
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

* [angler dashboards](angler_dashboards.md)	 - Lake of Tears BI Dashboards, Canvas Layouts, and Visual Widgets

