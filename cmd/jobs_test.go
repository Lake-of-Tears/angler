package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetJobsFlags() {
	jobsWait = false
	jobsForce = false
	jobsConf = ""
	jobsLimit = 25
	jobsTryNumber = 1
	jobsTimeout = 10 * time.Minute
	jobsPollInterval = 50 * time.Millisecond
	jsonOutput = false
	outputFlag = ""
}

func TestJobsCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundJobs bool
	for _, c := range commands {
		if c.Name() == "jobs" {
			foundJobs = true
			expectedSubcommands := map[string]bool{
				"list":    false,
				"get":     false,
				"run":     false,
				"cancel":  false,
				"pause":   false,
				"unpause": false,
				"delete":  false,
				"logs":    false,
				"runs":    false,
			}
			for _, sub := range c.Commands() {
				if _, ok := expectedSubcommands[sub.Name()]; ok {
					expectedSubcommands[sub.Name()] = true
				}
			}
			for sub, found := range expectedSubcommands {
				if !found {
					t.Errorf("expected subcommand %q registered under jobs", sub)
				}
			}
		}
	}
	if !foundJobs {
		t.Fatal("jobs command not registered on rootCmd")
	}
}

func setupJobsMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	now := time.Now()
	dur := 14.2

	// List pipelines: /api/pipelines
	mux.HandleFunc("/api/pipelines", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineListResponse{
			Workspace: "lake-ws",
			Total:     1,
			Pipelines: []client.Pipeline{
				{
					DagID:            "telemetry_etl",
					PipelineName:     "Telemetry ETL Pipeline",
					DisplayLabel:     "Telemetry ETL Pipeline",
					ScheduleInterval: "*/15 * * * *",
					IsPaused:         false,
					Tags:             []string{"production", "gold"},
					LastRun: &client.PipelineLatestRun{
						DagRunID:        "scheduled__20261108T120000Z",
						State:           "success",
						StartDate:       &now,
						DurationSeconds: &dur,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Get pipeline: /api/pipelines/telemetry_etl
	mux.HandleFunc("/api/pipelines/telemetry_etl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			resp := client.PipelineDetail{
				DagID:            "telemetry_etl",
				PipelineName:     "Telemetry ETL Pipeline",
				Description:      "Extracts and summarizes telemetry events into gold lakehouse tables",
				ScheduleInterval: "*/15 * * * *",
				IsPaused:         false,
				Tags:             []string{"production", "gold"},
				Tasks: []map[string]any{
					{"task_id": "extract", "operator": "KubernetesPodOperator"},
					{"task_id": "transform", "operator": "DuckDBOperator"},
				},
				LastRun: &client.PipelineLatestRun{
					DagRunID:        "scheduled__20261108T120000Z",
					State:           "success",
					StartDate:       &now,
					DurationSeconds: &dur,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"is_paused": true})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Trigger run: /api/pipelines/telemetry_etl/trigger
	mux.HandleFunc("/api/pipelines/telemetry_etl/trigger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := client.PipelineTriggerResponse{
			DagID:         "telemetry_etl",
			DagRunID:      "manual__20261108T160000Z__user",
			State:         "queued",
			ExecutionDate: "2026-11-08T16:00:00Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// List runs: /api/pipelines/telemetry_etl/runs
	mux.HandleFunc("/api/pipelines/telemetry_etl/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineRunsResponse{
			DagID:     "telemetry_etl",
			Workspace: "lake-ws",
			Total:     1,
			Runs: []client.PipelineRun{
				{
					DagRunID:        "manual__20261108T160000Z__user",
					State:           "success",
					ExecutionDate:   "2026-11-08T16:00:00Z",
					StartDate:       "2026-11-08T16:00:01Z",
					EndDate:         "2026-11-08T16:00:15Z",
					DurationSeconds: &dur,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Single run: /api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user
	pollCount := 0
	mux.HandleFunc("/api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pollCount++
		state := "running"
		if pollCount >= 2 {
			state = "success"
		}
		resp := client.PipelineRun{
			DagRunID:        "manual__20261108T160000Z__user",
			State:           state,
			ExecutionDate:   "2026-11-08T16:00:00Z",
			StartDate:       "2026-11-08T16:00:01Z",
			EndDate:         "2026-11-08T16:00:15Z",
			DurationSeconds: &dur,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Run tasks: /api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/tasks
	mux.HandleFunc("/api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineTasksResponse{
			DagID: "telemetry_etl",
			RunID: "manual__20261108T160000Z__user",
			Tasks: []client.PipelineTaskInstance{
				{
					TaskID:          "extract",
					State:           "success",
					TryNumber:       1,
					DurationSeconds: &dur,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Run cancel: /api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/cancel
	mux.HandleFunc("/api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineCancelResponse{
			DagID:    "telemetry_etl",
			DagRunID: "manual__20261108T160000Z__user",
			State:    "failed",
			Message:  "DAG run cancelled successfully",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Logs: /api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/tasks/extract/logs
	mux.HandleFunc("/api/pipelines/telemetry_etl/runs/manual__20261108T160000Z__user/tasks/extract/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.PipelineTaskLogResponse{
			DagID:     "telemetry_etl",
			RunID:     "manual__20261108T160000Z__user",
			TaskID:    "extract",
			TryNumber: 1,
			Content:   "[2026-11-08 16:00:02] Running extraction pipeline\n[2026-11-08 16:00:14] 15,000 events committed to bronze lakehouse",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestJobsListAndGet(t *testing.T) {
	ts := setupJobsMockServer(t)
	defer ts.Close()

	// 1. List pipelines in table format
	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "telemetry_etl") {
		t.Errorf("expected telemetry_etl in jobs list: %s", stdout)
	}
	if !strings.Contains(stdout, "Telemetry ETL Pipeline") {
		t.Errorf("expected name in jobs list: %s", stdout)
	}

	// 2. List pipelines in JSON format
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "jobs", "ls", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var pipes []client.Pipeline
	if err := json.Unmarshal([]byte(stdout), &pipes); err != nil {
		t.Fatalf("failed to parse JSON pipelines: %v\nstdout: %s", err, stdout)
	}
	if len(pipes) != 1 || pipes[0].DagID != "telemetry_etl" {
		t.Errorf("unexpected pipelines JSON payload: %+v", pipes)
	}

	// 3. Get pipeline details
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "get", "telemetry_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Pipeline: Telemetry ETL Pipeline") {
		t.Errorf("expected title in get output: %s", stdout)
	}
	if !strings.Contains(stdout, "extract") {
		t.Errorf("expected extract task in get output: %s", stdout)
	}
}

func TestJobsRunAndRuns(t *testing.T) {
	ts := setupJobsMockServer(t)
	defer ts.Close()

	// 1. Trigger run immediately without waiting
	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "run", "telemetry_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Triggered run 'manual__20261108T160000Z__user'") {
		t.Errorf("expected run confirmation: %s", stdout)
	}

	// 2. List runs
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "runs", "list", "telemetry_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "manual__20261108T160000Z__user") {
		t.Errorf("expected run id in runs list: %s", stdout)
	}

	// 3. Get run details
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "runs", "get", "telemetry_etl", "manual__20261108T160000Z__user", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Pipeline Run: manual__20261108T160000Z__user") {
		t.Errorf("expected run title: %s", stdout)
	}
	if !strings.Contains(stdout, "extract") {
		t.Errorf("expected task instance extract in run breakdown: %s", stdout)
	}
}

func TestJobsRunWait(t *testing.T) {
	ts := setupJobsMockServer(t)
	defer ts.Close()

	// Trigger run with --wait
	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "run", "telemetry_etl", "--wait", "--timeout", "5s", "--poll-interval", "10ms", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Pipeline run completed") {
		t.Errorf("expected completion message: %s", stdout)
	}
	if !strings.Contains(stdout, "extract") {
		t.Errorf("expected task table in wait output: %s", stdout)
	}
}

func TestJobsLogsAndCancel(t *testing.T) {
	ts := setupJobsMockServer(t)
	defer ts.Close()

	// 1. Fetch task logs
	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "logs", "telemetry_etl", "manual__20261108T160000Z__user", "extract", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "15,000 events committed to bronze lakehouse") {
		t.Errorf("expected log content in output: %s", stdout)
	}

	// 2. Cancel run
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "cancel", "telemetry_etl", "manual__20261108T160000Z__user", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Cancelled run") {
		t.Errorf("expected cancel confirmation: %s", stdout)
	}
}

func TestJobsPauseAndUnpauseAndDelete(t *testing.T) {
	ts := setupJobsMockServer(t)
	defer ts.Close()

	// 1. Pause
	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "pause", "telemetry_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "paused successfully") {
		t.Errorf("expected pause success: %s", stdout)
	}

	// 2. Unpause
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "unpause", "telemetry_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "unpaused successfully") {
		t.Errorf("expected unpause success: %s", stdout)
	}

	// 3. Delete
	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "delete", "telemetry_etl", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "deleted successfully") {
		t.Errorf("expected delete success: %s", stdout)
	}
}

// TestJobsShowsWallClockRunTimes covers #479: a scheduled run's logical date is one
// interval before it fires, so `jobs get` and `jobs runs list` must show the fire
// time and actual start time rather than the logical date.
func TestJobsShowsWallClockRunTimes(t *testing.T) {
	logical := time.Date(2026, 11, 11, 19, 0, 0, 0, time.UTC)
	fired := time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC)
	nextLogical := time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC)
	nextFire := time.Date(2026, 11, 12, 7, 0, 0, 0, time.UTC)
	dur := 91.9

	mux := http.NewServeMux()
	mux.HandleFunc("/api/pipelines/email_etl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.PipelineDetail{
			DagID:            "email_etl",
			PipelineName:     "Email ETL",
			ScheduleInterval: "0 1,7,13,19 * * *",
			NextDagrun:       &nextLogical,
			NextRunAt:        &nextFire,
			LastRun: &client.PipelineLatestRun{
				DagRunID:          "scheduled__2026-11-11T19:00:00+00:00",
				State:             "success",
				RunType:           "scheduled",
				ExecutionDate:     &logical,
				DataIntervalStart: &logical,
				DataIntervalEnd:   &fired,
				StartDate:         &fired,
				DurationSeconds:   &dur,
			},
		})
	})
	mux.HandleFunc("/api/pipelines/email_etl/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.PipelineRunsResponse{
			DagID: "email_etl",
			Total: 1,
			Runs: []client.PipelineRun{{
				DagRunID:          "scheduled__2026-11-11T19:00:00+00:00",
				State:             "success",
				RunType:           "scheduled",
				ExecutionDate:     "2026-11-11T19:00:00+00:00",
				DataIntervalStart: "2026-11-11T19:00:00+00:00",
				DataIntervalEnd:   "2026-11-12T01:00:00+00:00",
				StartDate:         "2026-11-12T01:00:00.532654+00:00",
				DurationSeconds:   &dur,
			}},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	// The --json call below sets the global jsonOutput flag; clear it for later tests.
	t.Cleanup(resetJobsFlags)

	resetJobsFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "jobs", "get", "email_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Next Run:    2026-11-12 07:00:00 UTC") {
		t.Errorf("expected next fire time, not logical date, in get output: %s", stdout)
	}
	if !strings.Contains(stdout, "Covers:      2026-11-11 19:00:00 → 2026-11-12 01:00:00") {
		t.Errorf("expected data interval for scheduled last run: %s", stdout)
	}

	resetJobsFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "jobs", "runs", "list", "email_etl", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "STARTED") || !strings.Contains(stdout, "2026-11-12T01:00:00.532654+00:00") {
		t.Errorf("expected STARTED column with actual start time: %s", stdout)
	}

	resetJobsFlags()
	stdout, _, _ = executeCLIForTest(t, false, "jobs", "get", "email_etl", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	var detail client.PipelineDetail
	if err := json.Unmarshal([]byte(stdout), &detail); err != nil {
		t.Fatalf("failed to parse JSON detail: %v\nstdout: %s", err, stdout)
	}
	if detail.NextRunAt == nil || !detail.NextRunAt.Equal(nextFire) {
		t.Errorf("expected next_run_at to survive JSON output, got %+v", detail.NextRunAt)
	}
	if detail.LastRun == nil || detail.LastRun.DataIntervalEnd == nil || detail.LastRun.RunType != "scheduled" {
		t.Errorf("expected run_type and data interval on last_run, got %+v", detail.LastRun)
	}
}
