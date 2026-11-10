## angler secrets acls set

Grant or update an ACL rule on a secret scope (READ, WRITE, MANAGE)

```
angler secrets acls set <scope> [flags]
```

### Options

```
  -h, --help                    help for set
      --permission string       Permission level: READ, WRITE, or MANAGE (default "READ")
      --principal string        Target user ID/email or workspace role (admin, write, read)
      --principal-type string   Principal type: 'role' or 'user' (default "role")
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

* [angler secrets acls](angler_secrets_acls.md)	 - List, grant, and revoke access control rules on a secret scope

