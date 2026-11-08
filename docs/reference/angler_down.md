## angler down

Stop and dock the Lake of Tears stack (alias: dock)

### Synopsis

Stops and removes Lake of Tears containers and networks.

Examples:
  angler down
  angler dock -v

```
angler down [flags]
```

### Options

```
  -f, --file string   path to docker-compose.yml
  -h, --help          help for down
  -v, --volumes       remove named volumes declared in compose
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse

