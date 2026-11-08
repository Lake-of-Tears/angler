package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetFSTestFlags() {
	fsRecursive = false
	fsAbsolute = false
	fsLimit = 100
	fsOffset = 0
	fsLength = 0
	fsUploadPrefix = ""
	fsForceDelete = false
	fsForceDownload = false
	jsonOutput = false
	outputFlag = ""
}

func executeFSCommand(t *testing.T, endpointURL string, args ...string) (string, string, int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut := ui.OutWriter
	origErr := ui.ErrWriter
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	stdoutTTY := false
	ui.SetTTYOverrides(&stdoutTTY, &stdoutTTY)

	resetFSTestFlags()

	defer func() {
		ui.OutWriter = origOut
		ui.ErrWriter = origErr
		ui.SetTTYOverrides(nil, nil)
		rootCmd.SetArgs(nil)
		endpoint = ""
		token = ""
	}()

	cmdArgs := []string{"--endpoint", endpointURL, "--token", "test-token"}
	cmdArgs = append(cmdArgs, args...)

	rootCmd.SetArgs(cmdArgs)
	code := ExecuteRoot()
	return outBuf.String(), errBuf.String(), code
}

func TestFSCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundFS bool
	for _, c := range commands {
		if c.Name() == "fs" {
			foundFS = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			expected := []string{"ls", "cat", "cp", "stat", "upload", "download", "rm", "mkdir", "stats"}
			for _, exp := range expected {
				if !subMap[exp] {
					t.Errorf("expected subcommand %q under fsCmd", exp)
				}
			}
		}
	}

	if !foundFS {
		t.Errorf("expected 'fs' command to be registered on rootCmd")
	}
}

