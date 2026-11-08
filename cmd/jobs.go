package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	jobsFormat       string
	jobsLimit        int
	jobsWait         bool
	jobsTimeout      time.Duration
	jobsPollInterval time.Duration
	jobsConf         string
	jobsForce        bool
	jobsTryNumber    int
)

var jobsCmd = &cobra.Command{
	Use:     "jobs",
	Aliases: []string{"pipelines", "job", "pipeline"},
	Short:   "Data pipelines, scheduled jobs, and workflow DAGs management (alias: pipelines)",
	Long: `Manage Lake of Tears data pipelines, scheduled notebook runs, dbt tasks, and workflow DAGs.

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
  angler jobs unpause telemetry_etl`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return runJobsGet(cmd, args)
		}
		return runJobsList(cmd, args)
	},
}

var jobsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all active and scheduled pipelines, DAGs, and jobs",
	Long:    `Lists all pipelines visible in the active workspace, with schedule intervals, pause state, and latest run status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runJobsList(cmd, args)
	},
}

func runJobsList(cmd *cobra.Command, args []string) error {
	c := getClient()
	resp, err := c.ListPipelines()
	if err != nil {
		return err
	}

	return ui.RenderOutput(resp.Pipelines, func() {
		if len(resp.Pipelines) == 0 {
			ui.PrintInfo("No pipelines found in active workspace.")
			return
		}

		headers := []string{"JOB ID", "NAME", "SCHEDULE", "LAST STATE", "LAST RUN AT", "PAUSED"}
		rows := make([][]string, len(resp.Pipelines))
		for i, p := range resp.Pipelines {
			name := p.PipelineName
			if name == "" {
				name = p.DagID
			}

			lastState := "-"
			lastRunAt := "-"
			if p.LastRun != nil {
				lastState = strings.ToUpper(p.LastRun.State)
				if p.LastRun.StartDate != nil {
					lastRunAt = p.LastRun.StartDate.Format("2006-01-02 15:04:05")
				}
			}

			pausedStr := "No"
			if p.IsPaused {
				pausedStr = "Yes"
			}

			sched := p.ScheduleInterval
			if sched == "" {
				sched = "Manual"
			}

			rows[i] = []string{
				p.DagID,
				name,
				sched,
				lastState,
				lastRunAt,
				pausedStr,
			}
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total jobs: %d", len(resp.Pipelines))
	})
}

var jobsGetCmd = &cobra.Command{
	Use:   "get <job_id>",
	Short: "Retrieve comprehensive job configuration, schedule, and tasks",
	Long:  `Inspects DAG configuration, tasks breakdown, schedule expressions, and latest run metrics.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runJobsGet(cmd, args)
	},
}

