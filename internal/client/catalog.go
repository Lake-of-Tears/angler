package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// CatalogTable represents a table definition within a catalog schema.
type CatalogTable struct {
	ID            string                   `json:"id"`
	SchemaID      string                   `json:"schema_id"`
	Name          string                   `json:"name"`
	Slug          string                   `json:"slug"`
	Description   string                   `json:"description,omitempty"`
	S3PathPattern string                   `json:"s3_path_pattern,omitempty"`
	ColumnDefs    []map[string]interface{} `json:"column_defs,omitempty"`
	SchemaDrift   bool                     `json:"schema_drift"`
	VectorIndex   map[string]interface{}   `json:"vector_index,omitempty"`
	CreatedAt     time.Time                `json:"created_at,omitempty"`
}

// CatalogSchema represents a logical schema grouping within a catalog.
type CatalogSchema struct {
	ID          string         `json:"id"`
	CatalogID   string         `json:"catalog_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description,omitempty"`
	Tier        string         `json:"tier,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	Tables      []CatalogTable `json:"tables,omitempty"`
}

// Catalog represents a Unity-style data catalog.
type Catalog struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	Description        string          `json:"description,omitempty"`
	OwnerWorkspaceID   string          `json:"owner_workspace_id,omitempty"`
	OwnerWorkspaceName string          `json:"owner_workspace_name,omitempty"`
	CreatedAt          time.Time       `json:"created_at,omitempty"`
	IsDefault          bool            `json:"is_default"`
	Schemas            []CatalogSchema `json:"schemas,omitempty"`
	MyAccess           string          `json:"my_access,omitempty"`
}

// CreateCatalogRequest represents the payload to register a catalog.
type CreateCatalogRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// CreateSchemaRequest represents the payload to register a schema.
type CreateSchemaRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateTableRequest represents the payload to register a table.
type CreateTableRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	S3PathPattern string                   `json:"s3_path_pattern,omitempty"`
	ColumnDefs    []map[string]interface{} `json:"column_defs,omitempty"`
}

// AnomalyDetectRequest represents options for triggering anomaly detection.
type AnomalyDetectRequest struct {
	Source        string  `json:"source,omitempty"`
	TableSlug     string  `json:"table_slug,omitempty"`
	Contamination float64 `json:"contamination,omitempty"`
}

// AnomalyDetectResponse represents results of anomaly detection.
type AnomalyDetectResponse struct {
	OK        bool                   `json:"ok"`
	Workspace string                 `json:"workspace,omitempty"`
	Summary   map[string]interface{} `json:"summary,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ListCatalogs fetches all registered catalogs for the active workspace.
func (c *Client) ListCatalogs() ([]Catalog, error) {
	httpReq, err := c.newRequest(http.MethodGet, "/api/catalogs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list catalogs failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var catalogs []Catalog
	if err := json.Unmarshal(bodyBytes, &catalogs); err != nil {
		return nil, fmt.Errorf("failed to decode catalogs list: %w", err)
	}

	return catalogs, nil
}

// GetCatalog retrieves a single catalog by ID or slug.
func (c *Client) GetCatalog(catalogID string) (*Catalog, error) {
	path := fmt.Sprintf("/api/catalogs/%s", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get catalog failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var cat Catalog
	if err := json.Unmarshal(bodyBytes, &cat); err != nil {
		return nil, fmt.Errorf("failed to decode catalog: %w", err)
	}

	return &cat, nil
}

// CreateCatalog registers a new catalog.
func (c *Client) CreateCatalog(req CreateCatalogRequest) (*Catalog, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/catalogs", req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create catalog failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var cat Catalog
	if err := json.Unmarshal(bodyBytes, &cat); err != nil {
		return nil, fmt.Errorf("failed to decode created catalog: %w", err)
	}

	return &cat, nil
}

// DeleteCatalog deletes or deactivates a catalog.
func (c *Client) DeleteCatalog(catalogID string, purge bool) error {
	path := fmt.Sprintf("/api/catalogs/%s", url.PathEscape(catalogID))
	if purge {
		path = fmt.Sprintf("/api/catalogs/%s/purge", url.PathEscape(catalogID))
		httpReq, err := c.newRequest(http.MethodPost, path, map[string]string{"confirm_name": catalogID})
		if err != nil {
			return err
		}
		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to purge catalog: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("purge catalog failed (HTTP %d): %s", resp.StatusCode, string(body))
		}
		return nil
	}

	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete catalog failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListSchemas lists all schemas within a catalog.
func (c *Client) ListSchemas(catalogID string) ([]CatalogSchema, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list schemas failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var schemas []CatalogSchema
	if err := json.Unmarshal(bodyBytes, &schemas); err != nil {
		return nil, fmt.Errorf("failed to decode schemas: %w", err)
	}

	return schemas, nil
}

// GetSchema retrieves schema details by catalog and schema ID.
func (c *Client) GetSchema(catalogID, schemaID string) (*CatalogSchema, error) {
	// List schemas and find by ID/slug or directly inspect tables
	schemas, err := c.ListSchemas(catalogID)
	if err != nil {
		return nil, err
	}

	cleanSchema := url.PathEscape(schemaID)
	for _, s := range schemas {
		if s.ID == schemaID || s.Slug == schemaID || s.Name == schemaID {
			// Populate tables if not already populated
			if len(s.Tables) == 0 {
				tables, _ := c.ListTables(catalogID, s.ID)
				s.Tables = tables
			}
			return &s, nil
		}
	}

	return nil, fmt.Errorf("schema %q not found in catalog %q", cleanSchema, catalogID)
}

// CreateSchema registers a new schema in a catalog.
func (c *Client) CreateSchema(catalogID string, req CreateSchemaRequest) (*CatalogSchema, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create schema failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var schema CatalogSchema
	if err := json.Unmarshal(bodyBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to decode created schema: %w", err)
	}

	return &schema, nil
}

// DeleteSchema drops a schema from a catalog.
func (c *Client) DeleteSchema(catalogID, schemaID string, cascade bool) error {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s", url.PathEscape(catalogID), url.PathEscape(schemaID))
	if cascade {
		path += "?cascade=true"
	}

	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete schema failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListTables lists all tables in a catalog schema.
func (c *Client) ListTables(catalogID, schemaID string) ([]CatalogTable, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables", url.PathEscape(catalogID), url.PathEscape(schemaID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tables failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tables []CatalogTable
	if err := json.Unmarshal(bodyBytes, &tables); err != nil {
		return nil, fmt.Errorf("failed to decode tables: %w", err)
	}

	return tables, nil
}

// GetTable retrieves a specific table by catalog, schema, and table ID.
func (c *Client) GetTable(catalogID, schemaID, tableID string) (*CatalogTable, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables/%s", url.PathEscape(catalogID), url.PathEscape(schemaID), url.PathEscape(tableID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get table failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var table CatalogTable
	if err := json.Unmarshal(bodyBytes, &table); err != nil {
		return nil, fmt.Errorf("failed to decode table: %w", err)
	}

	return &table, nil
}

// CreateTable registers a new table in a schema.
func (c *Client) CreateTable(catalogID, schemaID string, req CreateTableRequest) (*CatalogTable, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables", url.PathEscape(catalogID), url.PathEscape(schemaID))
	httpReq, err := c.newRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create table failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var table CatalogTable
	if err := json.Unmarshal(bodyBytes, &table); err != nil {
		return nil, fmt.Errorf("failed to decode created table: %w", err)
	}

	return &table, nil
}

// DeleteTable drops a table from a schema.
func (c *Client) DeleteTable(catalogID, schemaID, tableID string) error {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables/%s", url.PathEscape(catalogID), url.PathEscape(schemaID), url.PathEscape(tableID))
	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete table: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete table failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetCatalogManifest exports the complete catalog manifest projection.
func (c *Client) GetCatalogManifest() ([]map[string]interface{}, error) {
	httpReq, err := c.newRequest(http.MethodGet, "/api/catalogs/manifest", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog manifest: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get catalog manifest failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var manifest []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return manifest, nil
}

// DetectAnomalies triggers data anomaly detection across tables.
func (c *Client) DetectAnomalies(req AnomalyDetectRequest) (*AnomalyDetectResponse, error) {
	ws := c.Workspace
	if ws == "" {
		ws = "default"
	}
	if c.WorkspaceID != "" {
		ws = c.WorkspaceID
	}

	path := fmt.Sprintf("/api/workspaces/%s/anomalies/detect", url.PathEscape(ws))
	httpReq, err := c.newRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to detect anomalies: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("detect anomalies failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var anomalyResp AnomalyDetectResponse
	if err := json.Unmarshal(bodyBytes, &anomalyResp); err != nil {
		return nil, fmt.Errorf("failed to decode anomaly detection response: %w", err)
	}

	return &anomalyResp, nil
}
