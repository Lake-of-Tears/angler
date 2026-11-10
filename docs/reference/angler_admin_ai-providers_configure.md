## angler admin ai-providers configure

Save and optionally activate an AI provider configuration (gemini, vertex, openai, anthropic, azure_openai, bedrock)

```
angler admin ai-providers configure <provider> [flags]
```

### Options

```
      --activate                      Set this provider as the active cluster AI provider (default true)
      --api-key string                Provider API key
      --base-url string               Custom base URL or Azure endpoint
      --chat-model string             Default chat/completion model ID
      --config string                 JSON configuration dictionary for the AI provider
      --config-file string            Path to JSON configuration file
      --embedding-model string        Default text embedding model ID
  -h, --help                          help for configure
      --location string               Google Cloud Vertex AI region/location (e.g. us-central1)
      --project-id string             Google Cloud Vertex AI project ID
      --region string                 AWS Bedrock region
      --service-account-file string   Path to Google Cloud Service Account JSON key file
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

* [angler admin ai-providers](angler_admin_ai-providers.md)	 - Configure, test, and discover models across Enterprise AI Providers (Vertex AI, Bedrock, OpenAI, Anthropic, Azure, Gemini)

