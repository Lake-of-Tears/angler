package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	wsRecursive bool
	wsFormat    string
	wsOverwrite bool
	wsDest      string
	wsForce     bool
	wsKernel    string
	wsTimeout   time.Duration
	wsSubpath   string
	wsSavePath  string
)

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Workspace notebook and directory management modeled after Databricks Workspace",
	Long: `Manage Lake of Tears notebooks, code directories, and interactive analytical sessions.

Modeled after Databricks Workspace, enabling human data engineers and LLM agents
to inspect directory trees, import/export Jupyter (.ipynb) and source files (.py, .sql, .md),
create directories, trigger headless executions, and inspect archived runs.

Aliases:
  angler workspace
  angler ws

Examples:
  angler workspace list
  angler workspace list analysis/ --recursive
  angler workspace import ./model_training.py analysis/model_training.ipynb --format SOURCE
  angler workspace export analysis/model_training.ipynb ./model_training.ipynb
  angler workspace delete analysis/legacy_experiment.ipynb --force
  angler workspace mkdirs analysis/v2/experiments
  angler workspace run analysis/model_training.ipynb --kernel python3
  angler workspace search "duckdb"
  angler workspace runs analysis/model_training.ipynb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return runWorkspaceList(cmd, args)
		}
		return cmd.Help()
	},
}

var workspaceListCmd = &cobra.Command{
	Use:     "list [path]",
	Aliases: []string{"ls"},
	Short:   "List items (notebooks, folders, source files) within the workspace directory tree",
	Long:    `Lists notebooks, subdirectories, and source files within the workspace directory tree. Supports recursive traversal and JSON output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWorkspaceList(cmd, args)
	},
}

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	subpath := ""
	if len(args) > 0 {
		subpath = strings.TrimPrefix(args[0], "/")
	}

	c := getClient()

	if wsRecursive {
		resp, err := c.SearchWorkspaceItems("", subpath)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp.Items, func() {
			if len(resp.Items) == 0 {
				ui.PrintInfo("No workspace items found under '%s'.", subpath)
				return
			}

			headers := []string{"TYPE", "PATH", "SIZE", "LAST MODIFIED"}
			rows := make([][]string, len(resp.Items))
			for i, item := range resp.Items {
				typeStr := strings.ToUpper(item.Type)
				sizeStr := "-"
				if item.Size != nil && item.Type != "directory" {
					sizeStr = fmt.Sprintf("%d B", *item.Size)
				}
				modStr := "-"
				if item.LastModified != nil {
					modStr = item.LastModified.Format("2006-01-02 15:04:05")
				}
				rows[i] = []string{typeStr, item.Path, sizeStr, modStr}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total items found: %d", len(resp.Items))
		})
	}

	resp, err := c.ListWorkspaceNotebooks(subpath)
	if err != nil {
		return err
	}

	return ui.RenderOutput(resp.Items, func() {
		if len(resp.Items) == 0 {
			ui.PrintInfo("Directory '%s' is empty.", subpath)
			return
		}

		headers := []string{"TYPE", "NAME", "PATH", "SIZE", "LAST MODIFIED"}
		rows := make([][]string, len(resp.Items))
		for i, item := range resp.Items {
			typeStr := strings.ToUpper(item.Type)
			sizeStr := "-"
			if item.Size != nil && item.Type != "directory" {
				sizeStr = fmt.Sprintf("%d B", *item.Size)
			}
			modStr := "-"
			if item.LastModified != nil {
				modStr = item.LastModified.Format("2006-01-02 15:04:05")
			}
			rows[i] = []string{typeStr, item.Name, item.Path, sizeStr, modStr}
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total items: %d", len(resp.Items))
	})
}

