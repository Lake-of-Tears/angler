package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MCPServerDiscovery represents the discovery response from GET /api/mcp.
type MCPServerDiscovery struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	ProtocolVersions []string          `json:"protocolVersions"`
	Endpoints        map[string]string `json:"endpoints"`
	Tools            []string          `json:"tools"`
}

// MCPJSONRPCRequest represents a JSON-RPC 2.0 request to /api/mcp.
type MCPJSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPJSONRPCError represents a JSON-RPC 2.0 error payload.
type MCPJSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCPJSONRPCResponse represents a JSON-RPC 2.0 response from /api/mcp.
type MCPJSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      any              `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *MCPJSONRPCError `json:"error,omitempty"`
}

// MCPInitializeResult represents the JSON-RPC initialize result.
type MCPInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// MCPToolSpec represents an MCP tool specification from tools/list.
type MCPToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// MCPResourceSpec represents an MCP resource specification from resources/list.
type MCPResourceSpec struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPResourceContent represents a resource content item from resources/read.
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// MCPToolCallResult represents the result of a tools/call JSON-RPC request.
type MCPToolCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError"`
}

// GetMCPStatus fetches the remote MCP server discovery metadata from GET /api/mcp.
func (c *Client) GetMCPStatus() (*MCPServerDiscovery, error) {
	req, err := c.newRequest(http.MethodGet, "/api/mcp", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP status failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var out MCPServerDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// callMCPJSONRPC executes a JSON-RPC 2.0 method against POST /api/mcp and unmarshals result into out.
func (c *Client) callMCPJSONRPC(method string, params map[string]any, out any) error {
	rpcReq := MCPJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	req, err := c.newRequest(http.MethodPost, "/api/mcp", rpcReq)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MCP request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var rpcResp MCPJSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("MCP JSON-RPC error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("failed to unmarshal MCP JSON-RPC result: %w", err)
		}
	}
	return nil
}

// MCPInitialize performs the MCP initialize handshake against /api/mcp.
func (c *Client) MCPInitialize(protocolVersion string) (*MCPInitializeResult, error) {
	if protocolVersion == "" {
		protocolVersion = "2024-11-05"
	}
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "angler-cli",
			"version": "1.0.0",
		},
	}
	var res MCPInitializeResult
	if err := c.callMCPJSONRPC("initialize", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// MCPPing sends a JSON-RPC ping to /api/mcp.
func (c *Client) MCPPing() error {
	var res map[string]any
	return c.callMCPJSONRPC("ping", nil, &res)
}

// MCPListTools queries tools/list on the remote /api/mcp server.
func (c *Client) MCPListTools() ([]MCPToolSpec, error) {
	var wrapper struct {
		Tools []MCPToolSpec `json:"tools"`
	}
	if err := c.callMCPJSONRPC("tools/list", nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Tools, nil
}

// MCPCallTool invokes tools/call on the remote /api/mcp server.
func (c *Client) MCPCallTool(name string, arguments map[string]any) (*MCPToolCallResult, error) {
	if arguments == nil {
		arguments = map[string]any{}
	}
	params := map[string]any{
		"name":      name,
		"arguments": arguments,
	}
	var res MCPToolCallResult
	if err := c.callMCPJSONRPC("tools/call", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// MCPListResources queries resources/list on the remote /api/mcp server.
func (c *Client) MCPListResources() ([]MCPResourceSpec, error) {
	var wrapper struct {
		Resources []MCPResourceSpec `json:"resources"`
	}
	if err := c.callMCPJSONRPC("resources/list", nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Resources, nil
}

// MCPReadResource queries resources/read for a given URI (e.g., lot://catalogs/ws/cat) on /api/mcp.
func (c *Client) MCPReadResource(uri string) ([]MCPResourceContent, error) {
	params := map[string]any{
		"uri": uri,
	}
	var wrapper struct {
		Contents []MCPResourceContent `json:"contents"`
	}
	if err := c.callMCPJSONRPC("resources/read", params, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Contents, nil
}

// MCPForwardRaw forwards a raw JSON-RPC message line to POST /api/mcp and returns the raw response bytes and HTTP status.
func (c *Client) MCPForwardRaw(rawJSON []byte) ([]byte, int, error) {
	fullURL := c.Endpoint + "/api/mcp"
	req, err := http.NewRequest(http.MethodPost, fullURL, bytes.NewReader(rawJSON))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "angler-cli")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.Workspace != "" {
		req.Header.Set("X-Workspace-Slug", c.Workspace)
	}
	if c.WorkspaceID != "" {
		req.Header.Set("X-Workspace-Id", c.WorkspaceID)
	}
	if c.OnBehalfOf != "" {
		req.Header.Set("X-On-Behalf-Of", c.OnBehalfOf)
		req.Header.Set("X-Lot-User-Email", c.OnBehalfOf)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
