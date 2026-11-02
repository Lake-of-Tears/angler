package cmd

import (
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	sqlStatement string
	outputFormat string
)

var queryCmd = &cobra.Command{
	Use:     "query [SQL]",
	Aliases: []string{"cast"},
	Short:   "Cast a query into the data lake (alias: cast)",
	Long: `Execute SQL queries against DuckDB and S3 Parquet tables in Lake of Tears.
You can specify the query as an argument or using the -e / --execute flag.

Examples:
  angler cast "SELECT * FROM lake.telemetry LIMIT 10"
  angler query -e "SELECT department, count(*) FROM employees GROUP BY 1" --format csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statement := sqlStatement
		if statement == "" && len(args) > 0 {
			statement = strings.Join(args, " ")
		}

		if strings.TrimSpace(statement) == "" {
			return ui.NewUsageError(fmt.Errorf("no SQL query provided. Specify query as argument or with -e \"SELECT ...\""))
		}

		c := getClient()
		ui.PrintInfo("Casting query: %s", ui.StyleText(ui.TitleStyle, statement))

		res, err := c.Query(statement)
		if err != nil {
			return err
		}

		tableRows := make([][]string, len(res.Rows))
		for i, row := range res.Rows {
			tableRows[i] = make([]string, len(row))
			for j, val := range row {
				tableRows[i][j] = fmt.Sprintf("%v", val)
			}
		}

		activeFormat := ui.GetOutputFormat()
		if activeFormat == ui.FormatCSV || strings.EqualFold(outputFormat, ui.FormatCSV) && !cmd.Flags().Changed("output") && !jsonOutput {
			return ui.RenderCSV(res.Columns, tableRows)
		}

		return ui.RenderOutput(res, func() {
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(res.Columns, tableRows))
			ui.PrintSuccess("Reeled in %d rows in %s.", res.RowCount, res.Duration)
		})
	},
}

func init() {
	queryCmd.Flags().StringVarP(&sqlStatement, "execute", "e", "", "SQL query statement to execute")
	queryCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table, json, yaml, toon, csv")
	rootCmd.AddCommand(queryCmd)
}
