package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PipelineLatestRun describes the most recently created run of a pipeline.
//
// ExecutionDate is Airflow's logical date: for a scheduled run it is the start of
// the data interval the run covers, not when it ran. Use StartDate for that.
type PipelineLatestRun struct {
	DagRunID          string     `json:"dag_run_id"`
	State             string     `json:"state"`
	RunType           string     `json:"run_type,omitempty"`
	ExecutionDate     *time.Time `json:"execution_date,omitempty"`
	DataIntervalStart *time.Time `json:"data_interval_start,omitempty"`
	DataIntervalEnd   *time.Time `json:"data_interval_end,omitempty"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	DurationSeconds   *float64   `json:"duration_seconds,omitempty"`
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
	NextRunAt           *time.Time         `json:"next_run_at,omitempty"`
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
	WebhookEnabled      bool               `json:"webhook_enabled"`
	WebhookURL          string             `json:"webhook_url,omitempty"`
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
	NextDagrun       *time.Time         `json:"next_dagrun,omitempty"`
	NextRunAt        *time.Time         `json:"next_run_at,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
	IsNotebookJob    bool               `json:"is_notebook_job"`
	NotebookPath     string             `json:"notebook_path,omitempty"`
	Tasks            []map[string]any   `json:"tasks,omitempty"`
	LastRun          *PipelineLatestRun `json:"last_run,omitempty"`
	Settings         map[string]any     `json:"settings,omitempty"`
	WebhookEnabled   bool               `json:"webhook_enabled"`
	WebhookURL       string             `json:"webhook_url,omitempty"`
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
	DagRunID          string         `json:"dag_run_id"`
	State             string         `json:"state"`
	RunType           string         `json:"run_type,omitempty"`
	ExecutionDate     string         `json:"execution_date"`
	DataIntervalStart string         `json:"data_interval_start,omitempty"`
	DataIntervalEnd   string         `json:"data_interval_end,omitempty"`
	StartDate         string         `json:"start_date,omitempty"`
	EndDate           string         `json:"end_date,omitempty"`
	DurationSeconds   *float64       `json:"duration_seconds,omitempty"`
	Conf              map[string]any `json:"conf,omitempty"`
}

// NextFireTime returns when the next scheduled run starts. NextDagrun is the
// next run's logical date, which Airflow schedules one interval later.
func NextFireTime(nextRunAt, nextDagrun *time.Time) *time.Time {
	if nextRunAt != nil {
		return nextRunAt
	}
	return nextDagrun
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

// ── Pipeline Settings, Webhooks, Schedule & Timeline ─────────────────────────

type PipelineSettings struct {
	DagID               string   `json:"dag_id"`
	ColorHex            string   `json:"color_hex"`
	DisplayLabel        string   `json:"display_label"`
	Description         string   `json:"description"`
	IsVisibleOnTimeline bool     `json:"is_visible_on_timeline"`
	WebhookEnabled      bool     `json:"webhook_enabled"`
	WebhookURL          string   `json:"webhook_url"`
	Palette             []string `json:"palette,omitempty"`
}

type PipelineSettingsUpdateRequest struct {
	ColorHex            *string `json:"color_hex,omitempty"`
	DisplayLabel        *string `json:"display_label,omitempty"`
	Description         *string `json:"description,omitempty"`
	IsVisibleOnTimeline *bool   `json:"is_visible_on_timeline,omitempty"`
	WebhookEnabled      *bool   `json:"webhook_enabled,omitempty"`
}

type ScheduleNotebookRequest struct {
	NotebookPath     string            `json:"notebook_path"`
	ScheduleInterval string            `json:"schedule_interval"`
	PipelineTitle    string            `json:"pipeline_title,omitempty"`
	Parameters       map[string]any    `json:"parameters,omitempty"`
	ComputeResources map[string]string `json:"compute_resources,omitempty"`
}

type ScheduleNotebookResponse struct {
	DagID            string `json:"dag_id"`
	PipelineName     string `json:"pipeline_name,omitempty"`
	NotebookPath     string `json:"notebook_path"`
	ScheduleInterval string `json:"schedule_interval"`
	Status           string `json:"status,omitempty"`
	Message          string `json:"message,omitempty"`
}

type PipelineTimelineResponse struct {
	Workspace string           `json:"workspace,omitempty"`
	Date      string           `json:"date"`
	Timezone  string           `json:"timezone"`
	Pipelines []map[string]any `json:"pipelines,omitempty"`
	Events    []map[string]any `json:"events,omitempty"`
}

type PipelineWebhookTriggerResponse struct {
	Status        string `json:"status"`
	DagID         string `json:"dag_id"`
	DagRunID      string `json:"dag_run_id"`
	State         string `json:"state"`
	ExecutionDate string `json:"execution_date,omitempty"`
	WorkspaceSlug string `json:"workspace_slug"`
	EventID       string `json:"event_id"`
}

type WebhookSecretResponse struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceSlug string `json:"workspace_slug"`
	WebhookSecret string `json:"webhook_secret"`
	WebhookURL    string `json:"webhook_url"`
}

