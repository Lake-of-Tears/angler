## angler query

Execute SQL query against the data lake (alias: cast)

### Synopsis

Execute SQL queries against DuckDB and S3 Parquet tables in Lake of Tears.
You can specify the query as an argument or using the -e / --execute flag.

Aliases:
  angler query    - Standard analytical query command
  angler cast     - Nautical Lake of Tears query command

```
angler query [SQL] [flags]
```

### Options

```
      --catalog string     Catalog scope (default "main")
  -e, --execute string     SQL query statement to execute
      --explain            Explain execution plan prior to running
  -f, --file string        Path to file containing SQL query
      --format string      Output format: table, json, yaml, toon, csv, ndjson (default "table")
  -h, --help               help for query
      --limit int          Maximum number of rows to return (default 1000)
      --params string      JSON parameters array or object
      --schema string      Schema scope (default "default")
      --warehouse string   Execution warehouse ID (default "default")
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

