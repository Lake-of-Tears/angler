## angler admin ai-providers

Configure, test, and discover models across Enterprise AI Providers (Vertex AI, Bedrock, OpenAI, Anthropic, Azure, Gemini)

```
angler admin ai-providers [flags]
```

### Options

```
  -h, --help   help for ai-providers
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
* [angler admin ai-providers configure](angler_admin_ai-providers_configure.md)	 - Save and optionally activate an AI provider configuration (gemini, vertex, openai, anthropic, azure_openai, bedrock)
* [angler admin ai-providers deprecation](angler_admin_ai-providers_deprecation.md)	 - View active self-healing model deprecation fallback alerts
* [angler admin ai-providers discover-models](angler_admin_ai-providers_discover-models.md)	 - Dynamically query the provider's live API / Model Garden for available chat and embedding models
* [angler admin ai-providers dismiss-deprecation](angler_admin_ai-providers_dismiss-deprecation.md)	 - Clear active self-healing model deprecation fallback alerts
* [angler admin ai-providers list](angler_admin_ai-providers_list.md)	 - Inspect active AI provider and all provider configurations (secrets masked)
* [angler admin ai-providers test](angler_admin_ai-providers_test.md)	 - Probe chat latency, embedding latency, and vector dimensions for an AI provider

