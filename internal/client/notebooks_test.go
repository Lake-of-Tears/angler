package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestNotebooksClientMethods(t *testing.T) {
	mux := http.NewServeMux()

	// 1. /api/notebooks/files
	mux.HandleFunc("/api/notebooks/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		subpath := r.URL.Query().Get("subpath")
		now := time.Now()
		size := int64(1024)
		resp := NotebookListResponse{
			Workspace: "demo",
			Path:      subpath,
			Items: []NotebookItem{
				{
					Name:         "analytics.ipynb",
					Path:         "analytics.ipynb",
					Type:         "notebook",
					LastModified: &now,
					Size:         &size,
					Writable:     true,
				},
				{
					Name:     "models",
					Path:     "models",
					Type:     "directory",
					Writable: true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 2. /api/notebooks/directories
	mux.HandleFunc("/api/notebooks/directories", func(w http.ResponseWriter, r *http.Request) {
		resp := NotebookDirectoriesResponse{
			Workspace:   "demo",
			Directories: []string{"models", "models/features", "reports"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 3. /api/notebooks/search
	mux.HandleFunc("/api/notebooks/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		resp := NotebookSearchResponse{
			Workspace: "demo",
			Query:     q,
			Items: []NotebookItem{
				{
					Name: "analytics.ipynb",
					Path: "analytics.ipynb",
					Type: "notebook",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 4. /api/notebooks/create
	mux.HandleFunc("/api/notebooks/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["type"] == "directory" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
			return
		}
		resp := NotebookDetail{
			Name: payload["name"],
			Path: payload["name"],
			Type: "notebook",
			Content: &NotebookContent{
				Nbformat:      4,
				NbformatMinor: 5,
				Cells:         []NotebookCell{},
			},
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 5. /api/notebooks/runs
	mux.HandleFunc("/api/notebooks/runs", func(w http.ResponseWriter, r *http.Request) {
		resp := NotebookRunsResponse{
			Workspace: "demo",
			Notebook:  r.URL.Query().Get("path"),
			Runs: []NotebookRunArchive{
				{
					Key:          "runs/run_1.ipynb",
					Filename:     "run_1.ipynb",
					Size:         2048,
					LastModified: "2026-11-08T12:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 6. /api/notebooks/sessions
	mux.HandleFunc("/api/notebooks/sessions", func(w http.ResponseWriter, r *http.Request) {
		resp := NotebookSession{
			ID:   "session-123",
			Path: "analytics.ipynb",
			Name: "analytics",
			Type: "notebook",
			Kernel: map[string]any{
				"id":   "k-456",
				"name": "python3",
			},
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 7. /api/notebooks/kernels/
	mux.HandleFunc("/api/notebooks/kernels/k-456/restart", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "restarted"})
	})
	mux.HandleFunc("/api/notebooks/kernels/k-456/interrupt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "interrupted"})
	})

	// 8. /api/notebooks/analytics.ipynb
	mux.HandleFunc("/api/notebooks/analytics.ipynb", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("download") == "true" {
				w.Header().Set("Content-Type", "application/x-ipynb+json")
				_, _ = w.Write([]byte(`{"nbformat": 4, "cells": []}`))
				return
			}
			resp := NotebookDetail{
				Name: "analytics.ipynb",
				Path: "analytics.ipynb",
				Type: "notebook",
				Content: &NotebookContent{
					Nbformat:      4,
					NbformatMinor: 5,
					Cells: []NotebookCell{
						{
							CellType: "code",
							Source:   "print('hello world')",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodPut:
			resp := NotebookDetail{
				Name: "analytics.ipynb",
				Path: "analytics.ipynb",
				Type: "notebook",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "renamed"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint:  ts.URL,
		Token:     "test-token",
		Workspace: "demo",
	})

	// Test ListWorkspaceNotebooks
	listRes, err := c.ListWorkspaceNotebooks("")
	if err != nil {
		t.Fatalf("ListWorkspaceNotebooks error: %v", err)
	}
	if len(listRes.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(listRes.Items))
	}

	// Test ListWorkspaceDirectories
	dirsRes, err := c.ListWorkspaceDirectories()
	if err != nil {
		t.Fatalf("ListWorkspaceDirectories error: %v", err)
	}
	if len(dirsRes.Directories) != 3 {
		t.Errorf("expected 3 directories, got %d", len(dirsRes.Directories))
	}

	// Test SearchWorkspaceItems
	searchRes, err := c.SearchWorkspaceItems("analytics", "")
	if err != nil {
		t.Fatalf("SearchWorkspaceItems error: %v", err)
	}
	if len(searchRes.Items) != 1 {
		t.Errorf("expected 1 search result, got %d", len(searchRes.Items))
	}

	// Test GetNotebook
	nb, err := c.GetNotebook("analytics.ipynb")
	if err != nil {
		t.Fatalf("GetNotebook error: %v", err)
	}
	if nb.Name != "analytics.ipynb" {
		t.Errorf("expected analytics.ipynb, got %s", nb.Name)
	}

	// Test GetNotebookRaw
	raw, err := c.GetNotebookRaw("analytics.ipynb", true)
	if err != nil {
		t.Fatalf("GetNotebookRaw error: %v", err)
	}
	if !strings.Contains(string(raw), "nbformat") {
		t.Errorf("unexpected raw content: %s", string(raw))
	}

	// Test SaveNotebook
	saved, err := c.SaveNotebook("analytics.ipynb", nb.Content)
	if err != nil {
		t.Fatalf("SaveNotebook error: %v", err)
	}
	if saved.Name != "analytics.ipynb" {
		t.Errorf("unexpected saved name: %s", saved.Name)
	}

	// Test CreateNotebook
	created, err := c.CreateNotebook("", "new.ipynb")
	if err != nil {
		t.Fatalf("CreateNotebook error: %v", err)
	}
	if created.Name != "new.ipynb" {
		t.Errorf("expected new.ipynb, got %s", created.Name)
	}

	// Test CreateWorkspaceDirectory
	if err := c.CreateWorkspaceDirectory("new_folder"); err != nil {
		t.Fatalf("CreateWorkspaceDirectory error: %v", err)
	}

	// Test RenameWorkspaceItem
	if err := c.RenameWorkspaceItem("analytics.ipynb", "renamed.ipynb"); err != nil {
		t.Fatalf("RenameWorkspaceItem error: %v", err)
	}

	// Test DeleteWorkspaceItem
	if err := c.DeleteWorkspaceItem("analytics.ipynb"); err != nil {
		t.Fatalf("DeleteWorkspaceItem error: %v", err)
	}

	// Test ListNotebookRuns
	runs, err := c.ListNotebookRuns("analytics.ipynb")
	if err != nil {
		t.Fatalf("ListNotebookRuns error: %v", err)
	}
	if len(runs.Runs) != 1 {
		t.Errorf("expected 1 run archive, got %d", len(runs.Runs))
	}

	// Test CreateNotebookSession
	sess, err := c.CreateNotebookSession("analytics.ipynb", "python3")
	if err != nil {
		t.Fatalf("CreateNotebookSession error: %v", err)
	}
	if sess.ID != "session-123" {
		t.Errorf("expected session-123, got %s", sess.ID)
	}

	// Test Kernel Control
	if err := c.RestartKernel("k-456"); err != nil {
		t.Fatalf("RestartKernel error: %v", err)
	}
	if err := c.InterruptKernel("k-456"); err != nil {
		t.Fatalf("InterruptKernel error: %v", err)
	}
}

func TestNotebookConverters(t *testing.T) {
	pythonCode := `# %% [markdown]
# Initial markdown header

# %%
import duckdb
con = duckdb.connect()
print("connected")
`
	nb := NotebookFromSource(pythonCode, "python")
	if len(nb.Cells) < 2 {
		t.Fatalf("expected at least 2 cells, got %d", len(nb.Cells))
	}
	if nb.Nbformat != 4 {
		t.Errorf("expected nbformat 4, got %d", nb.Nbformat)
	}

	outSource := NotebookToSource(nb)
	if !strings.Contains(outSource, "duckdb") {
		t.Errorf("expected duckdb in source, got %s", outSource)
	}

	jsonBytes, err := NotebookToJSON(nb)
	if err != nil {
		t.Fatalf("NotebookToJSON error: %v", err)
	}
	parsedNb, err := ParseNotebookContent(jsonBytes)
	if err != nil {
		t.Fatalf("ParseNotebookContent error: %v", err)
	}
	if len(parsedNb.Cells) != len(nb.Cells) {
		t.Errorf("cell count mismatch: expected %d, got %d", len(nb.Cells), len(parsedNb.Cells))
	}
}
