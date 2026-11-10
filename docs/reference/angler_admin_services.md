## angler admin services

Inspect and control Kubernetes platform service deployments (list, restart, stop, start)

```
angler admin services [flags]
```

### Options

```
  -h, --help   help for services
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
* [angler admin services list](angler_admin_services_list.md)	 - List Kubernetes platform deployments, replica counts, pod states, and resource specs
* [angler admin services restart](angler_admin_services_restart.md)	 - Trigger a rolling restart of a platform deployment
* [angler admin services start](angler_admin_services_start.md)	 - Scale a stopped platform service back to active replicas
* [angler admin services stop](angler_admin_services_stop.md)	 - Scale a non-core platform service deployment to 0 replicas

