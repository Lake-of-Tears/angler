## angler jobs

Data pipelines, scheduled jobs, and workflow DAGs management (alias: pipelines)

### Synopsis

Manage Lake of Tears data pipelines, scheduled notebook runs, dbt tasks, and workflow DAGs.

Modeled after Databricks Jobs (jobs) and Pipelines, providing full programmatic lifecycle
control for data engineers and autonomous LLM agents to trigger DAG runs, monitor progress,
cancel executions, and inspect task logs.

Aliases:
  angler jobs
  angler pipelines
  angler job
  angler pipeline

Examples:
  angler jobs list
  angler jobs get telemetry_etl
  angler jobs run telemetry_etl --wait
  angler jobs runs list telemetry_etl
  angler jobs runs get telemetry_etl manual__20261108T120000Z__user
  angler jobs logs telemetry_etl manual__20261108T120000Z__user extract
  angler jobs cancel telemetry_etl manual__20261108T120000Z__user
  angler jobs pause telemetry_etl
  angler jobs unpause telemetry_etl

```
angler jobs [flags]
```

### Options

```
  -h, --help   help for jobs
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
* [angler jobs cancel](angler_jobs_cancel.md)	 - Cancel an in-flight pipeline run
* [angler jobs delete](angler_jobs_delete.md)	 - Delete a scheduled notebook pipeline or DAG
* [angler jobs get](angler_jobs_get.md)	 - Retrieve comprehensive job configuration, schedule, and tasks
* [angler jobs list](angler_jobs_list.md)	 - List all active and scheduled pipelines, DAGs, and jobs
* [angler jobs logs](angler_jobs_logs.md)	 - Fetch stdout/stderr execution logs for a task instance
* [angler jobs pause](angler_jobs_pause.md)	 - Pause scheduled execution of a pipeline
* [angler jobs run](angler_jobs_run.md)	 - Immediately trigger a pipeline run
* [angler jobs runs](angler_jobs_runs.md)	 - List, inspect, and manage historical pipeline runs
* [angler jobs schedule](angler_jobs_schedule.md)	 - Update a pipeline's cron schedule or preset (@webhook, @daily, @hourly, etc.)
* [angler jobs schedule-notebook](angler_jobs_schedule-notebook.md)	 - Create or update a scheduled notebook pipeline DAG
* [angler jobs settings](angler_jobs_settings.md)	 - Inspect and update pipeline display metadata, timeline visibility, and webhook triggers
* [angler jobs timeline](angler_jobs_timeline.md)	 - Render the 24-hour workspace pipeline execution schedule timeline
* [angler jobs unpause](angler_jobs_unpause.md)	 - Unpause/resume scheduled execution of a pipeline
* [angler jobs webhook](angler_jobs_webhook.md)	 - Enable, disable, and trigger native inbound pipeline webhooks

