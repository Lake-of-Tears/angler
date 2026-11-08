package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PipelineLatestRun describes the most recent run of a pipeline.
type PipelineLatestRun struct {
	DagRunID        string     `json:"dag_run_id"`
	State           string     `json:"state"`
	ExecutionDate   *time.Time `json:"execution_date,omitempty"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	DurationSeconds *float64   `json:"duration_seconds,omitempty"`
}

// Pipeline describes an individual workflow DAG, pipeline, or scheduled job.
type Pipeline struct {
	DagID               string             `json:"dag_id"`
	PipelineName        string             `json:"pipeline_name"`
	DisplayLabel        string             `json:"display_label,omitempty"`
	Description         string             `json:"description,omitempty"`
	ColorHex            string             `json:"color_hex,omitempty"`
	IsPaused            bool               `json:"is_paused"`
	ScheduleInterval    string             `json:"schedule_interval"`
	NextDagrun          *time.Time         `json:"next_dagrun,omitempty"`
	Tags                []string           `json:"tags,omitempty"`
	IsNotebookJob       bool               `json:"is_notebook_job"`
	NotebookPath        string             `json:"notebook_path,omitempty"`
	IsBundlePipeline    bool               `json:"is_bundle_pipeline"`
	BundleName          string             `json:"bundle_name,omitempty"`
	BundleID            string             `json:"bundle_id,omitempty"`
	BundleTarget        string             `json:"bundle_target,omitempty"`
	BundleVersion       string             `json:"bundle_version,omitempty"`
	LastRun             *PipelineLatestRun `json:"last_run,omitempty"`
	IsVisibleOnTimeline bool               `json:"is_visible_on_timeline,omitempty"`
}

// PipelineListResponse represents the pipeline collection returned by /api/pipelines.
type PipelineListResponse struct {
	Workspace string     `json:"workspace"`
	Total     int        `json:"total"`
	Pipelines []Pipeline `json:"pipelines"`
}

// PipelineDetail provides deep configuration and metadata for a single pipeline.
type PipelineDetail struct {
	DagID            string             `json:"dag_id"`
	PipelineName     string             `json:"pipeline_name"`
	DisplayLabel     string             `json:"display_label,omitempty"`
	Description      string             `json:"description,omitempty"`
	IsPaused         bool               `json:"is_paused"`
	ScheduleInterval string             `json:"schedule_interval"`
	Tags             []string           `json:"tags,omitempty"`
	IsNotebookJob    bool               `json:"is_notebook_job"`
	NotebookPath     string             `json:"notebook_path,omitempty"`
	Tasks            []map[string]any   `json:"tasks,omitempty"`
	LastRun          *PipelineLatestRun `json:"last_run,omitempty"`
	Settings         map[string]any     `json:"settings,omitempty"`
}

// PipelineTriggerResponse represents the response when initiating a run.
type PipelineTriggerResponse struct {
	DagID         string         `json:"dag_id"`
	DagRunID      string         `json:"dag_run_id"`
	State         string         `json:"state"`
	ExecutionDate string         `json:"execution_date"`
	Conf          map[string]any `json:"conf,omitempty"`
}

// PipelineRun describes a past or in-flight execution run.
type PipelineRun struct {
	DagRunID        string         `json:"dag_run_id"`
	State           string         `json:"state"`
	ExecutionDate   string         `json:"execution_date"`
	StartDate       string         `json:"start_date,omitempty"`
	EndDate         string         `json:"end_date,omitempty"`
	DurationSeconds *float64       `json:"duration_seconds,omitempty"`
	Conf            map[string]any `json:"conf,omitempty"`
}

// PipelineRunsResponse wraps the list of execution runs.
type PipelineRunsResponse struct {
	DagID     string        `json:"dag_id"`
	Workspace string        `json:"workspace"`
	Total     int           `json:"total"`
	Runs      []PipelineRun `json:"runs"`
}

// PipelineTaskInstance describes a single task instance within a DAG run.
type PipelineTaskInstance struct {
	TaskID          string   `json:"task_id"`
	State           string   `json:"state"`
	TryNumber       int      `json:"try_number"`
	StartDate       string   `json:"start_date,omitempty"`
	EndDate         string   `json:"end_date,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
}

// PipelineTasksResponse wraps task instances for an execution run.
type PipelineTasksResponse struct {
	DagID string                 `json:"dag_id"`
	RunID string                 `json:"run_id"`
	Tasks []PipelineTaskInstance `json:"tasks"`
}

// PipelineTaskLogResponse encapsulates task execution stdout/stderr logs.
type PipelineTaskLogResponse struct {
	DagID     string `json:"dag_id"`
	RunID     string `json:"run_id"`
	TaskID    string `json:"task_id"`
	TryNumber int    `json:"try_number"`
	Content   string `json:"content"`
}

