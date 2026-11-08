package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestPipelineClientMethods(t *testing.T) {
	mux := http.NewServeMux()

	now := time.Now()
	dur := 12.5

	// 1. /api/pipelines
	mux.HandleFunc("/api/pipelines", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := PipelineListResponse{
			Workspace: "lake-ws",
			Total:     1,
			Pipelines: []Pipeline{
				{
					DagID:            "demo_pipeline",
					PipelineName:     "Demo Pipeline",
					DisplayLabel:     "Demo Pipeline",
					ScheduleInterval: "0 * * * *",
					IsPaused:         false,
					Tags:             []string{"demo", "lakehouse"},
					LastRun: &PipelineLatestRun{
						DagRunID:        "run_123",
						State:           "success",
						StartDate:       &now,
						DurationSeconds: &dur,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 2. /api/pipelines/demo_pipeline
	mux.HandleFunc("/api/pipelines/demo_pipeline", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			resp := PipelineDetail{
				DagID:            "demo_pipeline",
				PipelineName:     "Demo Pipeline",
				ScheduleInterval: "0 * * * *",
				IsPaused:         false,
				Tasks: []map[string]any{
					{"task_id": "extract"},
					{"task_id": "transform"},
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

	// 3. /api/pipelines/demo_pipeline/trigger
	mux.HandleFunc("/api/pipelines/demo_pipeline/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := PipelineTriggerResponse{
			DagID:         "demo_pipeline",
			DagRunID:      "manual__20261108T120000Z__user",
			State:         "queued",
			ExecutionDate: "2026-11-08T12:00:00Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 4. /api/pipelines/demo_pipeline/runs
	mux.HandleFunc("/api/pipelines/demo_pipeline/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := PipelineRunsResponse{
			DagID:     "demo_pipeline",
			Workspace: "lake-ws",
			Total:     1,
			Runs: []PipelineRun{
				{
					DagRunID:        "run_123",
					State:           "success",
					ExecutionDate:   "2026-11-08T12:00:00Z",
					StartDate:       "2026-11-08T12:00:01Z",
					EndDate:         "2026-11-08T12:00:15Z",
					DurationSeconds: &dur,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 5. /api/pipelines/demo_pipeline/runs/run_123
	mux.HandleFunc("/api/pipelines/demo_pipeline/runs/run_123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := PipelineRun{
			DagRunID:        "run_123",
			State:           "success",
			ExecutionDate:   "2026-11-08T12:00:00Z",
			StartDate:       "2026-11-08T12:00:01Z",
			EndDate:         "2026-11-08T12:00:15Z",
			DurationSeconds: &dur,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 6. /api/pipelines/demo_pipeline/runs/run_123/tasks
	mux.HandleFunc("/api/pipelines/demo_pipeline/runs/run_123/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		taskDur := 6.0
		resp := PipelineTasksResponse{
			DagID: "demo_pipeline",
			RunID: "run_123",
			Tasks: []PipelineTaskInstance{
				{
					TaskID:          "extract",
					State:           "success",
					TryNumber:       1,
					DurationSeconds: &taskDur,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 7. /api/pipelines/demo_pipeline/runs/run_123/tasks/extract/logs
	mux.HandleFunc("/api/pipelines/demo_pipeline/runs/run_123/tasks/extract/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := PipelineTaskLogResponse{
			DagID:     "demo_pipeline",
			RunID:     "run_123",
			TaskID:    "extract",
			TryNumber: 1,
			Content:   "[2026-11-08 12:00:01] INFO - Extracting data from source\n[2026-11-08 12:00:07] INFO - Extraction complete",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 8. /api/pipelines/demo_pipeline/runs/run_123/cancel
	mux.HandleFunc("/api/pipelines/demo_pipeline/runs/run_123/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := PipelineCancelResponse{
			DagID:    "demo_pipeline",
			DagRunID: "run_123",
			State:    "failed",
			Message:  "DAG run cancelled successfully",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint:  ts.URL,
		Token:     "test-token",
		Workspace: "lake-ws",
	})

	// 1. ListPipelines
	listRes, err := c.ListPipelines()
	if err != nil {
		t.Fatalf("ListPipelines error: %v", err)
	}
	if len(listRes.Pipelines) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(listRes.Pipelines))
	}

	// 2. GetPipeline
	p, err := c.GetPipeline("demo_pipeline")
	if err != nil {
		t.Fatalf("GetPipeline error: %v", err)
	}
	if p.DagID != "demo_pipeline" {
		t.Errorf("expected demo_pipeline, got %s", p.DagID)
	}

	// 3. TriggerPipeline
	trig, err := c.TriggerPipeline("demo_pipeline", map[string]any{"env": "test"})
	if err != nil {
		t.Fatalf("TriggerPipeline error: %v", err)
	}
	if trig.State != "queued" {
		t.Errorf("expected queued, got %s", trig.State)
	}

	// 4. ListPipelineRuns
	runs, err := c.ListPipelineRuns("demo_pipeline", 10)
	if err != nil {
		t.Fatalf("ListPipelineRuns error: %v", err)
	}
	if len(runs.Runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs.Runs))
	}

	// 5. GetPipelineRun
	run, err := c.GetPipelineRun("demo_pipeline", "run_123")
	if err != nil {
		t.Fatalf("GetPipelineRun error: %v", err)
	}
	if run.DagRunID != "run_123" {
		t.Errorf("expected run_123, got %s", run.DagRunID)
	}

	// 6. ListPipelineTaskInstances
	tasks, err := c.ListPipelineTaskInstances("demo_pipeline", "run_123")
	if err != nil {
		t.Fatalf("ListPipelineTaskInstances error: %v", err)
	}
	if len(tasks.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks.Tasks))
	}

	// 7. GetPipelineTaskLogs
	logs, err := c.GetPipelineTaskLogs("demo_pipeline", "run_123", "extract", 1)
	if err != nil {
		t.Fatalf("GetPipelineTaskLogs error: %v", err)
	}
	if !strings.Contains(logs.Content, "Extraction complete") {
		t.Errorf("expected Extraction complete in logs: %s", logs.Content)
	}

	// 8. CancelPipelineRun
	cancelRes, err := c.CancelPipelineRun("demo_pipeline", "run_123")
	if err != nil {
		t.Fatalf("CancelPipelineRun error: %v", err)
	}
	if cancelRes.State != "failed" {
		t.Errorf("expected failed state, got %s", cancelRes.State)
	}

	// 9. SetPipelinePaused
	if err := c.SetPipelinePaused("demo_pipeline", true); err != nil {
		t.Fatalf("SetPipelinePaused error: %v", err)
	}

	// 10. DeletePipeline
	if err := c.DeletePipeline("demo_pipeline"); err != nil {
		t.Fatalf("DeletePipeline error: %v", err)
	}
}