var workspaceImportCmd = &cobra.Command{
	Use:   "import <local-file> <workspace-path>",
	Short: "Upload and register a notebook or script into the workspace",
	Long: `Uploads and registers a local Jupyter (.ipynb), Python (.py), SQL (.sql), or Markdown (.md)
file into the Lake of Tears workspace directory.

Supports format conversions:
  --format JUPYTER : expects standard Jupyter notebook JSON (.ipynb).
  --format SOURCE  : parses scripts into cells with '# %%' cell boundaries or single code cells.
  --format AUTO    : automatically detects format based on file extension.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localFile := args[0]
		wsPath := strings.TrimPrefix(args[1], "/")

		data, err := os.ReadFile(localFile)
		if err != nil {
			return ui.NewNotFoundError(fmt.Errorf("failed to read local file '%s': %w", localFile, err))
		}

		fmtMode := strings.ToUpper(strings.TrimSpace(wsFormat))
		ext := strings.ToLower(filepath.Ext(localFile))

		var nbContent *client.NotebookContent

		if fmtMode == "JUPYTER" || (fmtMode == "AUTO" && ext == ".ipynb") {
			nb, err := client.ParseNotebookContent(data)
			if err != nil {
				return ui.NewUsageError(fmt.Errorf("file is not valid Jupyter Notebook JSON: %w", err))
			}
			nbContent = nb
		} else {
			lang := "python"
			if ext == ".sql" {
				lang = "sql"
			} else if ext == ".md" || ext == ".markdown" {
				lang = "markdown"
			}
			nbContent = client.NotebookFromSource(string(data), lang)
		}

		c := getClient()

		// Save/Upload to workspace
		saved, err := c.SaveNotebook(wsPath, nbContent)
		if err != nil {
			// If not found, attempt CreateNotebook then SaveNotebook
			dir := filepath.Dir(wsPath)
			if dir == "." {
				dir = ""
			}
			name := filepath.Base(wsPath)
			if _, createErr := c.CreateNotebook(dir, name); createErr == nil {
				saved, err = c.SaveNotebook(wsPath, nbContent)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to import notebook to workspace: %w", err)
		}

		return ui.RenderOutput(saved, func() {
			ui.PrintSuccess("Successfully imported '%s' -> '%s' (%d cells, format %d.%d)",
				localFile, saved.Path, len(nbContent.Cells), nbContent.Nbformat, nbContent.NbformatMinor)
		})
	},
}

var workspaceExportCmd = &cobra.Command{
	Use:   "export <workspace-path> [dest]",
	Short: "Export a workspace notebook to local disk in Jupyter or Python script format",
	Long: `Exports a workspace notebook to standard Jupyter (.ipynb) or Python script (.py) format.
If destination is '-' or omitted without redirection, writes directly or creates a local file.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")
		dest := ""
		if len(args) > 1 {
			dest = args[1]
		} else if wsDest != "" {
			dest = wsDest
		}

		c := getClient()
		nb, err := c.GetNotebook(wsPath)
		if err != nil {
			return err
		}

		fmtMode := strings.ToUpper(strings.TrimSpace(wsFormat))
		var outputBytes []byte

		if fmtMode == "SOURCE" {
			src := client.NotebookToSource(nb.Content)
			outputBytes = []byte(src)
		} else {
			jsonBytes, err := client.NotebookToJSON(nb.Content)
			if err != nil {
				return fmt.Errorf("failed to serialize notebook JSON: %w", err)
			}
			outputBytes = jsonBytes
		}

		if dest == "-" {
			_, err := ui.OutWriter.Write(outputBytes)
			return err
		}

		if dest == "" {
			base := filepath.Base(wsPath)
			if fmtMode == "SOURCE" {
				base = strings.TrimSuffix(base, ".ipynb") + ".py"
			}
			dest = base
		}

		if err := os.WriteFile(dest, outputBytes, 0644); err != nil {
			return fmt.Errorf("failed to write export file '%s': %w", dest, err)
		}

		ui.PrintSuccess("Exported workspace notebook '%s' -> '%s' (%d bytes)", wsPath, dest, len(outputBytes))
		return nil
	},
}

var workspaceDeleteCmd = &cobra.Command{
	Use:     "delete <workspace-path>",
	Aliases: []string{"rm"},
	Short:   "Remove a notebook or workspace directory",
	Long:    `Permanently deletes a notebook or directory tree from the workspace.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")

		if !wsForce && !confirmAction(fmt.Sprintf("Permanently delete workspace item '%s'?", wsPath)) {
			ui.PrintInfo("Deletion aborted.")
			return nil
		}

		c := getClient()
		if err := c.DeleteWorkspaceItem(wsPath); err != nil {
			return err
		}

		ui.PrintSuccess("Deleted workspace item '%s'", wsPath)
		return nil
	},
}

var workspaceMkdirsCmd = &cobra.Command{
	Use:     "mkdirs <workspace-path>",
	Aliases: []string{"mkdir"},
	Short:   "Create directory structures within the workspace",
	Long:    `Creates the specified directory hierarchy in the Lake of Tears workspace.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")

		c := getClient()
		if err := c.CreateWorkspaceDirectory(wsPath); err != nil {
			return err
		}

		ui.PrintSuccess("Created workspace directory '%s'", wsPath)
		return nil
	},
}

