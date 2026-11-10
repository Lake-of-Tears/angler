## angler fs

Filesystem and storage management modeled after Databricks FS (alias: storage)

### Synopsis

Browse, stream, copy, upload, and remove files stored in Lake of Tears object stores (S3/MinIO)
and workspace volumes.

Modeled after Databricks FS (dbfs) and Volumes, enabling both human engineers and LLM agents
to inspect raw datasets, stream logs, upload artifacts, and download query extracts.

Aliases:
  angler fs       - Databricks FS standard command suite
  angler storage  - Lake of Tears storage alias

Examples:
  angler fs ls gold/telemetry
  angler fs cat gold/telemetry/sample.json --length 1024
  angler fs cp ./data.csv lot://bronze/data.csv
  angler fs cp lot://gold/report.parquet ./report.parquet
  angler fs stat gold/telemetry/data.parquet
  angler fs upload ./model.onnx models/
  angler fs download gold/summary.csv ./summary.csv
  angler fs rm bronze/old_data/ --recursive --force
  angler fs stats

```
angler fs [flags]
```

### Options

```
  -h, --help   help for fs
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
* [angler fs cat](angler_fs_cat.md)	 - Stream or print the content of a remote file to stdout
* [angler fs cp](angler_fs_cp.md)	 - Copy files between local filesystem and remote Lake of Tears storage
* [angler fs download](angler_fs_download.md)	 - Download remote objects to local disk
* [angler fs ls](angler_fs_ls.md)	 - List files, directories, and partitions in object storage
* [angler fs mkdir](angler_fs_mkdir.md)	 - Create a directory prefix marker in workspace storage
* [angler fs rm](angler_fs_rm.md)	 - Delete objects or prefixes from workspace storage
* [angler fs stat](angler_fs_stat.md)	 - Retrieve file metadata and Parquet schema inspection
* [angler fs stats](angler_fs_stats.md)	 - Get aggregated storage metrics and top-level breakdown
* [angler fs upload](angler_fs_upload.md)	 - Upload local artifacts, datasets, or models to remote storage

