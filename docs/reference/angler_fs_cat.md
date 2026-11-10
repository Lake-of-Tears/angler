## angler fs cat

Stream or print the content of a remote file to stdout

### Synopsis

Streams the content of a remote file directly to standard output. Supports byte-range slicing.

```
angler fs cat <path> [flags]
```

### Options

```
  -h, --help         help for cat
      --length int   Byte length to read (0 reads to EOF)
      --offset int   Byte offset start
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

* [angler fs](angler_fs.md)	 - Filesystem and storage management modeled after Databricks FS (alias: storage)

