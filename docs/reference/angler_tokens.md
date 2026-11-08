## angler tokens

Manage Personal Access Tokens (PATs) for Lake of Tears

### Synopsis

Create, list, and revoke Personal Access Tokens (PATs) for automated CI/CD pipelines,
autonomous agents, and headless scripts. Modeled after cloud token lifecycles.

### Options

```
  -h, --help   help for tokens
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
* [angler tokens create](angler_tokens_create.md)	 - Generate a new Personal Access Token
* [angler tokens list](angler_tokens_list.md)	 - List active Personal Access Tokens registered to your account
* [angler tokens revoke](angler_tokens_revoke.md)	 - Revoke a Personal Access Token immediately