func TestFSLs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/storage/browse" && r.Method == http.MethodGet {
			prefix := r.URL.Query().Get("prefix")
			if prefix == "silver" {
				json.NewEncoder(w).Encode(client.StorageBrowseResponse{
					Bucket:       "lakehouse",
					Prefix:       "silver",
					TotalFolders: 1,
					TotalFiles:   1,
					Folders: []client.StorageFolder{
						{Name: "customers", Prefix: "silver/customers/"},
					},
					Files: []client.StorageFile{
						{
							Key:           "silver/orders.parquet",
							Name:          "orders.parquet",
							Size:          2048576,
							SizeFormatted: "2.0 MB",
							IsParquet:     true,
						},
					},
				})
				return
			}
			if prefix == "silver/customers/" {
				json.NewEncoder(w).Encode(client.StorageBrowseResponse{
					Bucket:       "lakehouse",
					Prefix:       "silver/customers/",
					TotalFolders: 0,
					TotalFiles:   1,
					Files: []client.StorageFile{
						{
							Key:           "silver/customers/part-0.parquet",
							RelKey:        "silver/customers/part-0.parquet",
							Name:          "part-0.parquet",
							Size:          1024,
							SizeFormatted: "1.0 KB",
							IsParquet:     true,
						},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(client.StorageBrowseResponse{
				Bucket:       "lakehouse",
				Prefix:       "",
				TotalFolders: 0,
				TotalFiles:   0,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	// 1. Standard ls
	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "ls", "silver")
	if code != ui.ExitOK {
		t.Fatalf("fs ls failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "orders.parquet") {
		t.Errorf("expected orders.parquet in output, got:\n%s", stdout)
	}

	// 2. Recursive ls
	stdout, stderr, code = executeFSCommand(t, ts.URL, "fs", "ls", "silver", "-r")
	if code != ui.ExitOK {
		t.Fatalf("fs ls -r failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "part-0.parquet") {
		t.Errorf("expected part-0.parquet in recursive output, got:\n%s", stdout)
	}
}

func TestFSCat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/storage/objects/configs/app.json/stream" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"database": "duckdb", "threads": 4}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "cat", "lot://configs/app.json")
	if code != ui.ExitOK {
		t.Fatalf("fs cat failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, `"database": "duckdb"`) {
		t.Errorf("expected file content, got:\n%s", stdout)
	}
}

func TestFSCp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/storage/upload" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.StorageUploadResponse{
				Key:    "bronze/sample.txt",
				Size:   11,
				Status: "uploaded",
			})
		case r.URL.Path == "/api/storage/objects/silver/export.csv/stream" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "text/csv")
			w.Write([]byte("id,val\n1,test\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	tmpDir := t.TempDir()

	// 1. Local -> Remote copy (upload)
	localFile := filepath.Join(tmpDir, "sample.txt")
	if err := os.WriteFile(localFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write local test file: %v", err)
	}

	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "cp", localFile, "lot://bronze/sample.txt")
	if code != ui.ExitOK {
		t.Fatalf("fs cp upload failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Uploaded") && !strings.Contains(stdout, "Uploaded") {
		t.Errorf("expected upload confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 2. Remote -> Local copy (download)
	downloadTarget := filepath.Join(tmpDir, "downloaded_export.csv")
	stdout, stderr, code = executeFSCommand(t, ts.URL, "fs", "cp", "lot://silver/export.csv", downloadTarget)
	if code != ui.ExitOK {
		t.Fatalf("fs cp download failed: code %d, stderr: %s", code, stderr)
	}
	downloadedBytes, err := os.ReadFile(downloadTarget)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(downloadedBytes) != "id,val\n1,test\n" {
		t.Errorf("unexpected downloaded content: %s", string(downloadedBytes))
	}
}

func TestFSStat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/storage/objects/gold/events.parquet/metadata" {
			rowCount := int64(100000)
			colCount := 8
			json.NewEncoder(w).Encode(client.StorageMetadataResponse{
				Key:           "gold/events.parquet",
				Size:          5242880,
				SizeFormatted: "5.0 MB",
				ContentType:   "application/octet-stream",
				ETag:          "etag-stat-123",
				IsParquet:     true,
				RowCount:      &rowCount,
				ColumnCount:   &colCount,
				Compression:   "ZSTD",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "stat", "gold/events.parquet")
	if code != ui.ExitOK {
		t.Fatalf("fs stat failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "gold/events.parquet") || !strings.Contains(stdout, "100000") {
		t.Errorf("expected object and row count in output, got:\n%s", stdout)
	}
}

func TestFSUploadAndDownload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/storage/upload" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.StorageUploadResponse{
				Key:    "models/churn.onnx",
				Size:   1024,
				Status: "uploaded",
			})
		case r.URL.Path == "/api/storage/objects/models/churn.onnx/stream" && r.Method == http.MethodGet:
			w.Write([]byte("onnx-model-binary-data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "churn.onnx")
	_ = os.WriteFile(modelFile, []byte("onnx-model-binary-data"), 0644)

	// 1. Upload command
	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "upload", modelFile, "models/churn.onnx")
	if code != ui.ExitOK {
		t.Fatalf("fs upload failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Uploaded") && !strings.Contains(stdout, "Uploaded") {
		t.Errorf("expected upload confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 2. Download command
	targetFile := filepath.Join(tmpDir, "downloaded_model.onnx")
	stdout, stderr, code = executeFSCommand(t, ts.URL, "fs", "download", "models/churn.onnx", targetFile)
	if code != ui.ExitOK {
		t.Fatalf("fs download failed: code %d, stderr: %s", code, stderr)
	}
	content, _ := os.ReadFile(targetFile)
	if string(content) != "onnx-model-binary-data" {
		t.Errorf("unexpected downloaded model data: %s", string(content))
	}
}

func TestFSRmAndMkdirAndStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/storage/objects/bronze/temp") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/storage/upload" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(client.StorageUploadResponse{
				Key: "checkpoints/.keep",
			})
		case r.URL.Path == "/api/storage/stats" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(client.StorageStatsResponse{
				Bucket:       "lakehouse",
				Workspace:    "default",
				TotalBytes:   50000000,
				TotalObjects: 150,
				TopLevelPrefixes: map[string]int{
					"bronze": 50,
					"silver": 70,
					"gold":   30,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg = &config.Config{
		CurrentContext: "test",
		Contexts: map[string]config.Context{
			"test": {Endpoint: ts.URL, Token: "test-token", Workspace: "default"},
		},
	}

	// 1. Rm command
	stdout, stderr, code := executeFSCommand(t, ts.URL, "fs", "rm", "bronze/temp", "-r", "--force")
	if code != ui.ExitOK {
		t.Fatalf("fs rm failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Deleted") && !strings.Contains(stdout, "Deleted") {
		t.Errorf("expected delete confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 2. Mkdir command
	stdout, stderr, code = executeFSCommand(t, ts.URL, "fs", "mkdir", "checkpoints")
	if code != ui.ExitOK {
		t.Fatalf("fs mkdir failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Created") && !strings.Contains(stdout, "Created") {
		t.Errorf("expected created confirmation, got stdout: %s, stderr: %s", stdout, stderr)
	}

	// 3. Stats command
	stdout, stderr, code = executeFSCommand(t, ts.URL, "fs", "stats")
	if code != ui.ExitOK {
		t.Fatalf("fs stats failed: code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "150") || !strings.Contains(stdout, "lakehouse") {
		t.Errorf("expected stats metrics in output, got:\n%s", stdout)
	}
}