func runJobsGet(cmd *cobra.Command, args []string) error {
	dagID := args[0]
	c := getClient()

	detail, err := c.GetPipeline(dagID)
	if err != nil {
		return err
	}

	return ui.RenderOutput(detail, func() {
		title := detail.PipelineName
		if title == "" {
			title = detail.DagID
		}

		ui.PrintSuccess("Pipeline: %s (%s)", title, detail.DagID)
		if detail.Description != "" {
			fmt.Fprintf(ui.OutWriter, "  Description: %s\n", detail.Description)
		}
		sched := detail.ScheduleInterval
		if sched == "" {
			sched = "Manual"
		}
		fmt.Fprintf(ui.OutWriter, "  Schedule:    %s\n", sched)
		fmt.Fprintf(ui.OutWriter, "  Paused:      %v\n", detail.IsPaused)
		if len(detail.Tags) > 0 {
			fmt.Fprintf(ui.OutWriter, "  Tags:        %s\n", strings.Join(detail.Tags, ", "))
		}
		if detail.NotebookPath != "" {
			fmt.Fprintf(ui.OutWriter, "  Notebook:    %s\n", detail.NotebookPath)
		}

		if detail.LastRun != nil {
			fmt.Fprintln(ui.OutWriter, "\nLast Run:")
			fmt.Fprintf(ui.OutWriter, "  Run ID:      %s\n", detail.LastRun.DagRunID)
			fmt.Fprintf(ui.OutWriter, "  State:       %s\n", strings.ToUpper(detail.LastRun.State))
			if detail.LastRun.StartDate != nil {
				fmt.Fprintf(ui.OutWriter, "  Started:     %s\n", detail.LastRun.StartDate.Format("2006-01-02 15:04:05"))
			}
			if detail.LastRun.DurationSeconds != nil {
				fmt.Fprintf(ui.OutWriter, "  Duration:    %.1fs\n", *detail.LastRun.DurationSeconds)
			}
		}

		if len(detail.Tasks) > 0 {
			fmt.Fprintln(ui.OutWriter, "\nTasks:")
			headers := []string{"TASK ID", "OPERATOR"}
			rows := make([][]string, len(detail.Tasks))
			for i, t := range detail.Tasks {
				taskID, _ := t["task_id"].(string)
				op, _ := t["class_name"].(string)
				if op == "" {
					op, _ = t["operator"].(string)
				}
				rows[i] = []string{taskID, op}
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		}
	})
}

var jobsRunCmd = &cobra.Command{
	Use:     "run <job_id>",
	Aliases: []string{"trigger"},
	Short:   "Immediately trigger a pipeline run",
	Long: `Triggers on-demand execution of the specified pipeline.
When --wait is passed, blocks until completion, polls status, and displays the task breakdown upon finish.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		c := getClient()

		var confMap map[string]any
		if jobsConf != "" {
			if err := json.Unmarshal([]byte(jobsConf), &confMap); err != nil {
				return ui.NewUsageError(fmt.Errorf("invalid JSON for --conf: %w", err))
			}
		}

		resp, err := c.TriggerPipeline(dagID, confMap)
		if err != nil {
			return err
		}

		if !jobsWait {
			return ui.RenderOutput(resp, func() {
				ui.PrintSuccess("Triggered run '%s' for pipeline '%s' (State: %s)", resp.DagRunID, resp.DagID, strings.ToUpper(resp.State))
				fmt.Fprintf(ui.OutWriter, "Run ID: %s\n", resp.DagRunID)
				fmt.Fprintf(ui.OutWriter, "To view status: angler jobs runs get %s %s\n", dagID, resp.DagRunID)
			})
		}

		// Blocking wait with status polling
		ui.PrintInfo("Triggered run '%s'. Waiting for completion (timeout: %s)...", resp.DagRunID, jobsTimeout)

		deadline := time.Now().Add(jobsTimeout)
		pollInterval := jobsPollInterval
		if pollInterval <= 0 {
			pollInterval = 3 * time.Second
		}

		var finalRun *client.PipelineRun
		for time.Now().Before(deadline) {
			run, err := c.GetPipelineRun(dagID, resp.DagRunID)
			if err == nil {
				finalRun = run
				state := strings.ToLower(run.State)
				if state == "success" || state == "failed" || state == "cancelled" || state == "skipped" {
					break
				}
			}
			time.Sleep(pollInterval)
		}

		if finalRun == nil {
			return fmt.Errorf("timeout waiting for pipeline run '%s'", resp.DagRunID)
		}

		// Fetch final tasks
		tasksResp, _ := c.ListPipelineTaskInstances(dagID, resp.DagRunID)

		resultPayload := map[string]any{
			"dag_id":   dagID,
			"run_id":   finalRun.DagRunID,
			"state":    finalRun.State,
			"duration": finalRun.DurationSeconds,
			"tasks":    nil,
		}
		if tasksResp != nil {
			resultPayload["tasks"] = tasksResp.Tasks
		}

		return ui.RenderOutput(resultPayload, func() {
			if strings.ToLower(finalRun.State) == "success" {
				ui.PrintSuccess("Pipeline run completed: %s (State: %s)", finalRun.DagRunID, strings.ToUpper(finalRun.State))
			} else {
				ui.PrintError("Pipeline run finished with status: %s", strings.ToUpper(finalRun.State))
			}

			if tasksResp != nil && len(tasksResp.Tasks) > 0 {
				headers := []string{"TASK ID", "STATE", "TRY", "DURATION"}
				rows := make([][]string, len(tasksResp.Tasks))
				for i, t := range tasksResp.Tasks {
					durStr := "-"
					if t.DurationSeconds != nil {
						durStr = fmt.Sprintf("%.1fs", *t.DurationSeconds)
					}
					rows[i] = []string{
						t.TaskID,
						strings.ToUpper(t.State),
						fmt.Sprintf("%d", t.TryNumber),
						durStr,
					}
				}
				fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			}
		})
	},
}

var jobsCancelCmd = &cobra.Command{
	Use:     "cancel <job_id> <run_id>",
	Aliases: []string{"stop"},
	Short:   "Cancel an in-flight pipeline run",
	Long:    `Terminates or marks failed an active or queued DAG execution run.`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		runID := args[1]

		if !jobsForce && !confirmAction(fmt.Sprintf("Cancel execution run '%s' for pipeline '%s'?", runID, dagID)) {
			ui.PrintInfo("Cancellation aborted.")
			return nil
		}

		c := getClient()
		resp, err := c.CancelPipelineRun(dagID, runID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Cancelled run '%s' for pipeline '%s' (State: %s)", resp.DagRunID, resp.DagID, strings.ToUpper(resp.State))
		})
	},
}

var jobsPauseCmd = &cobra.Command{
	Use:   "pause <job_id>",
	Short: "Pause scheduled execution of a pipeline",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		c := getClient()

		if err := c.SetPipelinePaused(dagID, true); err != nil {
			return err
		}

		ui.PrintSuccess("Pipeline '%s' paused successfully", dagID)
		return nil
	},
}

var jobsUnpauseCmd = &cobra.Command{
	Use:     "unpause <job_id>",
	Aliases: []string{"resume"},
	Short:   "Unpause/resume scheduled execution of a pipeline",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		c := getClient()

		if err := c.SetPipelinePaused(dagID, false); err != nil {
			return err
		}

		ui.PrintSuccess("Pipeline '%s' unpaused successfully", dagID)
		return nil
	},
}

var jobsDeleteCmd = &cobra.Command{
	Use:     "delete <job_id>",
	Aliases: []string{"rm"},
	Short:   "Delete a scheduled notebook pipeline or DAG",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]

		if !jobsForce && !confirmAction(fmt.Sprintf("Permanently delete pipeline '%s'?", dagID)) {
			ui.PrintInfo("Deletion aborted.")
			return nil
		}

		c := getClient()
		if err := c.DeletePipeline(dagID); err != nil {
			return err
		}

		ui.PrintSuccess("Pipeline '%s' deleted successfully", dagID)
		return nil
	},
}

var jobsLogsCmd = &cobra.Command{
	Use:   "logs <job_id> <run_id> <task_id>",
	Short: "Fetch stdout/stderr execution logs for a task instance",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		runID := args[1]
		taskID := args[2]

		c := getClient()
		resp, err := c.GetPipelineTaskLogs(dagID, runID, taskID, jobsTryNumber)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			fmt.Fprint(ui.OutWriter, resp.Content)
			if !strings.HasSuffix(resp.Content, "\n") {
				fmt.Fprintln(ui.OutWriter)
			}
		})
	},
}

// ── Runs Subcommand Hierarchy ───────────────────────────────────────────────

var jobsRunsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List, inspect, and manage historical pipeline runs",
}

var jobsRunsListCmd = &cobra.Command{
	Use:     "list <job_id>",
	Aliases: []string{"ls"},
	Short:   "List past execution runs for a pipeline",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		c := getClient()

		resp, err := c.ListPipelineRuns(dagID, jobsLimit)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp.Runs, func() {
			if len(resp.Runs) == 0 {
				ui.PrintInfo("No runs found for pipeline '%s'.", dagID)
				return
			}

			headers := []string{"RUN ID", "STATE", "EXECUTION DATE", "DURATION"}
			rows := make([][]string, len(resp.Runs))
			for i, r := range resp.Runs {
				durStr := "-"
				if r.DurationSeconds != nil {
					durStr = fmt.Sprintf("%.1fs", *r.DurationSeconds)
				}
				rows[i] = []string{
					r.DagRunID,
					strings.ToUpper(r.State),
					r.ExecutionDate,
					durStr,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total runs: %d", len(resp.Runs))
		})
	},
}

var jobsRunsGetCmd = &cobra.Command{
	Use:   "get <job_id> <run_id>",
	Short: "Inspect granular execution details and task breakdown of a run",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dagID := args[0]
		runID := args[1]
		c := getClient()

		run, err := c.GetPipelineRun(dagID, runID)
		if err != nil {
			return err
		}

		tasksResp, _ := c.ListPipelineTaskInstances(dagID, runID)

		payload := map[string]any{
			"run":   run,
			"tasks": nil,
		}
		if tasksResp != nil {
			payload["tasks"] = tasksResp.Tasks
		}

		return ui.RenderOutput(payload, func() {
			ui.PrintSuccess("Pipeline Run: %s (%s)", run.DagRunID, strings.ToUpper(run.State))
			fmt.Fprintf(ui.OutWriter, "  Pipeline:   %s\n", dagID)
			fmt.Fprintf(ui.OutWriter, "  Executed:   %s\n", run.ExecutionDate)
			if run.StartDate != "" {
				fmt.Fprintf(ui.OutWriter, "  Started:    %s\n", run.StartDate)
			}
			if run.EndDate != "" {
				fmt.Fprintf(ui.OutWriter, "  Ended:      %s\n", run.EndDate)
			}
			if run.DurationSeconds != nil {
				fmt.Fprintf(ui.OutWriter, "  Duration:   %.1fs\n", *run.DurationSeconds)
			}

			if tasksResp != nil && len(tasksResp.Tasks) > 0 {
				fmt.Fprintln(ui.OutWriter, "\nTask Breakdown:")
				headers := []string{"TASK ID", "STATE", "TRY", "DURATION"}
				rows := make([][]string, len(tasksResp.Tasks))
				for i, t := range tasksResp.Tasks {
					durStr := "-"
					if t.DurationSeconds != nil {
						durStr = fmt.Sprintf("%.1fs", *t.DurationSeconds)
					}
					rows[i] = []string{
						t.TaskID,
						strings.ToUpper(t.State),
						fmt.Sprintf("%d", t.TryNumber),
						durStr,
					}
				}
				fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			}
		})
	},
}

var jobsRunsLogsCmd = &cobra.Command{
	Use:   "logs <job_id> <run_id> <task_id>",
	Short: "Stream or fetch stdout/stderr task execution logs",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return jobsLogsCmd.RunE(cmd, args)
	},
}

func init() {
	// run flags
	jobsRunCmd.Flags().BoolVar(&jobsWait, "wait", false, "Wait for run completion before exiting")
	jobsRunCmd.Flags().DurationVar(&jobsTimeout, "timeout", 10*time.Minute, "Timeout when waiting for completion")
	jobsRunCmd.Flags().DurationVar(&jobsPollInterval, "poll-interval", 3*time.Second, "Polling interval when waiting")
	jobsRunCmd.Flags().StringVar(&jobsConf, "conf", "", "JSON parameters to pass to pipeline run")

	// cancel flags
	jobsCancelCmd.Flags().BoolVarP(&jobsForce, "force", "f", false, "Force cancel without confirmation prompt")

	// delete flags
	jobsDeleteCmd.Flags().BoolVarP(&jobsForce, "force", "f", false, "Force delete without confirmation prompt")

	// logs flags
	jobsLogsCmd.Flags().IntVar(&jobsTryNumber, "try", 1, "Task retry attempt number")
	jobsRunsLogsCmd.Flags().IntVar(&jobsTryNumber, "try", 1, "Task retry attempt number")

	// runs list flags
	jobsRunsListCmd.Flags().IntVarP(&jobsLimit, "limit", "l", 25, "Maximum number of past runs to retrieve")

	// Subcommands under jobs runs
	jobsRunsCmd.AddCommand(jobsRunsListCmd)
	jobsRunsCmd.AddCommand(jobsRunsGetCmd)
	jobsRunsCmd.AddCommand(jobsRunsLogsCmd)
	jobsRunsCmd.AddCommand(jobsCancelCmd)

	// Subcommands under jobs
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsRunCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
	jobsCmd.AddCommand(jobsPauseCmd)
	jobsCmd.AddCommand(jobsUnpauseCmd)
	jobsCmd.AddCommand(jobsDeleteCmd)
	jobsCmd.AddCommand(jobsLogsCmd)
	jobsCmd.AddCommand(jobsRunsCmd)

	rootCmd.AddCommand(jobsCmd)
}
