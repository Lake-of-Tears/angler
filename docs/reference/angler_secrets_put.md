## angler secrets put

Create or rotate a secret value in a scope

```
angler secrets put <scope> <key> [flags]
```

### Options

```
  -h, --help                  help for put
      --string-value string   Secret string value to store
      --value-stdin           Read secret value from standard input (safe for CI/CD)
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

* [angler secrets](angler_secrets.md)	 - Enterprise Secret Scopes, Encrypted Keys, and Scope ACLs Management

