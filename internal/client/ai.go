package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GenieChatMessage represents a conversational message in the chat history.
type GenieChatMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// GenieChatRequest configures a query to Lake of Tears Genie AI.
type GenieChatRequest struct {
	Question      string             `json:"question"`
	Mode          string             `json:"mode,omitempty"` // "auto", "sql", "rag", "clarify"
	History       []GenieChatMessage `json:"history,omitempty"`
	Model         string             `json:"model,omitempty"`
	RequestID     string             `json:"request_id,omitempty"`
	WorkspaceSlug string             `json:"workspace_slug,omitempty"`
}

// GenieChatResponse encapsulates the response from Genie AI.
type GenieChatResponse struct {
	OK           bool             `json:"ok"`
	Mode         string           `json:"mode,omitempty"`
	Answer       string           `json:"answer,omitempty"`
	SQL          string           `json:"sql,omitempty"`
	Columns      []string         `json:"columns,omitempty"`
	Data         []map[string]any `json:"data,omitempty"`
	Summary      string           `json:"summary,omitempty"`
	GroundedDocs []string         `json:"grounded_docs,omitempty"`
	Error        string           `json:"error,omitempty"`
}

// AIContextResponse represents the system data context exposed to the internal AI layer.
type AIContextResponse struct {
	OK              bool             `json:"ok"`
	WorkspaceID     string           `json:"workspace_id"`
	WorkspaceSlug   string           `json:"workspace_slug"`
	WorkspaceName   string           `json:"workspace_name"`
	Role            string           `json:"role"`
	Catalogs        []map[string]any `json:"catalogs"`
	DefaultDatasets []map[string]any `json:"default_datasets"`
	SQLAIFunctions  []map[string]any `json:"sql_ai_functions"`
}

// TestGeminiKeyRequest configures testing of LLM provider credentials.
type TestGeminiKeyRequest struct {
	APIKey string `json:"api_key,omitempty"`
	Model  string `json:"model,omitempty"`
}

// TestGeminiKeyResponse represents verification results for an LLM provider key.
type TestGeminiKeyResponse struct {
	OK      bool   `json:"ok"`
	Model   string `json:"model"`
	Message string `json:"message"`
}

// VectorIndexSummary describes an active vector embedding index for a catalog table.
type VectorIndexSummary struct {
	CatalogID    string `json:"catalog_id"`
	SchemaID     string `json:"schema_id"`
	TableID      string `json:"table_id"`
	TableName    string `json:"table_name"`
	SourceColumn string `json:"source_column,omitempty"`
	TotalVectors int    `json:"total_vectors"`
	Status       string `json:"status,omitempty"`
	S3VectorPath string `json:"s3_vector_path,omitempty"`
}

// GenieChat sends a single-turn or multi-turn conversational question to Lake of Tears Genie AI.
func (c *Client) GenieChat(req GenieChatRequest) (*GenieChatResponse, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/ai/chat", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res GenieChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse chat response: %w", err)
	}
	return &res, nil
}

// GenieChatStream streams thinking steps and final results for a Genie query via Server-Sent Events (SSE).
func (c *Client) GenieChatStream(req GenieChatRequest, handler func(event string, raw []byte) error) error {
	httpReq, err := c.newRequest(http.MethodPost, "/api/ai/chat/stream", req)
	if err != nil {
		return fmt.Errorf("failed to create streaming chat request: %w", err)
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute streaming chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string
	var currentData bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataPart := strings.TrimPrefix(line, "data:")
			if currentData.Len() > 0 {
				currentData.WriteByte('\n')
			}
			currentData.WriteString(strings.TrimSpace(dataPart))
		} else if line == "" {
			// Dispatch event
			if currentData.Len() > 0 {
				evt := currentEvent
				if evt == "" {
					evt = "message"
				}
				if err := handler(evt, currentData.Bytes()); err != nil {
					return err
				}
				currentEvent = ""
				currentData.Reset()
			}
		}
	}

	if currentData.Len() > 0 {
		evt := currentEvent
		if evt == "" {
			evt = "message"
		}
		if err := handler(evt, currentData.Bytes()); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// GetAIContext fetches the active system data context, grounding prompt, and schemas.
func (c *Client) GetAIContext() (*AIContextResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/ai/context", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ai context request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ai context request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res AIContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse ai context response: %w", err)
	}
	return &res, nil
}

// TestGeminiKey validates configured LLM provider credentials.
func (c *Client) TestGeminiKey(apiKey, model string) (*TestGeminiKeyResponse, error) {
	payload := TestGeminiKeyRequest{
		APIKey: apiKey,
		Model:  model,
	}

	req, err := c.newRequest(http.MethodPost, "/api/admin/settings/test-gemini-key", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create test key request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute test key request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res TestGeminiKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse test key response: %w", err)
	}
	return &res, nil
}

// ListVectorIndexes lists all tables across workspace catalogs that have active vector indexes.
func (c *Client) ListVectorIndexes() ([]VectorIndexSummary, error) {
	catalogs, err := c.ListCatalogs()
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	var summaries []VectorIndexSummary
	for _, cat := range catalogs {
		schemas, err := c.ListSchemas(cat.ID)
		if err != nil {
			continue
		}
		for _, schema := range schemas {
			tables, err := c.ListTables(cat.ID, schema.ID)
			if err != nil {
				continue
			}
			for _, tbl := range tables {
				if tbl.VectorIndex != nil && len(tbl.VectorIndex) > 0 {
					sourceCol, _ := tbl.VectorIndex["source_column"].(string)
					totalVecs := 0
					if tv, ok := tbl.VectorIndex["total_vectors"].(float64); ok {
						totalVecs = int(tv)
					}
					status, _ := tbl.VectorIndex["status"].(string)
					s3Path, _ := tbl.VectorIndex["s3_vector_path"].(string)

					summaries = append(summaries, VectorIndexSummary{
						CatalogID:    cat.ID,
						SchemaID:     schema.ID,
						TableID:      tbl.ID,
						TableName:    tbl.Name,
						SourceColumn: sourceCol,
						TotalVectors: totalVecs,
						Status:       status,
						S3VectorPath: s3Path,
					})
				}
			}
		}
	}

	return summaries, nil
}

// SyncTableVectorIndex triggers an incremental or full vector index synchronization for a table.
func (c *Client) SyncTableVectorIndex(catalogID, schemaID, tableID string) error {
	relPath := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables/%s/vector-index/sync",
		url.PathEscape(catalogID), url.PathEscape(schemaID), url.PathEscape(tableID))

	req, err := c.newRequest(http.MethodPost, relPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create vector sync request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute vector sync request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
