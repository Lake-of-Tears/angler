## angler admin settings update

Update cluster-wide configuration settings

```
angler admin settings update [flags]
```

### Options

```
      --gemini-key string     Gemini API key
      --gemini-model string   Default Gemini AI model
  -h, --help                  help for update
      --max-hours int         Maximum session duration in hours (default 24)
      --session-timeout int   Session inactivity timeout in minutes (default 60)
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

* [angler admin settings](angler_admin_settings.md)	 - Inspect and update system-wide configuration settings

