package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NotebookItem describes a single item in the workspace file tree.
type NotebookItem struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	Type         string     `json:"type"` // "notebook", "directory", "file"
	ParentDir    string     `json:"parent_dir,omitempty"`
	LastModified *time.Time `json:"last_modified,omitempty"`
	Created      *time.Time `json:"created,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	Writable     bool       `json:"writable,omitempty"`
}

// NotebookListResponse is the response returned by /api/notebooks/files.
type NotebookListResponse struct {
	Items     []NotebookItem `json:"items"`
	Path      string         `json:"path"`
	Workspace string         `json:"workspace,omitempty"`
}

// NotebookDirectoriesResponse represents directory hierarchy listings.
type NotebookDirectoriesResponse struct {
	Directories []string `json:"directories"`
	Workspace   string   `json:"workspace,omitempty"`
}

// NotebookSearchResponse represents search results for workspace notebooks.
type NotebookSearchResponse struct {
	Items     []NotebookItem `json:"items"`
	Query     string         `json:"query"`
	Subpath   string         `json:"subpath,omitempty"`
	Workspace string         `json:"workspace,omitempty"`
}

// NotebookCellOutput represents a Jupyter cell output.
type NotebookCellOutput struct {
	OutputType string                 `json:"output_type,omitempty"`
	Text       interface{}            `json:"text,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Ename      string                 `json:"ename,omitempty"`
	Evalue     string                 `json:"evalue,omitempty"`
	Traceback  []string               `json:"traceback,omitempty"`
}

// NotebookCell represents a cell inside a Jupyter notebook.
type NotebookCell struct {
	CellType       string               `json:"cell_type"`
	ExecutionCount *int                 `json:"execution_count,omitempty"`
	Metadata       map[string]any       `json:"metadata,omitempty"`
	Outputs        []NotebookCellOutput `json:"outputs,omitempty"`
	Source         interface{}          `json:"source"` // []string or string
}

// GetSourceString returns cell source as a single joined string.
func (c *NotebookCell) GetSourceString() string {
	switch v := c.Source.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, line := range v {
			if s, ok := line.(string); ok {
				sb.WriteString(s)
			}
		}
		return sb.String()
	case []string:
		return strings.Join(v, "")
	default:
		return ""
	}
}

// NotebookContent represents the complete JSON structure of an .ipynb file.
type NotebookContent struct {
	Cells         []NotebookCell `json:"cells"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Nbformat      int            `json:"nbformat"`
	NbformatMinor int            `json:"nbformat_minor"`
}

// NotebookDetail represents a notebook item with metadata and contents.
type NotebookDetail struct {
	Name         string           `json:"name"`
	Path         string           `json:"path"`
	RelPath      string           `json:"rel_path,omitempty"`
	Type         string           `json:"type"`
	Writable     bool             `json:"writable"`
	Created      *time.Time       `json:"created,omitempty"`
	LastModified *time.Time       `json:"last_modified,omitempty"`
	Size         *int64           `json:"size,omitempty"`
	Mimetype     string           `json:"mimetype,omitempty"`
	Content      *NotebookContent `json:"content,omitempty"`
	Format       string           `json:"format,omitempty"`
}

// NotebookRunArchive describes a saved historical run from S3.
type NotebookRunArchive struct {
	Key          string `json:"key"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
}

// NotebookRunsResponse represents the list of historical runs for a notebook.
type NotebookRunsResponse struct {
	Runs      []NotebookRunArchive `json:"runs"`
	Workspace string               `json:"workspace"`
	Notebook  string               `json:"notebook"`
}

