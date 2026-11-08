package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// StorageFolder describes a subfolder/common prefix in object storage.
type StorageFolder struct {
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	FullPrefix string `json:"full_prefix,omitempty"`
}

// StorageFile describes a stored object in Lake of Tears object storage.
type StorageFile struct {
	Key           string     `json:"key"`
	RelKey        string     `json:"rel_key,omitempty"`
	Name          string     `json:"name"`
	Size          int64      `json:"size"`
	SizeFormatted string     `json:"size_formatted,omitempty"`
	LastModified  *time.Time `json:"last_modified,omitempty"`
	ETag          string     `json:"etag,omitempty"`
	IsParquet     bool       `json:"is_parquet,omitempty"`
	IsNotebook    bool       `json:"is_notebook,omitempty"`
}

// StorageBrowseResponse represents the directory listing returned by /api/storage/browse.
type StorageBrowseResponse struct {
	Bucket                string          `json:"bucket"`
	Workspace             string          `json:"workspace,omitempty"`
	Prefix                string          `json:"prefix"`
	RelPrefix             string          `json:"rel_prefix,omitempty"`
	ParentPrefix          *string         `json:"parent_prefix,omitempty"`
	Folders               []StorageFolder `json:"folders"`
	Files                 []StorageFile   `json:"files"`
	TotalFolders          int             `json:"total_folders"`
	TotalFiles            int             `json:"total_files"`
	IsTruncated           bool            `json:"is_truncated"`
	NextContinuationToken *string         `json:"next_continuation_token,omitempty"`
}

// ParquetColumnMetadata describes column types inside a Parquet file.
type ParquetColumnMetadata struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// StorageMetadataResponse contains object metadata and optional Parquet metrics.
type StorageMetadataResponse struct {
	Key            string                  `json:"key"`
	RelKey         string                  `json:"rel_key,omitempty"`
	Size           int64                   `json:"size"`
	SizeFormatted  string                  `json:"size_formatted,omitempty"`
	LastModified   *time.Time              `json:"last_modified,omitempty"`
	ContentType    string                  `json:"content_type"`
	ETag           string                  `json:"etag"`
	CustomMetadata map[string]string       `json:"custom_metadata,omitempty"`
	IsParquet      bool                    `json:"is_parquet"`
	ParquetColumns []ParquetColumnMetadata `json:"parquet_columns,omitempty"`
	RowCount       *int64                  `json:"row_count,omitempty"`
	ColumnCount    *int                    `json:"column_count,omitempty"`
	Compression    string                  `json:"compression,omitempty"`
}

