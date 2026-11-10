## angler admin vaults

Configure, test, and audit Enterprise Secret Vault providers (Internal, Azure KV, AWS SM, HashiCorp Vault, GCP SM)

```
angler admin vaults [flags]
```

### Options

```
  -h, --help   help for vaults
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

* [angler admin](angler_admin.md)	 - Lake of Tears System Administration, Settings, and Session Auditing
* [angler admin vaults configure](angler_admin_vaults_configure.md)	 - Update a vault provider's configuration
* [angler admin vaults get](angler_admin_vaults_get.md)	 - Inspect a specific vault provider's configuration (secrets masked)
* [angler admin vaults list](angler_admin_vaults_list.md)	 - List all supported vault providers and their configuration status
* [angler admin vaults scopes](angler_admin_vaults_scopes.md)	 - List all secret scopes across all workspaces for global security auditing
* [angler admin vaults test](angler_admin_vaults_test.md)	 - Execute live reachability, authentication, and seal status verification against a vault provider