type WebhookIngestResponse struct {
	Status            string  `json:"status"`
	EventID           string  `json:"event_id"`
	WorkspaceSlug     string  `json:"workspace_slug"`
	ReceivedAt        string  `json:"received_at"`
	TriggeredPipeline *string `json:"triggered_pipeline,omitempty"`
	DagRunID          *string `json:"dag_run_id,omitempty"`
}

func (c *Client) GetPipelineSettings(dagID string) (*PipelineSettings, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("/api/v1/pipelines/%s/settings", cleanID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get pipeline settings error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res PipelineSettings
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline settings: %w", err)
	}
	return &res, nil
}

func (c *Client) UpdatePipelineSettings(dagID string, reqPayload PipelineSettingsUpdateRequest) (*PipelineSettings, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	req, err := c.newRequest(http.MethodPatch, fmt.Sprintf("/api/v1/pipelines/%s/settings", cleanID), reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update pipeline settings error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res PipelineSettings
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode updated pipeline settings: %w", err)
	}
	if res.WebhookURL == "" {
		ws := c.Workspace
		if ws == "" {
			ws = "default"
		}
		res.WebhookURL = fmt.Sprintf("/api/v1/pipelines/%s/%s/webhook", ws, dagID)
	}
	return &res, nil
}

func (c *Client) UpdatePipelineSchedule(dagID, scheduleInterval string) (map[string]any, error) {
	cleanID := url.PathEscape(strings.TrimSpace(dagID))
	req, err := c.newRequest(
		http.MethodPatch,
		fmt.Sprintf("/api/v1/pipelines/%s/schedule", cleanID),
		map[string]string{"schedule_interval": scheduleInterval},
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update pipeline schedule error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode schedule response: %w", err)
	}
	return res, nil
}

func (c *Client) ScheduleNotebook(reqPayload ScheduleNotebookRequest) (*ScheduleNotebookResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/pipelines/schedule-notebook", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule notebook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("schedule notebook error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res ScheduleNotebookResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode schedule notebook response: %w", err)
	}
	return &res, nil
}

func (c *Client) GetPipelineTimeline(dateStr, tz string) (*PipelineTimelineResponse, error) {
	params := url.Values{}
	if dateStr != "" {
		params.Set("date", dateStr)
	}
	if tz != "" {
		params.Set("tz", tz)
	}

	path := "/api/v1/pipelines/timeline"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline timeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get pipeline timeline error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res PipelineTimelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline timeline: %w", err)
	}
	return &res, nil
}

func computeWebhookHMAC(secret string, rawBody []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) resolveWorkspaceSlug() string {
	if c.Workspace != "" {
		return c.Workspace
	}
	return "default"
}

