## angler docs

Generate CLI reference documentation and shell autocompletions

### Synopsis

Generates complete markdown reference documentation for all Angler CLI commands
and subcommands using Cobra's markdown documentation generator.

Optionally generates shell completion scripts for bash, zsh, fish, and PowerShell.

Examples:
  angler docs
  angler docs ./docs/reference
  angler docs ./docs/reference --completions-dir ./completions

```
angler docs [output-dir] [flags]
```

### Options

```
      --clean                    Clean existing markdown files in output directory before generation (default true)
      --completions-dir string   Directory to generate shell completion scripts (bash, zsh, fish, ps1)
  -h, --help                     help for docs
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

