## angler admin

Lake of Tears System Administration, Settings, and Session Auditing

### Synopsis

Administrative command suite for cluster superadmins.
Inspect and update cluster-wide runtime settings, review active sessions,
and revoke compromised tokens or connections.

Requires cluster superadmin privileges.

Examples:
  angler admin settings get
  angler admin settings update --session-timeout 120 --gemini-model gemini-2.0-flash
  angler admin sessions list
  angler admin sessions revoke sess-123 --force

### Options

```
  -h, --help   help for admin
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
* [angler admin ai-providers](angler_admin_ai-providers.md)	 - Configure, test, and discover models across Enterprise AI Providers (Vertex AI, Bedrock, OpenAI, Anthropic, Azure, Gemini)
* [angler admin services](angler_admin_services.md)	 - Inspect and control Kubernetes platform service deployments (list, restart, stop, start)
* [angler admin sessions](angler_admin_sessions.md)	 - Audit and manage active user and API sessions
* [angler admin settings](angler_admin_settings.md)	 - Inspect and update system-wide configuration settings
* [angler admin vaults](angler_admin_vaults.md)	 - Configure, test, and audit Enterprise Secret Vault providers (Internal, Azure KV, AWS SM, HashiCorp Vault, GCP SM)