// StorageUploadResponse represents the upload confirmation payload.
type StorageUploadResponse struct {
	Key         string `json:"key"`
	RelKey      string `json:"rel_key,omitempty"`
	Size        int64  `json:"size"`
	ETag        string `json:"etag,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Status      string `json:"status,omitempty"`
}

// StorageStatsResponse represents aggregated workspace storage metrics.
type StorageStatsResponse struct {
	Bucket            string         `json:"bucket"`
	Workspace         string         `json:"workspace,omitempty"`
	TotalBytes        int64          `json:"total_bytes"`
	TotalObjects      int64          `json:"total_objects"`
	TopLevelPrefixes  map[string]int `json:"top_level_prefixes,omitempty"`
	ParquetTotalBytes int64          `json:"parquet_total_bytes,omitempty"`
}

// CleanStoragePath strips protocol schemes like s3://, lot://, dbfs:// and leading slashes.
func CleanStoragePath(p string) string {
	cleaned := strings.TrimSpace(p)
	for _, scheme := range []string{"lot://", "s3://", "dbfs://"} {
		if strings.HasPrefix(strings.ToLower(cleaned), scheme) {
			cleaned = cleaned[len(scheme):]
			break
		}
	}
	return strings.TrimPrefix(cleaned, "/")
}

// BrowseStorage lists folders and files within a prefix jailed to the active workspace.
func (c *Client) BrowseStorage(prefix string, limit int, continuationToken string) (*StorageBrowseResponse, error) {
	cleanPrefix := CleanStoragePath(prefix)
	params := url.Values{}
	if cleanPrefix != "" {
		params.Set("prefix", cleanPrefix)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if continuationToken != "" {
		params.Set("continuation_token", continuationToken)
	}

	path := "/api/storage/browse"
	if len(params) > 0 {
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to browse storage: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("browse storage failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var browseResp StorageBrowseResponse
	if err := json.Unmarshal(bodyBytes, &browseResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage browse response: %w", err)
	}

	return &browseResp, nil
}

// GetStorageMetadata inspects an S3 object key for size, headers, and Parquet schema.
func (c *Client) GetStorageMetadata(key string) (*StorageMetadataResponse, error) {
	cleanKey := CleanStoragePath(key)
	path := fmt.Sprintf("/api/storage/objects/%s/metadata", url.PathEscape(cleanKey))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage metadata: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get storage metadata failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var metaResp StorageMetadataResponse
	if err := json.Unmarshal(bodyBytes, &metaResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage metadata: %w", err)
	}

	return &metaResp, nil
}

// StreamObject streams the content of a remote storage object, with optional byte-range support.
func (c *Client) StreamObject(key string, offset, length int64) (io.ReadCloser, int64, error) {
	cleanKey := CleanStoragePath(key)
	path := fmt.Sprintf("/api/storage/objects/%s/stream", url.PathEscape(cleanKey))
	httpReq, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}

	if offset > 0 || length > 0 {
		if length > 0 {
			httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
		} else {
			httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stream object: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("stream object failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	contentLength := resp.ContentLength
	return resp.Body, contentLength, nil
}

// DownloadObject streams a remote object and writes it directly to the provided writer.
func (c *Client) DownloadObject(key string, w io.Writer) error {
	reader, _, err := c.StreamObject(key, 0, 0)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(w, reader)
	return err
}

// UploadObject uploads data directly to a storage prefix via multipart/form-data.
func (c *Client) UploadObject(prefix, filename string, r io.Reader, contentType string) (*StorageUploadResponse, error) {
	cleanPrefix := CleanStoragePath(prefix)
	cleanFilename := filepath.Base(filename)
	if cleanFilename == "." || cleanFilename == "/" || cleanFilename == "" {
		cleanFilename = "upload"
	}

	var bodyBuf bytes.Buffer
	writer := multipart.NewWriter(&bodyBuf)

	if cleanPrefix != "" {
		if err := writer.WriteField("prefix", cleanPrefix); err != nil {
			return nil, err
		}
	}

	part, err := writer.CreateFormFile("file", cleanFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart form file: %w", err)
	}

	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("failed to write multipart form data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	path := "/api/storage/upload"
	reqURL := fmt.Sprintf("%s%s", c.Endpoint, path)
	httpReq, err := http.NewRequest(http.MethodPost, reqURL, &bodyBuf)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("User-Agent", "angler-cli")

	wsSlug := c.Workspace
	wsID := c.WorkspaceID
	if wsID == "" && isUUID(wsSlug) {
		wsID = wsSlug
	}
	if wsSlug != "" {
		httpReq.Header.Set("X-Workspace-Slug", wsSlug)
	} else {
		httpReq.Header.Set("X-Workspace-Slug", "default")
	}
	if wsID != "" {
		httpReq.Header.Set("X-Workspace-Id", wsID)
	}
	if c.Token != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("upload object failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var uploadResp StorageUploadResponse
	if err := json.Unmarshal(bodyBytes, &uploadResp); err != nil {
		return nil, fmt.Errorf("failed to decode upload response: %w", err)
	}

	return &uploadResp, nil
}

// DeleteObject removes an object or directory prefix from workspace storage.
func (c *Client) DeleteObject(key string, recursive bool) error {
	cleanKey := CleanStoragePath(key)
	path := fmt.Sprintf("/api/storage/objects/%s", url.PathEscape(cleanKey))
	if recursive {
		path = fmt.Sprintf("%s?recursive=true", path)
	}

	httpReq, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete object failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetStorageStats retrieves aggregate bucket metrics for the active workspace.
func (c *Client) GetStorageStats() (*StorageStatsResponse, error) {
	httpReq, err := c.newRequest(http.MethodGet, "/api/storage/stats", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get storage stats failed (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var statsResp StorageStatsResponse
	if err := json.Unmarshal(bodyBytes, &statsResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage stats: %w", err)
	}

	return &statsResp, nil
}
