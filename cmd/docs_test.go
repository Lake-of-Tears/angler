package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestDocsCommandGeneration(t *testing.T) {
	tempDir := t.TempDir()
	docOutDir := filepath.Join(tempDir, "docs")

	stdout, stderr, code := executeCLIForTest(t, false, "docs", docOutDir)
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	combined := stdout + stderr
	if !strings.Contains(combined, "Generated") {
		t.Errorf("expected success message containing 'Generated', got: %s", combined)
	}

	// Verify generated markdown files
	entries, err := os.ReadDir(docOutDir)
	if err != nil {
		t.Fatalf("failed to read doc dir: %v", err)
	}

	if len(entries) < 10 {
		t.Errorf("expected at least 10 markdown documentation files, got %d", len(entries))
	}

	// Check specific expected doc files
	expectedFiles := []string{
		"angler.md",
		"angler_query.md",
		"angler_mcp.md",
		"angler_dashboards.md",
		"angler_users.md",
		"angler_workspaces.md",
	}

	for _, ef := range expectedFiles {
		filePath := filepath.Join(docOutDir, ef)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Errorf("expected doc file %s to exist: %v", ef, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected doc file %s to be non-empty", ef)
		}
	}
}

func TestDocsCommandWithCompletions(t *testing.T) {
	tempDir := t.TempDir()
	docOutDir := filepath.Join(tempDir, "docs")
	compOutDir := filepath.Join(tempDir, "completions")

	stdout, stderr, code := executeCLIForTest(t, false, "docs", docOutDir, "--completions-dir", compOutDir)
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	combined := stdout + stderr
	if !strings.Contains(combined, "Generated shell autocompletions") {
		t.Errorf("expected autocompletions success message, got: %s", combined)
	}

	expectedCompletions := []string{
		"angler.bash",
		"angler.zsh",
		"angler.fish",
		"angler.ps1",
	}

	for _, cf := range expectedCompletions {
		filePath := filepath.Join(compOutDir, cf)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("expected completion file %s to exist: %v", cf, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected completion file %s to be non-empty", cf)
		}
	}
}

func TestDocsCleanFlag(t *testing.T) {
	tempDir := t.TempDir()
	docOutDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docOutDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create an obsolete markdown file
	staleDoc := filepath.Join(docOutDir, "angler_old_deprecated.md")
	if err := os.WriteFile(staleDoc, []byte("# Obsolete Doc"), 0644); err != nil {
		t.Fatalf("failed to write stale doc: %v", err)
	}

	// Run with clean (default is true)
	_, stderr, code := executeCLIForTest(t, false, "docs", docOutDir, "--clean=true")
	if code != ui.ExitOK {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr)
	}

	if _, err := os.Stat(staleDoc); !os.IsNotExist(err) {
		t.Errorf("expected stale doc %s to be cleaned up", staleDoc)
	}
}
