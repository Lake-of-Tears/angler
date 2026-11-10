## angler secrets scopes create

Create a new workspace secret scope

```
angler secrets scopes create <name> [flags]
```

### Options

```
      --backend string   Vault backend: internal, azure_keyvault, aws_secrets_manager, hashicorp_vault, gcp_secret_manager (default "internal")
      --config string    Provider-specific scope configuration JSON
  -h, --help             help for create
```

### Options inherited from parent commands

```
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

* [angler secrets scopes](angler_secrets_scopes.md)	 - Create, inspect, list, and delete workspace secret scopes

