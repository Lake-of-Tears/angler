## angler compute

Manage and inspect Lake of Tears cluster compute engines and topology

### Synopsis

Inspect and control cluster compute services (DuckDB, MinIO, Airflow, Jupyter, Superset)
and deployment topology across Kubernetes, Helm, and local stacks.

Aliases:
  angler clusters

```
angler compute [flags]
```

### Options

```
  -h, --help   help for compute
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
* [angler compute list](angler_compute_list.md)	 - List compute engines, pods, and resource limits
* [angler compute restart](angler_compute_restart.md)	 - Trigger a rolling restart of a platform deployment
* [angler compute services](angler_compute_services.md)	 - List Kubernetes platform deployments, replica counts, pod states, and resource specs
* [angler compute start](angler_compute_start.md)	 - Scale a stopped platform service back to active replicas
* [angler compute status](angler_compute_status.md)	 - Check compute cluster health, uptime, and service allocations
* [angler compute stop](angler_compute_stop.md)	 - Scale a non-core platform service deployment to 0 replicas
* [angler compute topology](angler_compute_topology.md)	 - Inspect cluster ingress routes, URLs, and persistent storage volumes

