## angler secrets acls

List, grant, and revoke access control rules on a secret scope

### Options

```
  -h, --help   help for acls
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
* [angler secrets acls delete](angler_secrets_acls_delete.md)	 - Revoke an ACL rule from a secret scope
* [angler secrets acls list](angler_secrets_acls_list.md)	 - List all access control rules on a secret scope
* [angler secrets acls set](angler_secrets_acls_set.md)	 - Grant or update an ACL rule on a secret scope (READ, WRITE, MANAGE)