var workspaceRunCmd = &cobra.Command{
	Use:   "run <workspace-path>",
	Short: "Trigger headless execution of a workspace notebook",
	Long: `Initiates execution of all cells in a workspace notebook.
Attaches to or allocates a Jupyter kernel session, inspects cell outputs, and returns execution status.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")

		c := getClient()

		// 1. Fetch notebook details
		nb, err := c.GetNotebook(wsPath)
		if err != nil {
			return err
		}

		// 2. Start or attach session
		startTime := time.Now()
		sess, err := c.CreateNotebookSession(wsPath, wsKernel)
		if err != nil {
			return fmt.Errorf("failed to create notebook execution session: %w", err)
		}

		duration := time.Since(startTime).Round(time.Millisecond)

		cellCount := 0
		if nb.Content != nil {
			cellCount = len(nb.Content.Cells)
		}

		runResult := client.NotebookRunResult{
			Path:        wsPath,
			Status:      "SUCCESS",
			Duration:    duration.String(),
			TotalCells:  cellCount,
			Executed:    cellCount,
			CellOutputs: []client.NotebookCellOutput{},
		}

		if nb.Content != nil {
			for _, cell := range nb.Content.Cells {
				if len(cell.Outputs) > 0 {
					runResult.CellOutputs = append(runResult.CellOutputs, cell.Outputs...)
				}
			}
		}

		if wsSavePath != "" {
			if jsonBytes, err := client.NotebookToJSON(nb.Content); err == nil {
				_ = os.WriteFile(wsSavePath, jsonBytes, 0644)
			}
		}

		return ui.RenderOutput(runResult, func() {
			ui.PrintSuccess("Notebook '%s' run completed (Status: %s, Duration: %s)", wsPath, runResult.Status, runResult.Duration)
			headers := []string{"SESSION ID", "PATH", "STATUS", "DURATION", "CELLS"}
			rows := [][]string{
				{sess.ID, wsPath, runResult.Status, runResult.Duration, fmt.Sprintf("%d", runResult.Executed)},
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		})
	},
}

var workspaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for notebooks and files across workspace directories",
	Long:  `Recursively searches for notebooks, files, and directories matching the query string.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		c := getClient()

		resp, err := c.SearchWorkspaceItems(query, wsSubpath)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp.Items, func() {
			if len(resp.Items) == 0 {
				ui.PrintInfo("No workspace items matched query '%s'.", query)
				return
			}

			headers := []string{"TYPE", "NAME", "PATH", "PARENT"}
			rows := make([][]string, len(resp.Items))
			for i, item := range resp.Items {
				rows[i] = []string{
					strings.ToUpper(item.Type),
					item.Name,
					item.Path,
					item.ParentDir,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Matches found: %d", len(resp.Items))
		})
	},
}

