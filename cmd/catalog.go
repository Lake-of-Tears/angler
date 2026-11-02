package cmd

import (
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:     "catalog",
	Aliases: []string{"net"},
	Short:   "Browse tables and datasets in the lake (alias: net)",
	Long: `Inspect datasets, Parquet partitions, and storage schemas in Lake of Tears.

Examples:
  angler catalog ls
  angler net`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogList()
	},
}

var catalogListCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all datasets in the lake",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogList()
	},
}

func runCatalogList() error {
	c := getClient()
	items, err := c.ListCatalog()
	if err != nil {
		return err
	}

	return ui.RenderOutput(items, func() {
		if len(items) == 0 {
			ui.PrintInfo("No datasets found in catalog.")
			return
		}

		tableRows := make([][]string, len(items))
		for i, item := range items {
			tableRows[i] = []string{
				item.Name,
				item.Type,
				item.Format,
				fmt.Sprintf("%d", item.Rows),
				item.Size,
				item.LastUpdated,
			}
		}

		headers := []string{"Dataset Name", "Type", "Format", "Rows", "Size", "Updated"}
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
		ui.PrintSuccess("Total datasets netted: %d", len(items))
	})
}

func init() {
	catalogCmd.AddCommand(catalogListCmd)
	rootCmd.AddCommand(catalogCmd)
}
