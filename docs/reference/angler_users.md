## angler users

Lake of Tears User and Identity Management (IAM)

### Synopsis

Manage Lake of Tears user accounts, global roles, active status, and credentials.
Modeled after Databricks Users and IAM controls.

Requires cluster superadmin privileges.

Aliases:
  angler users
  angler user

Examples:
  angler users list
  angler users get u-123
  angler users update u-123 --role superadmin
  angler users update u-123 --name "Jane Doe"

### Options

```
  -h, --help   help for users
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
* [angler users get](angler_users_get.md)	 - Inspect detailed user profile and authentication configuration
* [angler users list](angler_users_list.md)	 - List all registered user accounts across the cluster
* [angler users update](angler_users_update.md)	 - Modify user role, active status, or display name

