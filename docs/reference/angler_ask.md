## angler ask

Ask questions in natural language with AI search (alias: sonar)

### Synopsis

Transmit a sonar pulse into your Lake of Tears data.
Uses vector embeddings and Gemini RAG to answer questions grounded in your data.

Examples:
  angler ask "Which region had the highest revenue drop last quarter?"
  angler sonar "Show anomalies detected in server telemetry yesterday"

```
angler ask [QUESTION] [flags]
```

### Options

```
  -h, --help   help for ask
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

