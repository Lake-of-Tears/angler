package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetWorkspaceFlags() {
	wsRecursive = false
	wsFormat = "table"
	wsOverwrite = false
	wsDest = ""
	wsForce = false
	wsKernel = "python3"
	wsTimeout = 2 * time.Minute
	wsSubpath = ""
	wsSavePath = ""
	jsonOutput = false
	outputFlag = ""
}

func TestWorkspaceCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundWorkspace bool
	for _, c := range commands {
		if c.Name() == "workspace" {
			foundWorkspace = true
			expectedSubcommands := map[string]bool{
				"list":     false,
				"import":   false,
				"export":   false,
				"delete":   false,
				"mkdirs":   false,
				"run":      false,
				"search":   false,
				"runs":     false,
				"sessions": false,
			}
			for _, sub := range c.Commands() {
				if _, ok := expectedSubcommands[sub.Name()]; ok {
					expectedSubcommands[sub.Name()] = true
				}
			}
			for sub, found := range expectedSubcommands {
				if !found {
					t.Errorf("expected subcommand %q registered under workspace", sub)
				}
			}
		}
	}
	if !foundWorkspace {
		t.Fatal("workspace command not registered on rootCmd")
	}
}

func setupWorkspaceMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	now := time.Now()
	size := int64(2048)

	// List notebooks: /api/notebooks/files
	mux.HandleFunc("/api/notebooks/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.NotebookListResponse{
			Workspace: "lake-ws",
			Path:      r.URL.Query().Get("subpath"),
			Items: []client.NotebookItem{
				{
					Name:         "analytics.ipynb",
					Path:         "analysis/analytics.ipynb",
					Type:         "notebook",
					LastModified: &now,
					Size:         &size,
					Writable:     true,
				},
				{
					Name:     "experiments",
					Path:     "analysis/experiments",
					Type:     "directory",
					Writable: true,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Search items: /api/notebooks/search
	mux.HandleFunc("/api/notebooks/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query().Get("q")
		resp := client.NotebookSearchResponse{
			Workspace: "lake-ws",
			Query:     q,
			Subpath:   r.URL.Query().Get("subpath"),
			Items: []client.NotebookItem{
				{
					Name:      "analytics.ipynb",
					Path:      "analysis/analytics.ipynb",
					Type:      "notebook",
					ParentDir: "analysis",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Create item: /api/notebooks/create
	mux.HandleFunc("/api/notebooks/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.WriteHeader(http.StatusCreated)
		if payload["type"] == "directory" {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
			return
		}
		resp := client.NotebookDetail{
			Name: payload["name"],
			Path: payload["name"],
			Type: "notebook",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Single notebook operations: /api/notebooks/analysis/analytics.ipynb
	mux.HandleFunc("/api/notebooks/analysis/analytics.ipynb", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			resp := client.NotebookDetail{
				Name: "analytics.ipynb",
				Path: "analysis/analytics.ipynb",
				Type: "notebook",
				Content: &client.NotebookContent{
					Nbformat:      4,
					NbformatMinor: 5,
					Cells: []client.NotebookCell{
						{
							CellType: "markdown",
							Source:   "# Analytics Title\n",
						},
						{
							CellType: "code",
							Source:   "print('testing execution')",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := client.NotebookDetail{
				Name: "analytics.ipynb",
				Path: "analysis/analytics.ipynb",
				Type: "notebook",
			}
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Sessions: /api/notebooks/sessions
	mux.HandleFunc("/api/notebooks/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := client.NotebookSession{
			ID:   "sess-test-99",
			Path: "analysis/analytics.ipynb",
			Name: "analytics",
			Type: "notebook",
			Kernel: map[string]any{
				"id":   "k-99",
				"name": "python3",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Runs: /api/notebooks/runs
	mux.HandleFunc("/api/notebooks/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := client.NotebookRunsResponse{
			Workspace: "lake-ws",
			Notebook:  r.URL.Query().Get("path"),
			Runs: []client.NotebookRunArchive{
				{
					Key:          "runs/analysis/analytics/run_01.ipynb",
					Filename:     "run_01.ipynb",
					Size:         4096,
					LastModified: "2026-11-08T14:30:00Z",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Kernels
	mux.HandleFunc("/api/notebooks/kernels/k-99/interrupt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "interrupted"})
	})
	mux.HandleFunc("/api/notebooks/kernels/k-99/restart", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "restarted"})
	})

	return httptest.NewServer(mux)
}

func TestWorkspaceList(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	// 1. Table format list
	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "list", "analysis", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "analytics.ipynb") {
		t.Errorf("expected analytics.ipynb in list output: %s", stdout)
	}
	if !strings.Contains(stdout, "experiments") {
		t.Errorf("expected experiments in list output: %s", stdout)
	}

	// 2. Recursive search list
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "ls", "analysis", "--recursive", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "analysis/analytics.ipynb") {
		t.Errorf("expected analysis/analytics.ipynb in recursive search: %s", stdout)
	}

	// 3. JSON format
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "workspace", "ls", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var items []client.NotebookItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("failed to decode JSON list items: %v\nstdout: %s", err, stdout)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items in JSON output, got %d", len(items))
	}
}

func TestWorkspaceImportAndExport(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	tmpDir := t.TempDir()
	localScript := filepath.Join(tmpDir, "local_test.py")
	scriptContent := `# %% [markdown]
# Test Section

# %%
import math
print(math.pi)
`
	if err := os.WriteFile(localScript, []byte(scriptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Import script as SOURCE
	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "import", localScript, "analysis/analytics.ipynb", "--format", "SOURCE", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected import exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Successfully imported") {
		t.Errorf("expected import confirmation in stdout: %s", stdout)
	}

	// 2. Export as JUPYTER to local file
	resetWorkspaceFlags()
	exportNb := filepath.Join(tmpDir, "exported.ipynb")
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "export", "analysis/analytics.ipynb", exportNb, "--format", "JUPYTER", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected export exit 0, got %d (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(exportNb); err != nil {
		t.Errorf("expected exported notebook file to exist: %v", err)
	}

	// 3. Export as SOURCE to stdout (-)
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "export", "analysis/analytics.ipynb", "-", "--format", "SOURCE", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected stdout export exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "print('testing execution')") {
		t.Errorf("expected python source in stdout: %s", stdout)
	}
}

func TestWorkspaceDeleteAndMkdirs(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	// 1. Mkdirs
	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "mkdirs", "analysis/v2/models", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected mkdirs exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Created workspace directory") {
		t.Errorf("expected directory creation confirmation: %s", stdout)
	}

	// 2. Delete with --force
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "delete", "analysis/analytics.ipynb", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected delete exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Deleted workspace item") {
		t.Errorf("expected delete confirmation: %s", stdout)
	}
}

func TestWorkspaceRunAndRuns(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	// 1. Run headless execution
	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "run", "analysis/analytics.ipynb", "--kernel", "python3", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected run exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "run completed") {
		t.Errorf("expected run completed confirmation: %s", stdout)
	}
	if !strings.Contains(stdout, "sess-test-99") {
		t.Errorf("expected session id in table: %s", stdout)
	}

	// 2. Runs list
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "runs", "analysis/analytics.ipynb", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected runs exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "run_01.ipynb") {
		t.Errorf("expected run_01.ipynb in runs output: %s", stdout)
	}
}

func TestWorkspaceSessions(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	// 1. Start session
	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "sessions", "start", "analysis/analytics.ipynb", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected sessions start exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "sess-test-99") {
		t.Errorf("expected session sess-test-99 in stdout: %s", stdout)
	}

	// 2. Interrupt kernel
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "sessions", "interrupt", "k-99", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected interrupt exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "interrupted successfully") {
		t.Errorf("expected interrupt success message: %s", stdout)
	}

	// 3. Restart kernel
	resetWorkspaceFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspace", "sessions", "restart", "k-99", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected restart exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "restarted successfully") {
		t.Errorf("expected restart success message: %s", stdout)
	}
}

func TestWorkspaceSearch(t *testing.T) {
	ts := setupWorkspaceMockServer(t)
	defer ts.Close()

	resetWorkspaceFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspace", "search", "analytics", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected search exit 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "analytics.ipynb") {
		t.Errorf("expected analytics.ipynb in search output: %s", stdout)
	}
}
