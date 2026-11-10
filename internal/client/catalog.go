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

// ── Catalog Soft-Delete Trash & Undrop Recovery ──────────────────────────────

// DeletedCatalogAsset represents a soft-deleted catalog, schema, or table in the workspace trash.
type DeletedCatalogAsset struct {
	ID                    string                   `json:"id"`
	WorkspaceID           string                   `json:"workspace_id,omitempty"`
	CatalogID             string                   `json:"catalog_id,omitempty"`
	CatalogName           string                   `json:"catalog_name,omitempty"`
	SchemaID              string                   `json:"schema_id,omitempty"`
	SchemaName            string                   `json:"schema_name,omitempty"`
	AssetType             string                   `json:"asset_type"`
	Name                  string                   `json:"name"`
	Slug                  string                   `json:"slug"`
	Tier                  string                   `json:"tier,omitempty"`
	Description           string                   `json:"description,omitempty"`
	S3PathPattern         string                   `json:"s3_path_pattern,omitempty"`
	ColumnDefs            []map[string]interface{} `json:"column_defs,omitempty"`
	SchemaDrift           bool                     `json:"schema_drift"`
	IsManaged             bool                     `json:"is_managed"`
	TrashS3Prefix         string                   `json:"trash_s3_prefix,omitempty"`
	TrashEmbeddingsPrefix string                   `json:"trash_embeddings_prefix,omitempty"`
	BatchID               string                   `json:"batch_id,omitempty"`
	DroppedBy             string                   `json:"dropped_by,omitempty"`
	DroppedByEmail        string                   `json:"dropped_by_email,omitempty"`
	DroppedAt             time.Time                `json:"dropped_at"`
	PurgeAt               time.Time                `json:"purge_at"`
	DaysRemaining         float64                  `json:"days_remaining"`
}

// RestoreTrashAssetRequest defines optional conflict-resolution options when restoring a trashed asset.
type RestoreTrashAssetRequest struct {
	Strategy string `json:"strategy,omitempty"`
	NewName  string `json:"new_name,omitempty"`
}

