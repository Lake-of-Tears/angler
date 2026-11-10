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

// ── Permission-Aware Vector Retrieval, OpenAI Gateway & AI Table Enrichment ──

type VectorRetrieveRequest struct {
	Query          string   `json:"query"`
	TopK           int      `json:"top_k,omitempty"`
	ScoreThreshold *float64 `json:"score_threshold,omitempty"`
	WorkspaceSlug  string   `json:"workspace_slug,omitempty"`
	Catalog        string   `json:"catalog,omitempty"`
	SchemaName     string   `json:"schema,omitempty"`
	Table          string   `json:"table,omitempty"`
}

type VectorChunkResult struct {
	ID          string         `json:"id,omitempty"`
	ChunkID     string         `json:"chunk_id"`
	RowID       *string        `json:"row_id,omitempty"`
	Text        string         `json:"text,omitempty"`
	PageContent string         `json:"page_content,omitempty"`
	Content     string         `json:"content,omitempty"`
	Score       float64        `json:"score"`
	Source      string         `json:"source,omitempty"`
	IngestedAt  *string        `json:"ingested_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type VectorRetrieveResponse struct {
	OK           bool                `json:"ok"`
	Query        string              `json:"query"`
	Model        *string             `json:"model,omitempty"`
	Dimension    *int                `json:"dimension,omitempty"`
	TotalResults int                 `json:"total_results"`
	Results      []VectorChunkResult `json:"results"`
	Error        *string             `json:"error,omitempty"`
}

type OpenAIVectorStore struct {
	ID         string         `json:"id"`
	Object     string         `json:"object"`
	CreatedAt  int64          `json:"created_at"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	UsageBytes int64          `json:"usage_bytes"`
	FileCounts map[string]int `json:"file_counts,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type OpenAIVectorStoreListResponse struct {
	Object  string              `json:"object"`
	Data    []OpenAIVectorStore `json:"data"`
	FirstID *string             `json:"first_id,omitempty"`
	LastID  *string             `json:"last_id,omitempty"`
	HasMore bool                `json:"has_more"`
}

type VectorStoreSearchRequest struct {
	Query          string         `json:"query"`
	TopK           int            `json:"top_k,omitempty"`
	ScoreThreshold *float64       `json:"score_threshold,omitempty"`
	Filters        map[string]any `json:"filters,omitempty"`
}

type VectorStoreSearchItem struct {
	FileID     string           `json:"file_id"`
	Filename   string           `json:"filename"`
	Score      float64          `json:"score"`
	Attributes map[string]any   `json:"attributes,omitempty"`
	Content    []map[string]any `json:"content,omitempty"`
}

type VectorStoreSearchResponse struct {
	Object      string                  `json:"object"`
	SearchQuery string                  `json:"search_query"`
	Data        []VectorStoreSearchItem `json:"data"`
	HasMore     bool                    `json:"has_more"`
}

type OpenAIModel struct {
	ID            string  `json:"id"`
	Object        string  `json:"object"`
	Created       int64   `json:"created"`
	OwnedBy       string  `json:"owned_by"`
	Description   string  `json:"description,omitempty"`
	WorkspaceSlug *string `json:"workspace_slug,omitempty"`
}

type OpenAIModelListResponse struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

type OpenAIEmbeddingRequest struct {
	Input          any    `json:"input"`
	Model          string `json:"model,omitempty"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

type OpenAIEmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type OpenAIEmbeddingResponse struct {
	Object string                `json:"object"`
	Model  string                `json:"model"`
	Data   []OpenAIEmbeddingData `json:"data"`
	Usage  map[string]int        `json:"usage,omitempty"`
}

type TableFromPromptRequest struct {
	Prompt     string `json:"prompt"`
	Catalog    string `json:"catalog,omitempty"`
	SchemaName string `json:"schema,omitempty"`
	Model      string `json:"model,omitempty"`
}

type TableFromPromptResponse struct {
	OK          bool             `json:"ok"`
	Toon        string           `json:"toon"`
	TableName   string           `json:"table_name"`
	DisplayName string           `json:"display_name"`
	Description *string          `json:"description,omitempty"`
	Columns     []map[string]any `json:"columns"`
	SQLDDL      string           `json:"sql_ddl"`
	Error       *string          `json:"error,omitempty"`
}

type GenerateDescriptionRequest struct {
	AssetType   string           `json:"asset_type"`
	Name        string           `json:"name"`
	TableName   string           `json:"table_name,omitempty"`
	CatalogName string           `json:"catalog_name,omitempty"`
	SchemaName  string           `json:"schema_name,omitempty"`
	Columns     []map[string]any `json:"columns,omitempty"`
	SampleRows  []map[string]any `json:"sample_rows,omitempty"`
	Model       string           `json:"model,omitempty"`
}

type GenerateDescriptionResponse struct {
	OK                 bool              `json:"ok"`
	Description        string            `json:"description"`
	ColumnDescriptions map[string]string `json:"column_descriptions,omitempty"`
	Error              *string           `json:"error,omitempty"`
}

func (c *Client) RetrieveVectorChunks(reqPayload VectorRetrieveRequest) (*VectorRetrieveResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/retrieve", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve vector chunks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vector retrieve error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res VectorRetrieveResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode vector retrieve response: %w", err)
	}
	return &res, nil
}

func (c *Client) ListVectorStores() (*OpenAIVectorStoreListResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/v1/vector_stores", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list vector stores: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list vector stores error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res OpenAIVectorStoreListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode vector stores: %w", err)
	}
	return &res, nil
}

func (c *Client) SearchVectorStore(vectorStoreID string, reqPayload VectorStoreSearchRequest) (*VectorStoreSearchResponse, error) {
	path := fmt.Sprintf("/api/v1/vector_stores/%s/search", url.PathEscape(vectorStoreID))
	req, err := c.newRequest(http.MethodPost, path, reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store %s: %w", vectorStoreID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search vector store error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res VectorStoreSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode vector store search response: %w", err)
	}
	return &res, nil
}

func (c *Client) ListAIModels() (*OpenAIModelListResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/v1/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual AI models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list AI models error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res OpenAIModelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode AI models: %w", err)
	}
	return &res, nil
}

func (c *Client) CreateEmbeddings(reqPayload OpenAIEmbeddingRequest) (*OpenAIEmbeddingResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/embeddings", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create embeddings error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode embeddings response: %w", err)
	}
	return &res, nil
}

func (c *Client) GenerateTableFromPrompt(reqPayload TableFromPromptRequest) (*TableFromPromptResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/ai/table-from-prompt", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate table from prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("table-from-prompt error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res TableFromPromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode table-from-prompt response: %w", err)
	}
	return &res, nil
}

func (c *Client) EnrichCatalogAsset(reqPayload GenerateDescriptionRequest) (*GenerateDescriptionResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/api/ai/generate-description", reqPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich catalog asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrich catalog asset error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res GenerateDescriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode enrich response: %w", err)
	}
	return &res, nil
}
