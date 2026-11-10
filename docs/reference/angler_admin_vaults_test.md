## angler admin vaults test

Execute live reachability, authentication, and seal status verification against a vault provider

```
angler admin vaults test <provider> [flags]
```

### Options

```
      --auth-method string            Auth method (token, approle, kubernetes, adc, service_account)
      --client-id string              Azure client ID
      --client-secret string          Azure client secret
      --config string                 JSON configuration payload for vault provider
      --config-file string            Path to JSON configuration file
  -h, --help                          help for test
      --mount-point string            HashiCorp Vault KV mount point
      --namespace string              HashiCorp Vault Enterprise namespace
      --project-id string             GCP project ID
      --region string                 AWS region name
      --role-id string                HashiCorp Vault AppRole role_id
      --secret-id string              HashiCorp Vault AppRole secret_id
      --service-account-file string   Path to GCP Service Account JSON file
      --tenant-id string              Azure tenant ID
      --vault-token string            HashiCorp Vault token
      --vault-url string              Vault URL (Azure Key Vault or HashiCorp Vault)
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

* [angler admin vaults](angler_admin_vaults.md)	 - Configure, test, and audit Enterprise Secret Vault providers (Internal, Azure KV, AWS SM, HashiCorp Vault, GCP SM)

