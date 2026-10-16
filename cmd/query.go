package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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
			return fmt.Errorf("no SQL query provided. Specify query as argument or with -e \"SELECT ...\"")
		}

		c := getClient()
		ui.PrintInfo("Casting query: %s", ui.TitleStyle.Render(statement))

		res, err := c.Query(statement)
		if err != nil {
			ui.PrintError("Query failed: %v", err)
			return err
		}

		if jsonOutput || outputFormat == "json" {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if outputFormat == "csv" {
			writer := csv.NewWriter(os.Stdout)
			_ = writer.Write(res.Columns)
			for _, row := range res.Rows {
				rowStr := make([]string, len(row))
				for i, val := range row {
					rowStr[i] = fmt.Sprintf("%v", val)
				}
				_ = writer.Write(rowStr)
			}
			writer.Flush()
			return nil
		}

		// Render table using lipgloss/table
		tableRows := make([][]string, len(res.Rows))
		for i, row := range res.Rows {
			tableRows[i] = make([]string, len(row))
			for j, val := range row {
				tableRows[i][j] = fmt.Sprintf("%v", val)
			}
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(ui.BorderColor)).
			Headers(res.Columns...).
			Rows(tableRows...)

		fmt.Println(t)
		ui.PrintSuccess("Reeled in %d rows in %s.", res.RowCount, res.Duration)
		return nil
	},
}

func init() {
	queryCmd.Flags().StringVarP(&sqlStatement, "execute", "e", "", "SQL query statement to execute")
	queryCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table, json, csv")
	rootCmd.AddCommand(queryCmd)
}
