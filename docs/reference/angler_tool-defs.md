## angler tool-defs

Export LLM tool calling schemas (Gemini, OpenAI, OpenAPI 3.1)

### Synopsis

Introspect the Angler CLI command tree and export standard function/tool definitions
for LLM agent frameworks (Google Gemini, OpenAI, Anthropic, LangChain, AutoGen, CrewAI, Antigravity).

Examples:
  angler tool-defs --format gemini
  angler tool-defs --format openai
  angler schema --format openapi
  angler tool-defs --format openai --output yaml

```
angler tool-defs [flags]
```

### Options

```
  -f, --format string          Schema output format: gemini, openai, openapi (default "openai")
  -h, --help                   help for tool-defs
      --include-global-flags   Include global persistent flags in every tool definition
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

* [angler](angler.md)	 - The official CLI for Lake of Tears data lakehouse

