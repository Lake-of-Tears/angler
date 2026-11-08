## angler tokens revoke

Revoke a Personal Access Token immediately

### Synopsis

Immediately invalidate an active Personal Access Token.

Any scripts, workers, or CI/CD pipelines using the revoked token will immediately lose access.

Examples:
  angler tokens revoke 99999999-8888-7777-6666-555555555555
  angler tokens revoke 99999999-8888-7777-6666-555555555555 --yes

```
angler tokens revoke <token_id> [flags]
```

### Options

```
  -f, --force   bypass confirmation prompt
  -h, --help    help for revoke
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler tokens](angler_tokens.md)	 - Manage Personal Access Tokens (PATs) for Lake of Tears