// NotebookSession represents a running or attached Jupyter kernel session.
type NotebookSession struct {
	ID     string         `json:"id"`
	Path   string         `json:"path"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Kernel map[string]any `json:"kernel"`
}

// NotebookRunResult captures the outcome of headless notebook execution.
type NotebookRunResult struct {
	Path         string               `json:"path"`
	Status       string               `json:"status"` // "SUCCESS", "FAILED"
	Duration     string               `json:"duration"`
	TotalCells   int                  `json:"total_cells"`
	Executed     int                  `json:"executed"`
	FailedCell   *int                 `json:"failed_cell,omitempty"`
	ErrorMessage string               `json:"error_message,omitempty"`
	CellOutputs  []NotebookCellOutput `json:"cell_outputs,omitempty"`
}

// ListWorkspaceNotebooks retrieves notebooks and folders in the workspace directory.
func (c *Client) ListWorkspaceNotebooks(subpath string) (*NotebookListResponse, error) {
	relPath := "/api/notebooks/files"
	if subpath != "" {
		relPath = fmt.Sprintf("%s?subpath=%s", relPath, url.QueryEscape(subpath))
	}

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list notebooks request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list notebooks request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse notebooks response: %w", err)
	}
	return &res, nil
}

// ListWorkspaceDirectories retrieves all directory paths within the workspace.
func (c *Client) ListWorkspaceDirectories() (*NotebookDirectoriesResponse, error) {
	req, err := c.newRequest(http.MethodGet, "/api/notebooks/directories", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list directories request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list directories request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookDirectoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse directories response: %w", err)
	}
	return &res, nil
}

// SearchWorkspaceItems recursively searches for notebooks, files, and directories.
func (c *Client) SearchWorkspaceItems(query, subpath string) (*NotebookSearchResponse, error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if subpath != "" {
		params.Set("subpath", subpath)
	}

	relPath := "/api/notebooks/search"
	if len(params) > 0 {
		relPath = fmt.Sprintf("%s?%s", relPath, params.Encode())
	}

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}
	return &res, nil
}

// GetNotebook retrieves a workspace notebook and its content.
func (c *Client) GetNotebook(path string) (*NotebookDetail, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	relPath := fmt.Sprintf("/api/notebooks/%s", cleanPath)

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get notebook request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get notebook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookDetail
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse notebook response: %w", err)
	}
	return &res, nil
}

// GetNotebookRaw retrieves the raw JSON or exported bytes for a notebook.
func (c *Client) GetNotebookRaw(path string, download bool) ([]byte, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	relPath := fmt.Sprintf("/api/notebooks/%s", cleanPath)
	if download {
		relPath += "?download=true"
	}

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw notebook request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw notebook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// SaveNotebook saves or updates notebook content.
func (c *Client) SaveNotebook(path string, content *NotebookContent) (*NotebookDetail, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	relPath := fmt.Sprintf("/api/notebooks/%s", cleanPath)

	payload := map[string]interface{}{
		"type":    "notebook",
		"format":  "json",
		"content": content,
	}

	req, err := c.newRequest(http.MethodPut, relPath, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create save notebook request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute save notebook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookDetail
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse save response: %w", err)
	}
	return &res, nil
}

// CreateNotebook initializes a new blank notebook in the given folder with the given name.
func (c *Client) CreateNotebook(folder, name string) (*NotebookDetail, error) {
	payload := map[string]string{
		"type":   "notebook",
		"folder": folder,
		"name":   name,
	}

	req, err := c.newRequest(http.MethodPost, "/api/notebooks/create", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create notebook request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create notebook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookDetail
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse create notebook response: %w", err)
	}
	return &res, nil
}

// CreateWorkspaceDirectory creates a new directory folder inside the workspace.
func (c *Client) CreateWorkspaceDirectory(path string) error {
	cleanPath := strings.TrimPrefix(path, "/")
	parts := strings.Split(cleanPath, "/")

	name := parts[len(parts)-1]
	folder := ""
	if len(parts) > 1 {
		folder = strings.Join(parts[:len(parts)-1], "/")
	}

	payload := map[string]string{
		"type":   "directory",
		"name":   name,
		"folder": folder,
	}

	req, err := c.newRequest(http.MethodPost, "/api/notebooks/create", payload)
	if err != nil {
		return fmt.Errorf("failed to create directory request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute create directory request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteWorkspaceItem removes a notebook or directory at the specified workspace path.
func (c *Client) DeleteWorkspaceItem(path string) error {
	cleanPath := strings.TrimPrefix(path, "/")
	relPath := fmt.Sprintf("/api/notebooks/%s", cleanPath)

	req, err := c.newRequest(http.MethodDelete, relPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete item request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete item request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RenameWorkspaceItem moves or renames a workspace notebook or directory.
func (c *Client) RenameWorkspaceItem(oldPath, newPath string) error {
	cleanOld := strings.TrimPrefix(oldPath, "/")
	cleanNew := strings.TrimPrefix(newPath, "/")
	relPath := fmt.Sprintf("/api/notebooks/%s", cleanOld)

	payload := map[string]string{
		"new_path": cleanNew,
	}

	req, err := c.newRequest(http.MethodPatch, relPath, payload)
	if err != nil {
		return fmt.Errorf("failed to create rename item request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute rename item request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListNotebookRuns retrieves archived historical execution runs for a notebook.
func (c *Client) ListNotebookRuns(notebookPath string) (*NotebookRunsResponse, error) {
	relPath := "/api/notebooks/runs"
	if notebookPath != "" {
		relPath = fmt.Sprintf("%s?path=%s", relPath, url.QueryEscape(notebookPath))
	}

	req, err := c.newRequest(http.MethodGet, relPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list runs request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute list runs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse runs response: %w", err)
	}
	return &res, nil
}

// CreateNotebookSession starts or attaches to a kernel execution session.
func (c *Client) CreateNotebookSession(path, kernelName string) (*NotebookSession, error) {
	if kernelName == "" {
		kernelName = "python3"
	}
	cleanPath := strings.TrimPrefix(path, "/")

	payload := map[string]string{
		"path":        cleanPath,
		"kernel_name": kernelName,
	}

	req, err := c.newRequest(http.MethodPost, "/api/notebooks/sessions", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create notebook session request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create notebook session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var res NotebookSession
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse notebook session response: %w", err)
	}
	return &res, nil
}

// InterruptKernel interrupts an active kernel execution.
func (c *Client) InterruptKernel(kernelID string) error {
	relPath := fmt.Sprintf("/api/notebooks/kernels/%s/interrupt", kernelID)
	req, err := c.newRequest(http.MethodPost, relPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create interrupt kernel request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute interrupt kernel request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RestartKernel restarts a kernel instance.
func (c *Client) RestartKernel(kernelID string) error {
	relPath := fmt.Sprintf("/api/notebooks/kernels/%s/restart", kernelID)
	req, err := c.newRequest(http.MethodPost, relPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create restart kernel request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute restart kernel request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ParseNotebookContent parses raw JSON bytes into a NotebookContent struct.
func ParseNotebookContent(data []byte) (*NotebookContent, error) {
	var nb NotebookContent
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, fmt.Errorf("invalid notebook json: %w", err)
	}
	return &nb, nil
}

// NotebookFromSource converts source code (Python, SQL, Markdown) into a NotebookContent object.
func NotebookFromSource(sourceCode string, langOrFormat string) *NotebookContent {
	format := strings.ToLower(strings.TrimSpace(langOrFormat))
	title := "Source Notebook"

	var cells []NotebookCell

	if format == "markdown" || format == "md" {
		cells = append(cells, NotebookCell{
			CellType: "markdown",
			Metadata: map[string]any{},
			Source:   sourceCode,
		})
	} else if format == "sql" {
		cells = append(cells, NotebookCell{
			CellType: "markdown",
			Metadata: map[string]any{},
			Source:   "# SQL Query Notebook\nInteractive query registered via angler workspace import.",
		})
		cells = append(cells, NotebookCell{
			CellType: "code",
			Metadata: map[string]any{},
			Source:   sourceCode,
			Outputs:  []NotebookCellOutput{},
		})
	} else {
		// Python script: check for cell split markers (# %% or # <codecell>)
		lines := strings.Split(sourceCode, "\n")
		var currentCell []string
		inCode := true

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# %%") || strings.HasPrefix(trimmed, "# <codecell>") {
				if len(currentCell) > 0 {
					cellType := "code"
					if !inCode {
						cellType = "markdown"
					}
					cells = append(cells, NotebookCell{
						CellType: cellType,
						Metadata: map[string]any{},
						Source:   strings.Join(currentCell, "\n"),
						Outputs:  []NotebookCellOutput{},
					})
					currentCell = nil
				}
				inCode = true
				continue
			} else if strings.HasPrefix(trimmed, "# %%% md") || strings.HasPrefix(trimmed, "# <markdowncell>") {
				if len(currentCell) > 0 {
					cellType := "code"
					if !inCode {
						cellType = "markdown"
					}
					cells = append(cells, NotebookCell{
						CellType: cellType,
						Metadata: map[string]any{},
						Source:   strings.Join(currentCell, "\n"),
						Outputs:  []NotebookCellOutput{},
					})
					currentCell = nil
				}
				inCode = false
				continue
			}
			currentCell = append(currentCell, line)
		}

		if len(currentCell) > 0 {
			cellType := "code"
			if !inCode {
				cellType = "markdown"
			}
			cells = append(cells, NotebookCell{
				CellType: cellType,
				Metadata: map[string]any{},
				Source:   strings.Join(currentCell, "\n"),
				Outputs:  []NotebookCellOutput{},
			})
		}

		if len(cells) == 0 {
			cells = append(cells, NotebookCell{
				CellType: "code",
				Metadata: map[string]any{},
				Source:   sourceCode,
				Outputs:  []NotebookCellOutput{},
			})
		}
	}

	kernelLang := "python"
	kernelDisplayName := "Python 3"
	kernelName := "python3"
	if format == "sql" {
		kernelLang = "sql"
		kernelDisplayName = "DuckDB / SQL"
		kernelName = "sql"
	}

	return &NotebookContent{
		Cells: cells,
		Metadata: map[string]any{
			"title": title,
			"kernelspec": map[string]any{
				"display_name": kernelDisplayName,
				"language":     kernelLang,
				"name":         kernelName,
			},
			"language_info": map[string]any{
				"name": kernelLang,
			},
		},
		Nbformat:      4,
		NbformatMinor: 5,
	}
}

// NotebookToSource extracts cells into Python or source script representation with # %% markers.
func NotebookToSource(nb *NotebookContent) string {
	if nb == nil || len(nb.Cells) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, cell := range nb.Cells {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if cell.CellType == "markdown" {
			sb.WriteString("# %% [markdown]\n")
			for _, line := range strings.Split(cell.GetSourceString(), "\n") {
				sb.WriteString("# ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("# %%\n")
			sb.WriteString(cell.GetSourceString())
		}
	}
	return sb.String()
}

// NotebookToJSON formats notebook contents as indented JSON bytes.
func NotebookToJSON(nb *NotebookContent) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(nb); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
