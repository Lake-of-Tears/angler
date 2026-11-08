package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// StatementExecutionRequest matches the schema for POST /api/sql/statements.
type StatementExecutionRequest struct {
	Statement   string        `json:"statement"`
	WarehouseID string        `json:"warehouse_id,omitempty"`
	Catalog     string        `json:"catalog,omitempty"`
	Schema      string        `json:"schema,omitempty"`
	Parameters  []interface{} `json:"parameters,omitempty"`
	RowLimit    int           `json:"row_limit,omitempty"`
	ByteLimit   int           `json:"byte_limit,omitempty"`
	ChunkSize   int           `json:"chunk_size,omitempty"`
	Format      string        `json:"format,omitempty"` // JSON_ARRAY, JSON_OBJECT, CSV
	WaitTimeout string        `json:"wait_timeout,omitempty"`
}

// StatementErrorInfo represents error details in a statement execution.
type StatementErrorInfo struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// StatementStatus represents the execution state of a statement.
type StatementStatus struct {
	State string              `json:"state"` // PENDING, RUNNING, SUCCEEDED, FAILED, CANCELED, CLOSED
	Error *StatementErrorInfo `json:"error,omitempty"`
}

// StatementColumnInfo represents a single column descriptor in schema info.
type StatementColumnInfo struct {
	Name     string `json:"name"`
	TypeText string `json:"type_text"`
	TypeName string `json:"type_name,omitempty"`
	Position int    `json:"position"`
}

// StatementSchemaInfo represents columns metadata in manifest.
type StatementSchemaInfo struct {
	ColumnCount int                   `json:"column_count"`
	Columns     []StatementColumnInfo `json:"columns"`
}

// StatementChunkInfo describes a pagination chunk in the result set.
type StatementChunkInfo struct {
	ChunkIndex int `json:"chunk_index"`
	RowOffset  int `json:"row_offset"`
	RowCount   int `json:"row_count"`
	ByteCount  int `json:"byte_count"`
}

// StatementManifest contains schema and pagination metrics for statement results.
type StatementManifest struct {
	Format          string               `json:"format"`
	Schema          StatementSchemaInfo  `json:"schema"`
	TotalChunkCount int                  `json:"total_chunk_count"`
	Chunks          []StatementChunkInfo `json:"chunks"`
	TotalRowCount   int                  `json:"total_row_count"`
	TotalByteCount  int                  `json:"total_byte_count"`
	Truncated       bool                 `json:"truncated"`
}

// StatementResultChunk holds result data rows or records.
type StatementResultChunk struct {
	ChunkIndex            int                      `json:"chunk_index"`
	RowOffset             int                      `json:"row_offset"`
	RowCount              int                      `json:"row_count"`
	NextChunkIndex        *int                     `json:"next_chunk_index,omitempty"`
	NextChunkInternalLink string                   `json:"next_chunk_internal_link,omitempty"`
	DataArray             [][]interface{}          `json:"data_array,omitempty"`
	DataObjects           []map[string]interface{} `json:"data_objects,omitempty"`
	CSVData               string                   `json:"csv_data,omitempty"`
}

// StatementExecutionResponse represents the response from /api/sql/statements.
type StatementExecutionResponse struct {
	StatementID   string                `json:"statement_id"`
	Status        StatementStatus       `json:"status"`
	Manifest      *StatementManifest    `json:"manifest,omitempty"`
	Result        *StatementResultChunk `json:"result,omitempty"`
	ExecutionMs   *float64              `json:"execution_ms,omitempty"`
	WorkspaceSlug string                `json:"workspace_slug,omitempty"`
	Catalog       string                `json:"catalog,omitempty"`
	Schema        string                `json:"schema,omitempty"`
	CreatedAt     string                `json:"created_at,omitempty"`
}

// AI SQL requests and responses
type SqlExplainRequest struct {
	SQL   string `json:"sql"`
	Model string `json:"model,omitempty"`
}

