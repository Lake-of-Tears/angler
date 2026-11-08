package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var (
	docsCompletionsDir string
	docsClean          bool
)

var docsCmd = &cobra.Command{
	Use:     "docs [output-dir]",
	Aliases: []string{"doc"},
	Short:   "Generate CLI reference documentation and shell autocompletions",
	Long: `Generates complete markdown reference documentation for all Angler CLI commands
and subcommands using Cobra's markdown documentation generator.

Optionally generates shell completion scripts for bash, zsh, fish, and PowerShell.

Examples:
  angler docs
  angler docs ./docs/reference
  angler docs ./docs/reference --completions-dir ./completions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "docs/reference"
		if len(args) > 0 && args[0] != "" {
			targetDir = args[0]
		}

		// Ensure target directory exists
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create documentation directory %s: %w", targetDir, err)
		}

		// Optional clean of existing files
		if docsClean {
			entries, _ := os.ReadDir(targetDir)
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
					_ = os.Remove(filepath.Join(targetDir, entry.Name()))
				}
			}
		}

		// Disable auto-generated tag timestamp to prevent noisy git diffs
		rootCmd.DisableAutoGenTag = true

		// Generate markdown documentation tree
		if err := doc.GenMarkdownTree(rootCmd, targetDir); err != nil {
			return fmt.Errorf("failed to generate markdown documentation: %w", err)
		}

		// Count generated markdown files
		entries, _ := os.ReadDir(targetDir)
		docCount := 0
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
				docCount++
			}
		}

		ui.PrintSuccess("Generated %d markdown documentation pages in %s", docCount, targetDir)

		// Optionally generate shell completion files
		if docsCompletionsDir != "" {
			if err := os.MkdirAll(docsCompletionsDir, 0755); err != nil {
				return fmt.Errorf("failed to create completions directory: %w", err)
			}

			// 1. Bash
			bashPath := filepath.Join(docsCompletionsDir, "angler.bash")
			if f, err := os.Create(bashPath); err == nil {
				_ = rootCmd.GenBashCompletion(f)
				f.Close()
			}

			// 2. Zsh
			zshPath := filepath.Join(docsCompletionsDir, "angler.zsh")
			if f, err := os.Create(zshPath); err == nil {
				_ = rootCmd.GenZshCompletion(f)
				f.Close()
			}

			// 3. Fish
			fishPath := filepath.Join(docsCompletionsDir, "angler.fish")
			if f, err := os.Create(fishPath); err == nil {
				_ = rootCmd.GenFishCompletion(f, true)
				f.Close()
			}

			// 4. PowerShell
			psPath := filepath.Join(docsCompletionsDir, "angler.ps1")
			if f, err := os.Create(psPath); err == nil {
				_ = rootCmd.GenPowerShellCompletionWithDesc(f)
				f.Close()
			}

			ui.PrintSuccess("Generated shell autocompletions (bash, zsh, fish, ps1) in %s", docsCompletionsDir)
		}

		return nil
	},
}

func init() {
	docsCmd.Flags().StringVar(&docsCompletionsDir, "completions-dir", "", "Directory to generate shell completion scripts (bash, zsh, fish, ps1)")
	docsCmd.Flags().BoolVar(&docsClean, "clean", true, "Clean existing markdown files in output directory before generation")

	rootCmd.AddCommand(docsCmd)
}