// PipelineCancelResponse represents the response when canceling an in-flight run.
type PipelineCancelResponse struct {
	DagID    string `json:"dag_id"`
	DagRunID string `json:"dag_run_id"`
	State    string `json:"state"`
	Message  string `json:"message"`
}

// ListPipelines retrieves all active and scheduled pipelines in the workspace.
func (c *Client) ListPipelines() (*PipelineListResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/pipelines", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list pipelines request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list pipelines request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse pipelines response: %w", err)
	}
	return &res, nil
}

// GetPipeline retrieves detailed configuration and status for a pipeline.
func (c *Client) GetPipeline(dagID string) (*PipelineDetail, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	relPath := fmt.Sprintf("/api/pipelines/%s", cleanID)

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get pipeline request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get pipeline request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineDetail
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline details response: %w", err)
	}
	return &res, nil
}

// TriggerPipeline triggers immediate on-demand execution of a workspace pipeline.
func (c *Client) TriggerPipeline(dagID string, conf map[string]any) (*PipelineTriggerResponse, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	relPath := fmt.Sprintf("/api/pipelines/%s/trigger", cleanID)

	payload := map[string]any{
		"conf": conf,
	}

	req, err := c.newRequest(http.MethodPost, relPath, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create trigger pipeline request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute trigger pipeline request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineTriggerResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse trigger response: %w", err)
	}
	return &res, nil
}

// ListPipelineRuns lists past and active runs for a pipeline.
func (c *Client) ListPipelineRuns(dagID string, limit int) (*PipelineRunsResponse, error) {
	if limit <= 0 {
		limit = 25
	}
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	relPath := fmt.Sprintf("/api/pipelines/%s/runs?limit=%d", cleanID, limit)

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list pipeline runs request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list pipeline runs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline runs response: %w", err)
	}
	return &res, nil
}

// GetPipelineRun retrieves details for a single execution run.
func (c *Client) GetPipelineRun(dagID, runID string) (*PipelineRun, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	cleanRun := url.PathEscape(strings.TrimSpace(runID))
	relPath := fmt.Sprintf("/api/pipelines/%s/runs/%s", cleanID, cleanRun)

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get pipeline run request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get pipeline run request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineRun
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline run response: %w", err)
	}
	return &res, nil
}

// ListPipelineTaskInstances returns all task instances for a specific run.
func (c *Client) ListPipelineTaskInstances(dagID, runID string) (*PipelineTasksResponse, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	cleanRun := url.PathEscape(strings.TrimSpace(runID))
	relPath := fmt.Sprintf("/api/pipelines/%s/runs/%s/tasks", cleanID, cleanRun)

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list task instances request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list task instances request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse task instances response: %w", err)
	}
	return &res, nil
}

// GetPipelineTaskLogs retrieves task execution stdout/stderr logs.
func (c *Client) GetPipelineTaskLogs(dagID, runID, taskID string, tryNumber int) (*PipelineTaskLogResponse, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	cleanRun := url.PathEscape(strings.TrimSpace(runID))
	cleanTask := url.PathEscape(strings.TrimSpace(taskID))
	relPath := fmt.Sprintf("/api/pipelines/%s/runs/%s/tasks/%s/logs", cleanID, cleanRun, cleanTask)
	if tryNumber > 0 {
		relPath = fmt.Sprintf("%s?try_number=%d", relPath, tryNumber)
	}

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get task logs request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get task logs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineTaskLogResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse task logs response: %w", err)
	}
	return &res, nil
}

// CancelPipelineRun terminates or marks failed an active pipeline execution run.
func (c *Client) CancelPipelineRun(dagID, runID string) (*PipelineCancelResponse, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	cleanRun := url.PathEscape(strings.TrimSpace(runID))
	relPath := fmt.Sprintf("/api/pipelines/%s/runs/%s/cancel", cleanID, cleanRun)

	req, err := c.newRequest(http.MethodPost, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cancel pipeline run request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute cancel pipeline run request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res PipelineCancelResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse cancel response: %w", err)
	}
	return &res, nil
}

// SetPipelinePaused pauses or unpauses a pipeline.
func (c *Client) SetPipelinePaused(dagID string, isPaused bool) error {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	relPath := fmt.Sprintf("/api/pipelines/%s", cleanID)

	payload := map[string]bool{
		"is_paused": isPaused,
	}

	req, err := c.newRequest(http.MethodPatch, relPath, payload)
	if err != nil {
		return fmt.Errorf("failed to create pause pipeline request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute pause pipeline request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeletePipeline deletes a pipeline and unprovisions its dynamic DAG code.
func (c *Client) DeletePipeline(dagID string) error {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	relPath := fmt.Sprintf("/api/pipelines/%s", cleanID)

	req, err := c.newRequest(http.MethodDelete, relPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete pipeline request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete pipeline request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