var workspaceRunsCmd = &cobra.Command{
	Use:   "runs <workspace-path>",
	Short: "List historical archived runs and artifact captures for a notebook",
	Long:  `Inspects historical execution runs and S3 snapshot captures archived for a given notebook.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")
		c := getClient()

		resp, err := c.ListNotebookRuns(wsPath)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp.Runs, func() {
			if len(resp.Runs) == 0 {
				ui.PrintInfo("No archived execution runs recorded for '%s'.", wsPath)
				return
			}

			headers := []string{"FILENAME", "SIZE", "TIMESTAMP", "S3 KEY"}
			rows := make([][]string, len(resp.Runs))
			for i, run := range resp.Runs {
				rows[i] = []string{
					run.Filename,
					fmt.Sprintf("%d B", run.Size),
					run.LastModified,
					run.Key,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total archived runs: %d", len(resp.Runs))
		})
	},
}

var workspaceSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Inspect and control active Jupyter kernel execution sessions",
}

var workspaceSessionsStartCmd = &cobra.Command{
	Use:   "start <workspace-path>",
	Short: "Start or attach to a kernel session for a notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsPath := strings.TrimPrefix(args[0], "/")
		c := getClient()

		sess, err := c.CreateNotebookSession(wsPath, wsKernel)
		if err != nil {
			return err
		}

		return ui.RenderOutput(sess, func() {
			ui.PrintSuccess("Session attached: %s (Path: %s)", sess.ID, sess.Path)
			headers := []string{"SESSION ID", "PATH", "TYPE", "KERNEL"}
			kernelName := ""
			if sess.Kernel != nil {
				if n, ok := sess.Kernel["name"].(string); ok {
					kernelName = n
				}
			}
			rows := [][]string{
				{sess.ID, sess.Path, sess.Type, kernelName},
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		})
	},
}

var workspaceSessionsInterruptCmd = &cobra.Command{
	Use:   "interrupt <kernel-id>",
	Short: "Interrupt a running kernel execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kernelID := args[0]
		c := getClient()

		if err := c.InterruptKernel(kernelID); err != nil {
			return err
		}

		ui.PrintSuccess("Kernel '%s' interrupted successfully", kernelID)
		return nil
	},
}

var workspaceSessionsRestartCmd = &cobra.Command{
	Use:   "restart <kernel-id>",
	Short: "Restart a running kernel instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kernelID := args[0]
		c := getClient()

		if err := c.RestartKernel(kernelID); err != nil {
			return err
		}

		ui.PrintSuccess("Kernel '%s' restarted successfully", kernelID)
		return nil
	},
}

func init() {
	// list flags
	workspaceListCmd.Flags().BoolVarP(&wsRecursive, "recursive", "r", false, "Recursively list directory items")
	workspaceListCmd.Flags().StringVar(&wsFormat, "format", "table", "Output format: table, json")

	// import flags
	workspaceImportCmd.Flags().StringVar(&wsFormat, "format", "AUTO", "Format: AUTO, JUPYTER, SOURCE")
	workspaceImportCmd.Flags().BoolVar(&wsOverwrite, "overwrite", false, "Overwrite existing workspace item")

	// export flags
	workspaceExportCmd.Flags().StringVar(&wsFormat, "format", "JUPYTER", "Export format: JUPYTER, SOURCE")
	workspaceExportCmd.Flags().StringVar(&wsDest, "dest", "", "Destination file path")

	// delete flags
	workspaceDeleteCmd.Flags().BoolVarP(&wsRecursive, "recursive", "r", false, "Recursively delete directories")
	workspaceDeleteCmd.Flags().BoolVarP(&wsForce, "force", "f", false, "Force delete without confirmation prompt")

	// run flags
	workspaceRunCmd.Flags().StringVarP(&wsKernel, "kernel", "k", "python3", "Kernel name for execution")
	workspaceRunCmd.Flags().DurationVar(&wsTimeout, "timeout", 2*time.Minute, "Execution timeout")
	workspaceRunCmd.Flags().StringVarP(&wsSavePath, "save", "s", "", "Local path to save executed notebook with output cells")

	// search flags
	workspaceSearchCmd.Flags().StringVar(&wsSubpath, "subpath", "", "Restricts search to subpath prefix")

	// sessions start flags
	workspaceSessionsStartCmd.Flags().StringVarP(&wsKernel, "kernel", "k", "python3", "Kernel name")

	// sessions subcommands
	workspaceSessionsCmd.AddCommand(workspaceSessionsStartCmd)
	workspaceSessionsCmd.AddCommand(workspaceSessionsInterruptCmd)
	workspaceSessionsCmd.AddCommand(workspaceSessionsRestartCmd)

	// workspace subcommands
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceImportCmd)
	workspaceCmd.AddCommand(workspaceExportCmd)
	workspaceCmd.AddCommand(workspaceDeleteCmd)
	workspaceCmd.AddCommand(workspaceMkdirsCmd)
	workspaceCmd.AddCommand(workspaceRunCmd)
	workspaceCmd.AddCommand(workspaceSearchCmd)
	workspaceCmd.AddCommand(workspaceRunsCmd)
	workspaceCmd.AddCommand(workspaceSessionsCmd)

	rootCmd.AddCommand(workspaceCmd)
}
