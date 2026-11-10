## angler context set

Create or update a context profile

```
angler context set <name> [flags]
```

### Options

```
  -e, --endpoint string       API endpoint URL
  -h, --help                  help for set
  -w, --workspace string      workspace slug or name
      --workspace-id string   workspace ID or UUID
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --force                 bypass confirmation prompts
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
      --on-behalf-of string   impersonate target user email via trusted superadmin service token (X-On-Behalf-Of)
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler context](angler_context.md)	 - Manage connection environments and profiles

