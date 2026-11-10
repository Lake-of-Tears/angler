## angler secrets scopes

Create, inspect, list, and delete workspace secret scopes

```
angler secrets scopes [flags]
```

### Options

```
  -h, --help   help for scopes
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
* [angler secrets scopes create](angler_secrets_scopes_create.md)	 - Create a new workspace secret scope
* [angler secrets scopes delete](angler_secrets_scopes_delete.md)	 - Delete a secret scope and its keys
* [angler secrets scopes get](angler_secrets_scopes_get.md)	 - Inspect a secret scope's metadata and provider configuration
* [angler secrets scopes list](angler_secrets_scopes_list.md)	 - List all secret scopes in the active workspace

