package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	e2eBinPath string
	mockServer *MockBackend
)

func TestMain(m *testing.M) {
	// 1. Start mock Lake of Tears backend
	mockServer = NewMockBackend()

	// 2. Compile angler binary to a temporary folder
	tmpDir, err := os.MkdirTemp("", "angler-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir for binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	e2eBinPath = filepath.Join(tmpDir, "angler")
	buildCmd := exec.Command("go", "build", "-o", e2eBinPath, "../../main.go")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build angler binary: %v\n", err)
		os.Exit(1)
	}

	// 3. Run all tests
	code := m.Run()

	// 4. Teardown
	mockServer.Close()
	os.Exit(code)
}

func runCLI(t *testing.T, homeDir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(e2eBinPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", homeDir),
		"ANGLER_NON_INTERACTIVE=true",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestE2E_AuthLifecycle(t *testing.T) {
	homeDir := t.TempDir()

	// 1. Initial login with email and password
	stdout, stderr, err := runCLI(t, homeDir, "auth", "login",
		"--endpoint", mockServer.URL,
		"--email", "admin@thelakeoftears.com",
		"--password", "secret123",
	)
	if err != nil {
		t.Fatalf("auth login failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout+stderr, "Successfully logged in") && !strings.Contains(stdout+stderr, "admin@thelakeoftears.com") {
		t.Fatalf("unexpected login output:\n%s\n%s", stdout, stderr)
	}

	// 2. Check auth status
	stdout, stderr, err = runCLI(t, homeDir, "auth", "status", "-o", "json")
	if err != nil {
		t.Fatalf("auth status failed: %v\nstderr: %s", err, stderr)
	}
	var status struct {
		Authenticated bool   `json:"authenticated"`
		Email         string `json:"email"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("failed to parse auth status JSON: %v\nstdout: %s", err, stdout)
	}
	if !status.Authenticated || status.Email != "admin@thelakeoftears.com" {
		t.Fatalf("auth status mismatch: %+v", status)
	}

	// 3. Create a Personal Access Token (PAT)
	stdout, stderr, err = runCLI(t, homeDir, "tokens", "create", "--name", "e2e-ci-pat", "--lifetime-days", "30", "-o", "json")
	if err != nil {
		t.Fatalf("tokens create failed: %v\nstderr: %s", err, stderr)
	}
	var patResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(stdout), &patResp); err != nil {
		t.Fatalf("failed to parse token response JSON: %v\nstdout: %s", err, stdout)
	}
	if !strings.HasPrefix(patResp.Token, "lot_pat_") {
		t.Fatalf("expected lot_pat_ token prefix, got: %s", patResp.Token)
	}

	// 4. Re-login using the created PAT
	stdout, stderr, err = runCLI(t, homeDir, "auth", "login",
		"--endpoint", mockServer.URL,
		"--token", patResp.Token,
	)
	if err != nil {
		t.Fatalf("auth login with PAT failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout+stderr, "authenticated") {
		t.Fatalf("expected PAT authentication confirmation, got:\n%s\n%s", stdout, stderr)
	}
}

func TestE2E_StorageLifecycle(t *testing.T) {
	homeDir := t.TempDir()
	sampleContent := "col1,col2,col3\nval1,val2,val3\nhello,lake,of-tears\n"
	sampleFilePath := filepath.Join(homeDir, "data_sample.csv")
	if err := os.WriteFile(sampleFilePath, []byte(sampleContent), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	// 1. Upload file to bronze/telemetry/
	stdout, stderr, err := runCLI(t, homeDir, "fs", "upload", sampleFilePath, "bronze/telemetry/",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("fs upload failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}

	// 2. List storage objects (ls)
	stdout, stderr, err = runCLI(t, homeDir, "fs", "ls", "bronze/telemetry/", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("fs ls failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "data_sample.csv") {
		t.Fatalf("expected ls output to contain data_sample.csv, got: %s", stdout)
	}

	// 3. Stream content (cat)
	stdout, stderr, err = runCLI(t, homeDir, "fs", "cat", "bronze/telemetry/data_sample.csv",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("fs cat failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "hello,lake,of-tears") {
		t.Fatalf("cat output did not match uploaded content: %s", stdout)
	}

	// 4. Download file
	downloadDest := filepath.Join(homeDir, "downloaded_sample.csv")
	stdout, stderr, err = runCLI(t, homeDir, "fs", "download", "bronze/telemetry/data_sample.csv", downloadDest,
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("fs download failed: %v\nstderr: %s", err, stderr)
	}

	downloadedBytes, err := os.ReadFile(downloadDest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(downloadedBytes) != sampleContent {
		t.Fatalf("downloaded content mismatch.\nExpected:\n%s\nGot:\n%s", sampleContent, string(downloadedBytes))
	}
}

func TestE2E_SQLQueryLifecycle(t *testing.T) {
	homeDir := t.TempDir()

	// 1. JSON output formatting
	stdout, stderr, err := runCLI(t, homeDir, "query", "SELECT 1 AS num, 'lake' AS name", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("query json failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "num") || !strings.Contains(stdout, "lake") {
		t.Fatalf("unexpected json query result: %s", stdout)
	}

	// 2. CSV output formatting
	stdout, stderr, err = runCLI(t, homeDir, "query", "SELECT 1 AS num, 'lake' AS name", "-o", "csv",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("query csv failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "num,name") || !strings.Contains(stdout, "1,lake") {
		t.Fatalf("unexpected csv query result: %s", stdout)
	}

	// 3. Table output formatting
	stdout, stderr, err = runCLI(t, homeDir, "query", "SELECT 1 AS num, 'lake' AS name", "-o", "table",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("query table failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "num") || !strings.Contains(stdout, "name") {
		t.Fatalf("unexpected table query result: %s", stdout)
	}
}

func TestE2E_CatalogInspection(t *testing.T) {
	homeDir := t.TempDir()

	// 1. List catalogs
	stdout, stderr, err := runCLI(t, homeDir, "catalog", "list", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("catalog list failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "default_catalog") {
		t.Fatalf("expected default_catalog in list output: %s", stdout)
	}

	// 2. Inspect catalog details
	stdout, stderr, err = runCLI(t, homeDir, "catalog", "get", "default_catalog", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("catalog get failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "default_catalog") {
		t.Fatalf("expected catalog get output: %s", stdout)
	}

	// 3. List schemas
	stdout, stderr, err = runCLI(t, homeDir, "catalog", "schemas", "list", "default_catalog", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("catalog schemas list failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "analytics") {
		t.Fatalf("expected analytics schema: %s", stdout)
	}

	// 4. List tables
	stdout, stderr, err = runCLI(t, homeDir, "catalog", "tables", "list", "default_catalog", "sch-analytics", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("catalog tables list failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "events") {
		t.Fatalf("expected events table: %s", stdout)
	}

	// 5. Catalog manifest export
	stdout, stderr, err = runCLI(t, homeDir, "catalog", "manifest", "-o", "json",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	if err != nil {
		t.Fatalf("catalog manifest failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "default_catalog") || !strings.Contains(stdout, "analytics") {
		t.Fatalf("unexpected manifest output: %s", stdout)
	}
}

func TestE2E_MCPStdio(t *testing.T) {
	homeDir := t.TempDir()

	cmd := exec.Command(e2eBinPath, "mcp", "serve",
		"--endpoint", mockServer.URL,
		"--token", "lot_pat_e2e_test",
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", homeDir))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start mcp server process: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)

	// 1. Send JSON-RPC initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e-test","version":"1.0"}}}` + "\n"
	if _, err := stdin.Write([]byte(initReq)); err != nil {
		t.Fatalf("failed to write initialize request: %v", err)
	}

	// Read initialize response line
	line, err := readWithTimeout(reader, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to read initialize response: %v", err)
	}
	if !strings.Contains(line, "capabilities") || !strings.Contains(line, "angler") {
		t.Fatalf("unexpected initialize response: %s", line)
	}

	// 2. Send notifications/initialized
	initializedNotif := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	if _, err := stdin.Write([]byte(initializedNotif)); err != nil {
		t.Fatalf("failed to write initialized notification: %v", err)
	}

	// 3. Send tools/list request
	toolsReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	if _, err := stdin.Write([]byte(toolsReq)); err != nil {
		t.Fatalf("failed to write tools/list request: %v", err)
	}

	toolsResp, err := readWithTimeout(reader, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to read tools/list response: %v", err)
	}

	expectedTools := []string{
		"angler_sql_query",
		"angler_catalog_inspect",
		"angler_fs_browse",
		"angler_workspace_read_notebook",
		"angler_jobs_trigger",
		"angler_ai_genie",
	}
	for _, expected := range expectedTools {
		if !strings.Contains(toolsResp, expected) {
			t.Fatalf("tools/list response missing %s: %s", expected, toolsResp)
		}
	}

	// 4. Send tools/call request for angler_sql_query
	callReq := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"angler_sql_query","arguments":{"sql":"SELECT 42"}}}` + "\n"
	if _, err := stdin.Write([]byte(callReq)); err != nil {
		t.Fatalf("failed to write tools/call request: %v", err)
	}

	callResp, err := readWithTimeout(reader, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to read tools/call response: %v", err)
	}
	if !strings.Contains(callResp, "lake") && !strings.Contains(callResp, "1") {
		t.Fatalf("unexpected tools/call response: %s", callResp)
	}
}

func readWithTimeout(r *bufio.Reader, timeout time.Duration) (string, error) {
	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)

	go func() {
		line, err := r.ReadString('\n')
		ch <- readResult{line: line, err: err}
	}()

	select {
	case res := <-ch:
		return res.line, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("read timed out after %v", timeout)
	}
}