// ListCatalogTrash fetches all soft-deleted catalog assets in the active workspace.
func (c *Client) ListCatalogTrash() ([]DeletedCatalogAsset, error) {
	httpReq, err := c.newRequest(http.MethodGet, "/api/catalogs/trash", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog trash: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list catalog trash failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var assets []DeletedCatalogAsset
	if err := json.Unmarshal(bodyBytes, &assets); err != nil {
		return nil, fmt.Errorf("failed to decode catalog trash: %w", err)
	}

	return assets, nil
}

// RestoreCatalogTrashAsset restores (undrops) a soft-deleted asset by ID.
func (c *Client) RestoreCatalogTrashAsset(assetID string, req *RestoreTrashAssetRequest) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/catalogs/trash/%s/restore", url.PathEscape(assetID))
	var payload interface{}
	if req != nil && (req.Strategy != "" || req.NewName != "") {
		payload = req
	} else {
		payload = map[string]interface{}{}
	}

	httpReq, err := c.newRequest(http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to restore trashed asset: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("restore trashed asset failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &res)
	}
	if res == nil {
		res = map[string]interface{}{"ok": true, "restored_id": assetID}
	}
	return res, nil
}

// PurgeCatalogTrashAsset permanently deletes a specific trashed asset.
func (c *Client) PurgeCatalogTrashAsset(assetID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/catalogs/trash/%s/purge", url.PathEscape(assetID))
	httpReq, err := c.newRequest(http.MethodPost, path, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to purge trashed asset: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("purge trashed asset failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &res)
	}
	if res == nil {
		res = map[string]interface{}{"ok": true, "purged_id": assetID}
	}
	return res, nil
}

// PurgeExpiredCatalogTrash permanently purges all trashed assets past their retention TTL.
func (c *Client) PurgeExpiredCatalogTrash() (map[string]interface{}, error) {
	httpReq, err := c.newRequest(http.MethodPost, "/api/catalogs/trash/purge-expired", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to purge expired catalog trash: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("purge expired catalog trash failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &res)
	}
	if res == nil {
		res = map[string]interface{}{"ok": true}
	}
	return res, nil
}

// ── Catalog, Schema & Table Updates + Native TOON Export ─────────────────────

// UpdateCatalogRequest represents the payload to update a catalog.
type UpdateCatalogRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateSchemaRequest represents the payload to update a schema.
type UpdateSchemaRequest struct {
	Description *string `json:"description,omitempty"`
}

// UpdateTableRequest represents the payload to update a table.
type UpdateTableRequest struct {
	Description   *string                  `json:"description,omitempty"`
	S3PathPattern *string                  `json:"s3_path_pattern,omitempty"`
	ColumnDefs    []map[string]interface{} `json:"column_defs,omitempty"`
}

// TOONExportResponse represents the response when exporting a schema or table in TOON format.
type TOONExportResponse struct {
	OK       bool   `json:"ok"`
	TOON     string `json:"toon"`
	Catalog  string `json:"catalog,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Table    string `json:"table,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// UpdateCatalog updates catalog metadata (name, description).
func (c *Client) UpdateCatalog(catalogID string, req UpdateCatalogRequest) (*Catalog, error) {
	path := fmt.Sprintf("/api/catalogs/%s", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update catalog: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update catalog failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var cat Catalog
	if err := json.Unmarshal(bodyBytes, &cat); err != nil {
		return nil, fmt.Errorf("failed to decode updated catalog: %w", err)
	}

	return &cat, nil
}

// UpdateSchema updates schema metadata (description).
func (c *Client) UpdateSchema(catalogID, schemaID string, req UpdateSchemaRequest) (*CatalogSchema, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s", url.PathEscape(catalogID), url.PathEscape(schemaID))
	httpReq, err := c.newRequest(http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update schema: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update schema failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var sch CatalogSchema
	if err := json.Unmarshal(bodyBytes, &sch); err != nil {
		return nil, fmt.Errorf("failed to decode updated schema: %w", err)
	}

	return &sch, nil
}

// UpdateTable updates table metadata (description, s3_path_pattern, column_defs).
func (c *Client) UpdateTable(catalogID, schemaID, tableID string, req UpdateTableRequest) (*CatalogTable, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables/%s",
		url.PathEscape(catalogID),
		url.PathEscape(schemaID),
		url.PathEscape(tableID),
	)
	httpReq, err := c.newRequest(http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update table: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update table failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tbl CatalogTable
	if err := json.Unmarshal(bodyBytes, &tbl); err != nil {
		return nil, fmt.Errorf("failed to decode updated table: %w", err)
	}

	return &tbl, nil
}

// GetSchemaTOON exports a catalog schema in Token-Oriented Object Notation (TOON).
func (c *Client) GetSchemaTOON(catalogID, schemaID string) (*TOONExportResponse, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/toon", url.PathEscape(catalogID), url.PathEscape(schemaID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to export schema TOON: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("export schema TOON failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var toonResp TOONExportResponse
	if err := json.Unmarshal(bodyBytes, &toonResp); err != nil {
		// Fallback if server returned raw text
		return &TOONExportResponse{
			OK:      true,
			TOON:    string(bodyBytes),
			Catalog: catalogID,
			Schema:  schemaID,
		}, nil
	}

	return &toonResp, nil
}

// GetTableTOON exports a catalog table in Token-Oriented Object Notation (TOON).
func (c *Client) GetTableTOON(catalogID, schemaID, tableID string) (*TOONExportResponse, error) {
	path := fmt.Sprintf("/api/catalogs/%s/schemas/%s/tables/%s/toon",
		url.PathEscape(catalogID),
		url.PathEscape(schemaID),
		url.PathEscape(tableID),
	)
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to export table TOON: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("export table TOON failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var toonResp TOONExportResponse
	if err := json.Unmarshal(bodyBytes, &toonResp); err != nil {
		return &TOONExportResponse{
			OK:      true,
			TOON:    string(bodyBytes),
			Catalog: catalogID,
			Schema:  schemaID,
			Table:   tableID,
		}, nil
	}

	return &toonResp, nil
}

// ── User-Scoped Catalog Directory & Access Requests ──────────────────────────

// CatalogDirectoryItem represents a discoverable catalog across the organization.
type CatalogDirectoryItem struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Description        string `json:"description,omitempty"`
	OwnerWorkspaceID   string `json:"owner_workspace_id"`
	OwnerWorkspaceName string `json:"owner_workspace_name,omitempty"`
	SchemaCount        int    `json:"schema_count"`
	AccessStatus       string `json:"access_status"`
	AccessID           string `json:"access_id,omitempty"`
}

// CatalogAccessItem represents an inbound access request or active catalog share.
type CatalogAccessItem struct {
	ID               string     `json:"id"`
	CatalogID        string     `json:"catalog_id"`
	WorkspaceID      string     `json:"workspace_id"`
	WorkspaceName    string     `json:"workspace_name,omitempty"`
	Mode             string     `json:"mode"`
	Status           string     `json:"status"`
	Suspended        bool       `json:"suspended"`
	RequestedByEmail string     `json:"requested_by_email,omitempty"`
	RequestedAt      time.Time  `json:"requested_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
}

// RequestCatalogAccessRequest represents a payload to request access to a catalog.
type RequestCatalogAccessRequest struct {
	Mode        string `json:"mode,omitempty"`
	Message     string `json:"message,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// UpdateCatalogAccessRequest represents a payload to approve or reject an access request.
type UpdateCatalogAccessRequest struct {
	Status string `json:"status"`
}

// ListCatalogDirectory lists all discoverable catalogs across the organization with caller access status.
func (c *Client) ListCatalogDirectory(workspaceID string) ([]CatalogDirectoryItem, error) {
	wsID := workspaceID
	if wsID == "" {
		wsID = c.WorkspaceID
	}
	if wsID == "" && isUUID(c.Workspace) {
		wsID = c.Workspace
	}
	if wsID == "" {
		// Attempt to resolve active workspace UUID from ListWorkspaces if possible
		if workspaces, err := c.ListWorkspaces(); err == nil {
			target := c.Workspace
			if target == "" {
				target = "default"
			}
			for _, w := range workspaces {
				if w.ID == target || w.Slug == target || w.Name == target {
					wsID = w.ID
					break
				}
			}
			if wsID == "" && len(workspaces) > 0 {
				wsID = workspaces[0].ID
			}
		}
	}
	if wsID == "" {
		wsID = c.Workspace
		if wsID == "" {
			wsID = "default"
		}
	}

	path := fmt.Sprintf("/api/catalogs/directory?workspace_id=%s", url.QueryEscape(wsID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog directory: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list catalog directory failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var items []CatalogDirectoryItem
	if err := json.Unmarshal(bodyBytes, &items); err != nil {
		return nil, fmt.Errorf("failed to decode catalog directory: %w", err)
	}

	return items, nil
}

// ListCatalogAccess lists inbound access requests and active user-scoped shares for a catalog.
func (c *Client) ListCatalogAccess(catalogID string) ([]CatalogAccessItem, error) {
	path := fmt.Sprintf("/api/catalogs/%s/access", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog access: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list catalog access failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var items []CatalogAccessItem
	if err := json.Unmarshal(bodyBytes, &items); err != nil {
		return nil, fmt.Errorf("failed to decode catalog access list: %w", err)
	}

	return items, nil
}

// RequestCatalogAccess requests access to an external catalog.
func (c *Client) RequestCatalogAccess(catalogID string, req RequestCatalogAccessRequest) (map[string]interface{}, error) {
	if req.Mode == "" {
		req.Mode = "read"
	}
	if req.Message == "" {
		req.Message = "Requested via Angler CLI"
	}
	path := fmt.Sprintf("/api/catalogs/%s/access/request", url.PathEscape(catalogID))
	httpReq, err := c.newRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to request catalog access: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request catalog access failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &res)
	}
	if res == nil {
		res = map[string]interface{}{"ok": true}
	}
	return res, nil
}

// UpdateCatalogAccess approves or rejects an inbound catalog access request.
func (c *Client) UpdateCatalogAccess(catalogID, accessID, status string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/api/catalogs/%s/access/%s", url.PathEscape(catalogID), url.PathEscape(accessID))
	httpReq, err := c.newRequest(http.MethodPatch, path, UpdateCatalogAccessRequest{Status: status})
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update catalog access: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update catalog access failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &res)
	}
	if res == nil {
		res = map[string]interface{}{"ok": true, "status": status}
	}
	return res, nil
}

// RevokeCatalogAccess revokes a user-scoped catalog share.
func (c *Client) RevokeCatalogAccess(catalogID, accessID string) error {
	path := fmt.Sprintf("/api/catalogs/%s/access/%s", url.PathEscape(catalogID), url.PathEscape(accessID))
	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to revoke catalog access: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revoke catalog access failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