type SqlExplainResponse struct {
	OK               bool     `json:"ok"`
	Summary          string   `json:"summary,omitempty"`
	Steps            []string `json:"steps,omitempty"`
	TablesReferenced []string `json:"tables_referenced,omitempty"`
	AIFunctionsUsed  []string `json:"ai_functions_used,omitempty"`
	Error            string   `json:"error,omitempty"`
}

type SqlGenerateRequest struct {
	Prompt     string `json:"prompt"`
	CurrentSQL string `json:"current_sql,omitempty"`
	Model      string `json:"model,omitempty"`
}

type SqlGenerateResponse struct {
	OK              bool     `json:"ok"`
	SQL             string   `json:"sql,omitempty"`
	Explanation     string   `json:"explanation,omitempty"`
	TablesUsed      []string `json:"tables_used,omitempty"`
	AIFunctionsUsed []string `json:"ai_functions_used,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type SqlFixRequest struct {
	SQL   string `json:"sql"`
	Error string `json:"error"`
	Model string `json:"model,omitempty"`
}

type SqlFixResponse struct {
	OK          bool   `json:"ok"`
	FixedSQL    string `json:"fixed_sql,omitempty"`
	Explanation string `json:"explanation,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ExecuteStatement submits a SQL query or DDL/DML statement to /api/sql/statements.
func (c *Client) ExecuteStatement(req StatementExecutionRequest) (*StatementExecutionResponse, error) {
	if req.WarehouseID == "" {
		req.WarehouseID = "default"
	}
	if req.Catalog == "" {
		req.Catalog = "main"
	}
	if req.Schema == "" {
		req.Schema = "default"
	}
	if req.RowLimit <= 0 {
		req.RowLimit = 1000
	}
	if req.Format == "" {
		req.Format = "JSON_ARRAY"
	}

	httpReq, err := c.newRequest(http.MethodPost, "/api/sql/statements", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("statement execution failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res StatementExecutionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode statement response: %w", err)
	}

	return &res, nil
}

// ListStatements fetches recent statement executions from /api/sql/statements.
func (c *Client) ListStatements(limit int, state string) ([]StatementExecutionResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	} else {
		params.Set("limit", "50")
	}
	if state != "" {
		params.Set("state", state)
	}

	path := fmt.Sprintf("/api/sql/statements?%s", params.Encode())
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list statements: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list statements (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res []StatementExecutionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode statements list: %w", err)
	}

	return res, nil
}

// GetStatement retrieves a specific statement execution by ID.
func (c *Client) GetStatement(statementID string) (*StatementExecutionResponse, error) {
	path := fmt.Sprintf("/api/sql/statements/%s", url.PathEscape(statementID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get statement %s: %w", statementID, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get statement (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res StatementExecutionResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode statement details: %w", err)
	}

	return &res, nil
}

// CancelStatement cancels an in-flight statement execution.
func (c *Client) CancelStatement(statementID string) error {
	path := fmt.Sprintf("/api/sql/statements/%s/cancel", url.PathEscape(statementID))
	httpReq, err := c.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to cancel statement: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel statement failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ExplainSQL requests a query explanation from /api/ai/sql/explain.
func (c *Client) ExplainSQL(req SqlExplainRequest) (*SqlExplainResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/ai/sql/explain", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("explain SQL failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("explain SQL failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res SqlExplainResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode explain response: %w", err)
	}

	return &res, nil
}

// GenerateSQL translates natural language prompt into SQL via /api/ai/sql/generate.
func (c *Client) GenerateSQL(req SqlGenerateRequest) (*SqlGenerateResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/ai/sql/generate", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("generate SQL failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generate SQL failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res SqlGenerateResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode generate response: %w", err)
	}

	return &res, nil
}

// FixSQL diagnoses and suggests a fix for broken SQL via /api/ai/sql/fix.
func (c *Client) FixSQL(req SqlFixRequest) (*SqlFixResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/ai/sql/fix", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fix SQL failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fix SQL failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res SqlFixResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("failed to decode fix response: %w", err)
	}

	return &res, nil
}
