## angler secrets

Enterprise Secret Scopes, Encrypted Keys, and Scope ACLs Management

### Synopsis

Manage workspace secret scopes, encrypted key-value pairs, and fine-grained
access control rules (READ, WRITE, MANAGE) backed by Internal DB, Azure Key Vault,
AWS Secrets Manager, HashiCorp Vault, or Google Cloud Secret Manager.

Modeled after 'databricks secrets' for seamless CI/CD and developer workflows.

Examples:
  angler secrets scopes list
  angler secrets scopes create prod-vault --backend hashicorp_vault --config '{"mount_point":"secret"}'
  angler secrets put prod-vault db_password --string-value "s3cr3t!"
  echo "s3cr3t!" | angler secrets put prod-vault db_password --value-stdin
  angler secrets list prod-vault
  angler secrets get prod-vault db_password
  export DB_PASS=$(angler secrets get prod-vault db_password --raw)
  angler secrets acls list prod-vault
  angler secrets acls set prod-vault --principal read --principal-type role --permission READ

```
angler secrets [flags]
```

### Options

```
  -h, --help   help for secrets
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse
* [angler secrets acls](angler_secrets_acls.md)	 - List, grant, and revoke access control rules on a secret scope
* [angler secrets delete](angler_secrets_delete.md)	 - Delete a secret key from a scope
* [angler secrets get](angler_secrets_get.md)	 - Retrieve secret metadata or value (masked in table output unless --unmask or --raw is used)
* [angler secrets keys](angler_secrets_keys.md)	 - Manage secret keys within a scope
* [angler secrets list](angler_secrets_list.md)	 - List secret key names, versions, and timestamps in a scope (without exposing values)
* [angler secrets list-scopes](angler_secrets_list-scopes.md)	 - List all secret scopes in the active workspace (Databricks-compatible alias)
* [angler secrets put](angler_secrets_put.md)	 - Create or rotate a secret value in a scope
* [angler secrets scopes](angler_secrets_scopes.md)	 - Create, inspect, list, and delete workspace secret scopes

