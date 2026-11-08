## angler ai test-key

Validate configured LLM provider credentials (e.g. Gemini, OpenAI)

### Synopsis

Validates configured or candidate LLM API keys against the backend provider verification endpoint.

```
angler ai test-key [flags]
```

### Options

```
  -h, --help           help for test-key
      --key string     Candidate API key to test (uses configured key if omitted)
      --model string   LLM model to verify (default "gemini-2.0-flash")
```

### Options inherited from parent commands

```
      --config string         config file (default is $HOME/.angler/config.yaml)
  -c, --context string        context profile to use
      --endpoint string       Lake of Tears API endpoint
      --force                 bypass confirmation prompts
      --json                  output results in JSON format
      --non-interactive       disable interactive prompts
  -o, --output string         output format: table, json, yaml, toon
  -t, --token string          API authentication token
  -w, --workspace string      Lake of Tears workspace slug or name
      --workspace-id string   explicit Lake of Tears workspace ID
  -y, --yes                   automatic yes to prompts; run non-interactively
```

### SEE ALSO

* [angler ai](angler_ai.md)	 - Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)