func (c *Client) TriggerPipelineWebhook(dagID string, rawPayload []byte, secret string, useHMAC bool) (*PipelineWebhookTriggerResponse, error) {
	ws := c.resolveWorkspaceSlug()
	if secret == "" {
		secResp, err := c.GetWebhookSecret(ws)
		if err == nil && secResp != nil {
			secret = secResp.WebhookSecret
		}
	}

	if len(rawPayload) == 0 {
		rawPayload = []byte(`{}`)
	}

	path := fmt.Sprintf("/api/v1/pipelines/%s/%s/webhook", url.PathEscape(ws), url.PathEscape(strings.TrimSpace(dagID)))
	reqURL := fmt.Sprintf("%s%s", c.Endpoint, path)
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(rawPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "angler-cli")
	req.Header.Set("X-Workspace-Slug", ws)

	if secret != "" {
		if useHMAC {
			req.Header.Set("X-Hub-Signature-256", computeWebhookHMAC(secret, rawPayload))
		} else {
			req.Header.Set("X-Lake-Webhook-Secret", secret)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pipeline webhook trigger error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res PipelineWebhookTriggerResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline webhook response: %w", err)
	}
	return &res, nil
}

func (c *Client) doWebhookEndpointRequest(method, subPath string, body any) (*http.Response, error) {
	primary := "/api/v1/webhooks" + subPath
	req, err := c.newRequest(method, primary, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		fallback := "/api/v1/ingest/webhooks" + subPath
		req2, err2 := c.newRequest(method, fallback, body)
		if err2 != nil {
			return nil, err2
		}
		return c.HTTPClient.Do(req2)
	}
	return resp, nil
}

func (c *Client) GetWebhookSecret(workspaceSlug string) (*WebhookSecretResponse, error) {
	if workspaceSlug == "" {
		workspaceSlug = c.resolveWorkspaceSlug()
	}
	resp, err := c.doWebhookEndpointRequest(http.MethodGet, fmt.Sprintf("/%s/secret", url.PathEscape(workspaceSlug)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch webhook secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get webhook secret error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res WebhookSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode webhook secret: %w", err)
	}
	return &res, nil
}

func (c *Client) RegenerateWebhookSecret(workspaceSlug string) (*WebhookSecretResponse, error) {
	if workspaceSlug == "" {
		workspaceSlug = c.resolveWorkspaceSlug()
	}
	resp, err := c.doWebhookEndpointRequest(http.MethodPost, fmt.Sprintf("/%s/secret/regenerate", url.PathEscape(workspaceSlug)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate webhook secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("regenerate webhook secret error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res WebhookSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode regenerated webhook secret: %w", err)
	}
	return &res, nil
}

func (c *Client) IngestWebhookEvent(workspaceSlug string, rawPayload []byte, triggerPipeline, secret string, useHMAC bool) (*WebhookIngestResponse, error) {
	if workspaceSlug == "" {
		workspaceSlug = c.resolveWorkspaceSlug()
	}
	if secret == "" {
		secResp, err := c.GetWebhookSecret(workspaceSlug)
		if err == nil && secResp != nil {
			secret = secResp.WebhookSecret
		}
	}
	if len(rawPayload) == 0 {
		rawPayload = []byte(`{}`)
	}

	buildReq := func(prefix string) (*http.Request, error) {
		relPath := fmt.Sprintf("%s/%s", prefix, url.PathEscape(workspaceSlug))
		if triggerPipeline != "" {
			relPath += "?trigger_pipeline=" + url.QueryEscape(triggerPipeline)
		}
		req, err := http.NewRequest(http.MethodPost, c.Endpoint+relPath, bytes.NewReader(rawPayload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "angler-cli")
		req.Header.Set("X-Workspace-Slug", workspaceSlug)
		if triggerPipeline != "" {
			req.Header.Set("X-Lake-Trigger-Pipeline", triggerPipeline)
		}
		if secret != "" {
			if useHMAC {
				req.Header.Set("X-Hub-Signature-256", computeWebhookHMAC(secret, rawPayload))
			} else {
				req.Header.Set("X-Lake-Webhook-Secret", secret)
			}
		}
		return req, nil
	}

	req, err := buildReq("/api/v1/webhooks")
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send webhook event: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		req2, err2 := buildReq("/api/v1/ingest/webhooks")
		if err2 != nil {
			return nil, err2
		}
		resp, err = c.HTTPClient.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("failed to send webhook event: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("webhook ingest error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res WebhookIngestResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode webhook ingest response: %w", err)
	}
	return &res, nil
}
