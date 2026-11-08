package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	fsRecursive     bool
	fsAbsolute      bool
	fsLimit         int
	fsOffset        int64
	fsLength        int64
	fsUploadPrefix  string
	fsForceDelete   bool
	fsForceDownload bool
)

// fsCmd represents the root of the filesystem and storage command suite.
var fsCmd = &cobra.Command{
	Use:     "fs",
	Aliases: []string{"storage"},
	Short:   "Filesystem and storage management modeled after Databricks FS (alias: storage)",
	Long: `Browse, stream, copy, upload, and remove files stored in Lake of Tears object stores (S3/MinIO)
and workspace volumes.

Modeled after Databricks FS (dbfs) and Volumes, enabling both human engineers and LLM agents
to inspect raw datasets, stream logs, upload artifacts, and download query extracts.

Aliases:
  angler fs       - Databricks FS standard command suite
  angler storage  - Lake of Tears storage alias

Examples:
  angler fs ls gold/telemetry
  angler fs cat gold/telemetry/sample.json --length 1024
  angler fs cp ./data.csv lot://bronze/data.csv
  angler fs cp lot://gold/report.parquet ./report.parquet
  angler fs stat gold/telemetry/data.parquet
  angler fs upload ./model.onnx models/
  angler fs download gold/summary.csv ./summary.csv
  angler fs rm bronze/old_data/ --recursive --force
  angler fs stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return runFSLs(cmd, args)
		}
		return cmd.Help()
	},
}

var fsLsCmd = &cobra.Command{
	Use:     "ls [path]",
	Aliases: []string{"list"},
	Short:   "List files, directories, and partitions in object storage",
	Long:    `List objects and common prefixes in workspace storage. Supports recursive traversal and JSON output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFSLs(cmd, args)
	},
}

var fsCatCmd = &cobra.Command{
	Use:   "cat <path>",
	Short: "Stream or print the content of a remote file to stdout",
	Long:  `Streams the content of a remote file directly to standard output. Supports byte-range slicing.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := client.CleanStoragePath(args[0])
		if remotePath == "" {
			return ui.NewUsageError(fmt.Errorf("remote path cannot be empty"))
		}

		c := getClient()
		reader, _, err := c.StreamObject(remotePath, fsOffset, fsLength)
		if err != nil {
			return err
		}
		defer reader.Close()

		_, err = io.Copy(ui.OutWriter, reader)
		return err
	},
}

var fsCpCmd = &cobra.Command{
	Use:   "cp <source> <destination>",
	Short: "Copy files between local filesystem and remote Lake of Tears storage",
	Long: `Copy files between local paths and remote storage. Supports schemes like lot://, s3://, and dbfs://.

Examples:
  angler fs cp ./local.parquet lot://bronze/local.parquet
  angler fs cp lot://silver/data.parquet ./downloaded.parquet
  angler fs cp s3://bronze/raw.json lot://silver/raw.json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		srcIsRemote := isRemotePath(src)
		dstIsRemote := isRemotePath(dst)

		c := getClient()

		// Local -> Remote upload
		if !srcIsRemote && dstIsRemote {
			return copyLocalToRemote(c, src, dst)
		}

		// Remote -> Local download
		if srcIsRemote && !dstIsRemote {
			return copyRemoteToLocal(c, src, dst)
		}

		// Remote -> Remote copy
		if srcIsRemote && dstIsRemote {
			return copyRemoteToRemote(c, src, dst)
		}

		// Local -> Local (fallback to local copy)
		return copyLocalToLocal(src, dst)
	},
}

var fsStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Retrieve file metadata and Parquet schema inspection",
	Long:  `Inspect an S3 object key to view file size, headers, ETag, last modified date, and Parquet column metrics.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := client.CleanStoragePath(args[0])
		if remotePath == "" {
			return ui.NewUsageError(fmt.Errorf("remote path cannot be empty"))
		}

		c := getClient()
		meta, err := c.GetStorageMetadata(remotePath)
		if err != nil {
			return err
		}

		return ui.RenderOutput(meta, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, fmt.Sprintf("Object: %s", meta.Key)))
			fmt.Fprintf(ui.OutWriter, "  Size:          %s (%d bytes)\n", meta.SizeFormatted, meta.Size)
			fmt.Fprintf(ui.OutWriter, "  Content-Type:  %s\n", meta.ContentType)
			fmt.Fprintf(ui.OutWriter, "  ETag:          %s\n", meta.ETag)
			if meta.LastModified != nil {
				fmt.Fprintf(ui.OutWriter, "  Last Modified: %s\n", meta.LastModified.Format("2006-01-02 15:04:05 MST"))
			}
			if meta.IsParquet {
				fmt.Fprintln(ui.OutWriter)
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Parquet Metadata:"))
				if meta.RowCount != nil {
					fmt.Fprintf(ui.OutWriter, "  Rows:        %d\n", *meta.RowCount)
				}
				if meta.ColumnCount != nil {
					fmt.Fprintf(ui.OutWriter, "  Columns:     %d\n", *meta.ColumnCount)
				}
				if meta.Compression != "" {
					fmt.Fprintf(ui.OutWriter, "  Compression: %s\n", meta.Compression)
				}
				if len(meta.ParquetColumns) > 0 {
					fmt.Fprintln(ui.OutWriter, "  Schema:")
					for _, col := range meta.ParquetColumns {
						fmt.Fprintf(ui.OutWriter, "    - %s (%s)\n", col.Name, col.Type)
					}
				}
			}
		})
	},
}

var fsUploadCmd = &cobra.Command{
	Use:     "upload <local-path> [remote-key]",
	Aliases: []string{"put"},
	Short:   "Upload local artifacts, datasets, or models to remote storage",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := args[0]
		remoteKey := ""
		if len(args) > 1 {
			remoteKey = args[1]
		}
		if remoteKey == "" && fsUploadPrefix != "" {
			remoteKey = fsUploadPrefix
		}

		c := getClient()
		return copyLocalToRemote(c, localPath, remoteKey)
	},
}

var fsDownloadCmd = &cobra.Command{
	Use:     "download <remote-key> [local-path]",
	Aliases: []string{"get"},
	Short:   "Download remote objects to local disk",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteKey := args[0]
		localPath := ""
		if len(args) > 1 {
			localPath = args[1]
		}
		if localPath == "" {
			localPath = filepath.Base(client.CleanStoragePath(remoteKey))
		}

		c := getClient()
		return copyRemoteToLocal(c, remoteKey, localPath)
	},
}

var fsRmCmd = &cobra.Command{
	Use:     "rm <path>",
	Aliases: []string{"delete"},
	Short:   "Delete objects or prefixes from workspace storage",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := client.CleanStoragePath(args[0])
		if remotePath == "" {
			return ui.NewUsageError(fmt.Errorf("path cannot be empty"))
		}

		if !fsForceDelete && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			confirmMsg := fmt.Sprintf("Are you sure you want to delete '%s'?", remotePath)
			if fsRecursive {
				confirmMsg = fmt.Sprintf("Are you sure you want to RECURSIVELY delete '%s' and all contents?", remotePath)
			}
			if !confirmAction(confirmMsg) {
				ui.PrintWarning("Deletion aborted.")
				return nil
			}
		}

		c := getClient()
		if err := c.DeleteObject(remotePath, fsRecursive); err != nil {
			return err
		}

		ui.PrintSuccess("Deleted %s", remotePath)
		return nil
	},
}

var fsMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a directory prefix marker in workspace storage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := client.CleanStoragePath(args[0])
		if prefix == "" {
			return ui.NewUsageError(fmt.Errorf("prefix cannot be empty"))
		}
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		c := getClient()
		emptyReader := strings.NewReader("")
		resp, err := c.UploadObject(prefix, ".keep", emptyReader, "application/octet-stream")
		if err != nil {
			return err
		}

		ui.PrintSuccess("Created directory prefix %s", resp.Key)
		return nil
	},
}

var fsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get aggregated storage metrics and top-level breakdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		stats, err := c.GetStorageStats()
		if err != nil {
			return err
		}

		return ui.RenderOutput(stats, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Workspace Storage Metrics"))
			fmt.Fprintf(ui.OutWriter, "  Bucket:        %s\n", stats.Bucket)
			if stats.Workspace != "" {
				fmt.Fprintf(ui.OutWriter, "  Workspace:     %s\n", stats.Workspace)
			}
			fmt.Fprintf(ui.OutWriter, "  Total Objects: %d\n", stats.TotalObjects)
			fmt.Fprintf(ui.OutWriter, "  Total Storage: %.2f MB (%d bytes)\n", float64(stats.TotalBytes)/(1024*1024), stats.TotalBytes)

			if len(stats.TopLevelPrefixes) > 0 {
				fmt.Fprintln(ui.OutWriter)
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Prefix Breakdown:"))
				for pfx, count := range stats.TopLevelPrefixes {
					fmt.Fprintf(ui.OutWriter, "  %-20s %d objects\n", pfx, count)
				}
			}
		})
	},
}

func runFSLs(cmd *cobra.Command, args []string) error {
	prefix := ""
	if len(args) > 0 {
		prefix = client.CleanStoragePath(args[0])
	}

	c := getClient()
	browse, err := c.BrowseStorage(prefix, fsLimit, "")
	if err != nil {
		return err
	}

	if fsRecursive {
		// Traverse recursively through subfolders
		return traverseRecursive(c, browse, fsLimit)
	}

	return ui.RenderOutput(browse, func() {
		headers := []string{"TYPE", "NAME", "SIZE", "LAST MODIFIED"}
		var rows [][]string

		for _, folder := range browse.Folders {
			folderName := folder.Name
			if fsAbsolute && folder.Prefix != "" {
				folderName = folder.Prefix
			}
			rows = append(rows, []string{"DIR", folderName + "/", "-", "-"})
		}

		for _, file := range browse.Files {
			fileName := file.Name
			if fsAbsolute && file.Key != "" {
				fileName = file.Key
			}
			lastMod := "-"
			if file.LastModified != nil {
				lastMod = file.LastModified.Format("2006-01-02 15:04:05")
			}
			sizeStr := file.SizeFormatted
			if sizeStr == "" {
				sizeStr = fmt.Sprintf("%d B", file.Size)
			}
			rows = append(rows, []string{"FILE", fileName, sizeStr, lastMod})
		}

		if len(rows) == 0 {
			ui.PrintInfo("No files or directories found in '%s'.", prefix)
			return
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total: %d folder(s), %d file(s).", browse.TotalFolders, browse.TotalFiles)
	})
}

func traverseRecursive(c *client.Client, root *client.StorageBrowseResponse, limit int) error {
	allFiles := append([]client.StorageFile{}, root.Files...)
	allFolders := append([]client.StorageFolder{}, root.Folders...)

	queue := append([]client.StorageFolder{}, root.Folders...)
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		subBrowse, err := c.BrowseStorage(curr.Prefix, limit, "")
		if err != nil {
			continue
		}
		allFiles = append(allFiles, subBrowse.Files...)
		allFolders = append(allFolders, subBrowse.Folders...)
		queue = append(queue, subBrowse.Folders...)
	}

	aggregated := map[string]interface{}{
		"prefix":        root.Prefix,
		"folders":       allFolders,
		"files":         allFiles,
		"total_folders": len(allFolders),
		"total_files":   len(allFiles),
	}

	return ui.RenderOutput(aggregated, func() {
		headers := []string{"TYPE", "PATH", "SIZE", "LAST MODIFIED"}
		var rows [][]string

		for _, folder := range allFolders {
			rows = append(rows, []string{"DIR", folder.Prefix, "-", "-"})
		}
		for _, file := range allFiles {
			path := file.RelKey
			if path == "" {
				path = file.Key
			}
			lastMod := "-"
			if file.LastModified != nil {
				lastMod = file.LastModified.Format("2006-01-02 15:04:05")
			}
			rows = append(rows, []string{"FILE", path, file.SizeFormatted, lastMod})
		}

		if len(rows) > 0 {
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		}
		ui.PrintSuccess("Recursive Total: %d folder(s), %d file(s).", len(allFolders), len(allFiles))
	})
}

func isRemotePath(p string) bool {
	lower := strings.ToLower(strings.TrimSpace(p))
	return strings.HasPrefix(lower, "lot://") ||
		strings.HasPrefix(lower, "s3://") ||
		strings.HasPrefix(lower, "dbfs://")
}

func copyLocalToRemote(c *client.Client, localPath, remoteTarget string) error {
	cleanRemote := client.CleanStoragePath(remoteTarget)

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local file error: %w", err)
	}
	if info.IsDir() {
		return ui.NewUsageError(fmt.Errorf("directory upload not supported directly; please specify a file"))
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer f.Close()

	prefix := ""
	filename := filepath.Base(localPath)

	if cleanRemote != "" {
		if strings.HasSuffix(cleanRemote, "/") {
			prefix = cleanRemote
		} else {
			dir, base := filepath.Split(cleanRemote)
			prefix = dir
			filename = base
		}
	}

	ui.PrintInfo("Uploading %s to storage prefix '%s'...", localPath, prefix)
	resp, err := c.UploadObject(prefix, filename, f, "application/octet-stream")
	if err != nil {
		return err
	}

	ui.PrintSuccess("Uploaded %s -> %s (%d bytes)", localPath, resp.Key, resp.Size)
	return nil
}

func copyRemoteToLocal(c *client.Client, remotePath, localTarget string) error {
	cleanRemote := client.CleanStoragePath(remotePath)
	if cleanRemote == "" {
		return ui.NewUsageError(fmt.Errorf("remote path cannot be empty"))
	}

	target := localTarget
	if target == "" {
		target = filepath.Base(cleanRemote)
	}

	info, err := os.Stat(target)
	if err == nil && info.IsDir() {
		target = filepath.Join(target, filepath.Base(cleanRemote))
	}

	if _, err := os.Stat(target); err == nil && !fsForceDownload && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
		if !confirmAction(fmt.Sprintf("Local file '%s' already exists. Overwrite?", target)) {
			ui.PrintWarning("Download aborted.")
			return nil
		}
	}

	// Ensure parent directory exists
	if dir := filepath.Dir(target); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create local directory: %w", err)
		}
	}

	outF, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outF.Close()

	ui.PrintInfo("Downloading %s to %s...", cleanRemote, target)
	if err := c.DownloadObject(cleanRemote, outF); err != nil {
		return err
	}

	ui.PrintSuccess("Downloaded %s -> %s", cleanRemote, target)
	return nil
}

func copyRemoteToRemote(c *client.Client, srcRemote, dstRemote string) error {
	cleanSrc := client.CleanStoragePath(srcRemote)
	cleanDst := client.CleanStoragePath(dstRemote)

	reader, _, err := c.StreamObject(cleanSrc, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to stream source object: %w", err)
	}
	defer reader.Close()

	prefix, filename := filepath.Split(cleanDst)
	ui.PrintInfo("Copying %s -> %s...", cleanSrc, cleanDst)
	resp, err := c.UploadObject(prefix, filename, reader, "application/octet-stream")
	if err != nil {
		return err
	}

	ui.PrintSuccess("Copied %s -> %s (%d bytes)", cleanSrc, resp.Key, resp.Size)
	return nil
}

func copyLocalToLocal(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func confirmAction(prompt string) bool {
	if assumeYes || nonInteractive || ui.IsNonInteractive() {
		return true
	}
	fmt.Fprintf(ui.OutWriter, "%s (y/N): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func init() {
	// ls flags
	fsLsCmd.Flags().BoolVarP(&fsRecursive, "recursive", "r", false, "Recursively list directories")
	fsLsCmd.Flags().BoolVarP(&fsAbsolute, "absolute", "a", false, "Show absolute keys instead of relative filenames")
	fsLsCmd.Flags().IntVarP(&fsLimit, "limit", "l", 100, "Maximum number of items to list")

	// cat flags
	fsCatCmd.Flags().Int64Var(&fsOffset, "offset", 0, "Byte offset start")
	fsCatCmd.Flags().Int64Var(&fsLength, "length", 0, "Byte length to read (0 reads to EOF)")

	// cp flags
	fsCpCmd.Flags().BoolVarP(&fsForceDownload, "force", "f", false, "Overwrite existing destination files without prompt")

	// upload flags
	fsUploadCmd.Flags().StringVarP(&fsUploadPrefix, "prefix", "p", "", "Destination storage prefix")

	// download flags
	fsDownloadCmd.Flags().BoolVarP(&fsForceDownload, "force", "f", false, "Overwrite existing destination file without prompt")

	// rm flags
	fsRmCmd.Flags().BoolVarP(&fsRecursive, "recursive", "r", false, "Delete all objects under the specified prefix recursively")
	fsRmCmd.Flags().BoolVarP(&fsForceDelete, "force", "f", false, "Delete without interactive confirmation prompt")

	// Register subcommands under fsCmd
	fsCmd.AddCommand(fsLsCmd)
	fsCmd.AddCommand(fsCatCmd)
	fsCmd.AddCommand(fsCpCmd)
	fsCmd.AddCommand(fsStatCmd)
	fsCmd.AddCommand(fsUploadCmd)
	fsCmd.AddCommand(fsDownloadCmd)
	fsCmd.AddCommand(fsRmCmd)
	fsCmd.AddCommand(fsMkdirCmd)
	fsCmd.AddCommand(fsStatsCmd)

	// Add fsCmd to rootCmd
	rootCmd.AddCommand(fsCmd)
}
