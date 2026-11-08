package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	outputFormat     = "table"
	sqlStatement     string
	sqlFile          string
	sqlOutputFormat  = "table"
	sqlWarehouse     string
	sqlCatalog       string
	sqlSchema        string
	sqlRowLimit      int
	sqlParams        string
	sqlExplainFlag   bool
	sqlModel         string
	sqlCurrentSQL    string
	sqlAutoExecute   bool
	sqlErrorMsg      string
	sqlHistoryLimit  int
	sqlHistoryStatus string
)

// sqlCmd represents the root of the SQL command suite
var sqlCmd = &cobra.Command{
	Use:     "sql [SQL]",
	Aliases: []string{"query", "cast"},
	Short:   "Unified SQL statement execution, history, and AI assistance (aliases: query, cast)",
	Long: `Execute SQL statements against DuckDB and Lake of Tears lakehouse storage,
explain query execution plans, generate SQL from natural language prompts,
and inspect or repair failing queries.

Aliases:
  angler sql      - Databricks / ANSI SQL command suite
  angler query    - Standard analytical query command
  angler cast     - Nautical Lake of Tears query command

Examples:
  angler sql "SELECT * FROM lake.telemetry LIMIT 10"
  angler sql exec -f queries/report.sql --format csv
  angler sql explain "SELECT count(*) FROM users"
  angler sql generate "Find top 10 users by spend last month" --execute
  angler sql fix "SELECT empid FROM emps" --error "column empid not found"
  angler sql history --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statement := resolveStatement(args)
		if strings.TrimSpace(statement) == "" {
			return cmd.Help()
		}
		return runSQLExec(cmd, args)
	},
}

var sqlExecCmd = &cobra.Command{
	Use:     "exec [SQL]",
	Aliases: []string{"run", "execute"},
	Short:   "Execute a SQL query or DDL/DML statement",
	Long: `Execute analytical queries (SELECT) or DDL/DML statements (CREATE TABLE, INSERT, ALTER)
against the Lake of Tears execution engine.

Supports reading from command arguments, flags (-e), or files (-f).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSQLExec(cmd, args)
	},
}

var sqlExplainCmd = &cobra.Command{
	Use:   "explain [SQL]",
	Short: "Explain query execution plan and optimization steps",
	Long:  `Analyze and explain a DuckDB SQL statement step-by-step using Lake of Tears AI engine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statement := resolveStatement(args)
		if strings.TrimSpace(statement) == "" {
			return ui.NewUsageError(fmt.Errorf("no SQL statement provided. Specify statement as argument, with -e, or via -f/--file"))
		}

		c := getClient()
		ui.PrintInfo("Explaining query: %s", ui.StyleText(ui.TitleStyle, statement))

		resp, err := c.ExplainSQL(client.SqlExplainRequest{
			SQL:   statement,
			Model: sqlModel,
		})
		if err != nil {
			return err
		}

		if !resp.OK && resp.Error != "" {
			return ui.NewCLIError(ui.ExitGeneralError, fmt.Errorf("explain failed: %s", resp.Error))
		}

		return ui.RenderOutput(resp, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Query Plan & Optimization Analysis"))
			if resp.Summary != "" {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Summary: ")+resp.Summary)
				fmt.Fprintln(ui.OutWriter)
			}

			if len(resp.Steps) > 0 {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Execution Steps:"))
				for i, step := range resp.Steps {
					fmt.Fprintf(ui.OutWriter, "  %d. %s\n", i+1, step)
				}
				fmt.Fprintln(ui.OutWriter)
			}

			if len(resp.TablesReferenced) > 0 {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.MutedStyle, "Tables Referenced: ")+strings.Join(resp.TablesReferenced, ", "))
			}
			if len(resp.AIFunctionsUsed) > 0 {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.WarningStyle, "AI Functions: ")+strings.Join(resp.AIFunctionsUsed, ", "))
			}
		})
	},
}

var sqlGenerateCmd = &cobra.Command{
	Use:     "generate <prompt>",
	Aliases: []string{"gen", "create"},
	Short:   "Generate executable SQL from a natural language prompt",
	Long: `Translate a natural language question into executable DuckDB SQL grounded in your catalog schemas.
Optionally execute the generated query immediately with -x/--execute.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.TrimSpace(strings.Join(args, " "))
		if prompt == "" {
			return ui.NewUsageError(fmt.Errorf("please provide a natural language prompt to generate SQL"))
		}

		c := getClient()
		ui.PrintInfo("Generating SQL for prompt: %s", ui.StyleText(ui.TitleStyle, prompt))

		resp, err := c.GenerateSQL(client.SqlGenerateRequest{
			Prompt:     prompt,
			CurrentSQL: sqlCurrentSQL,
			Model:      sqlModel,
		})
		if err != nil {
			return err
		}

		if !resp.OK && resp.Error != "" {
			return ui.NewCLIError(ui.ExitGeneralError, fmt.Errorf("SQL generation failed: %s", resp.Error))
		}

		err = ui.RenderOutput(resp, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Generated SQL:"))
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, resp.SQL))
			fmt.Fprintln(ui.OutWriter)
			if resp.Explanation != "" {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.MutedStyle, "Explanation: ")+resp.Explanation)
			}
			if len(resp.TablesUsed) > 0 {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Tables: ")+strings.Join(resp.TablesUsed, ", "))
			}
		})
		if err != nil {
			return err
		}

		if sqlAutoExecute && strings.TrimSpace(resp.SQL) != "" {
			ui.PrintInfo("Executing generated SQL statement...")
			sqlStatement = resp.SQL
			return runSQLExec(cmd, []string{resp.SQL})
		}

		return nil
	},
}

var sqlFixCmd = &cobra.Command{
	Use:     "fix [SQL]",
	Aliases: []string{"repair"},
	Short:   "Diagnose and repair a failing SQL query",
	Long: `Analyze a broken SQL statement alongside its error message and propose an AI-repaired query grounded in catalog schemas.
Optionally execute the repaired query immediately with -x/--execute.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statement := resolveStatement(args)
		if strings.TrimSpace(statement) == "" {
			return ui.NewUsageError(fmt.Errorf("no SQL statement provided. Specify statement as argument, with -e, or via -f/--file"))
		}
		if strings.TrimSpace(sqlErrorMsg) == "" {
			return ui.NewUsageError(fmt.Errorf("missing required error message. Provide the database error with -e/--error <msg>"))
		}

		c := getClient()
		ui.PrintInfo("Diagnosing query error...")

		resp, err := c.FixSQL(client.SqlFixRequest{
			SQL:   statement,
			Error: sqlErrorMsg,
			Model: sqlModel,
		})
		if err != nil {
			return err
		}

		if !resp.OK && resp.Error != "" {
			return ui.NewCLIError(ui.ExitGeneralError, fmt.Errorf("SQL diagnosis failed: %s", resp.Error))
		}

		err = ui.RenderOutput(resp, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Repaired SQL:"))
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, resp.FixedSQL))
			fmt.Fprintln(ui.OutWriter)
			if resp.Explanation != "" {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.MutedStyle, "Diagnosis & Fix: ")+resp.Explanation)
			}
		})
		if err != nil {
			return err
		}

		if sqlAutoExecute && strings.TrimSpace(resp.FixedSQL) != "" {
			ui.PrintInfo("Executing repaired SQL statement...")
			sqlStatement = resp.FixedSQL
			return runSQLExec(cmd, []string{resp.FixedSQL})
		}

		return nil
	},
}

var sqlHistoryCmd = &cobra.Command{
	Use:     "history",
	Aliases: []string{"list", "ls"},
	Short:   "List recent SQL statement executions",
	Long:    `Display recent SQL statement executions in the active workspace with status, row count, and duration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		items, err := c.ListStatements(sqlHistoryLimit, sqlHistoryStatus)
		if err != nil {
			return err
		}

		return ui.RenderOutput(items, func() {
			if len(items) == 0 {
				ui.PrintInfo("No statement executions found.")
				return
			}

			headers := []string{"STATEMENT ID", "STATUS", "ROWS", "DURATION", "CATALOG.SCHEMA", "STATEMENT PREVIEW"}
			rows := make([][]string, len(items))

			for i, item := range items {
				statusStr := item.Status.State
				rowCountStr := "-"
				durationStr := "-"
				catSchema := fmt.Sprintf("%s.%s", item.Catalog, item.Schema)

				if item.Manifest != nil {
					rowCountStr = fmt.Sprintf("%d", item.Manifest.TotalRowCount)
				}
				if item.ExecutionMs != nil {
					durationStr = fmt.Sprintf("%.2fms", *item.ExecutionMs)
				}

				stmtPreview := strings.ReplaceAll(item.StatementID, "\n", " ")
				// If StatementID is uuid, keep preview concise
				rows[i] = []string{
					item.StatementID,
					statusStr,
					rowCountStr,
					durationStr,
					catSchema,
					truncateString(stmtPreview, 40),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Showing %d recent statements.", len(items))
		})
	},
}

var sqlGetCmd = &cobra.Command{
	Use:   "get <statement_id>",
	Short: "Get status and details of a statement execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		statementID := args[0]
		c := getClient()

		stmt, err := c.GetStatement(statementID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(stmt, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, fmt.Sprintf("Statement %s", stmt.StatementID)))
			fmt.Fprintf(ui.OutWriter, "  Status:    %s\n", stmt.Status.State)
			if stmt.Catalog != "" {
				fmt.Fprintf(ui.OutWriter, "  Catalog:   %s.%s\n", stmt.Catalog, stmt.Schema)
			}
			if stmt.ExecutionMs != nil {
				fmt.Fprintf(ui.OutWriter, "  Execution: %.2fms\n", *stmt.ExecutionMs)
			}
			if stmt.Manifest != nil {
				fmt.Fprintf(ui.OutWriter, "  Rows:      %d (Total Chunks: %d)\n", stmt.Manifest.TotalRowCount, stmt.Manifest.TotalChunkCount)
			}
			if stmt.Status.Error != nil {
				fmt.Fprintf(ui.OutWriter, "  Error:     %s (%s)\n", stmt.Status.Error.Message, stmt.Status.Error.ErrorCode)
			}
		})
	},
}

var sqlCancelCmd = &cobra.Command{
	Use:   "cancel <statement_id>",
	Short: "Cancel an in-flight statement execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		statementID := args[0]
		c := getClient()

		if err := c.CancelStatement(statementID); err != nil {
			return err
		}

		ui.PrintSuccess("Statement %s canceled.", statementID)
		return nil
	},
}

func resolveStatement(args []string) string {
	if sqlFile != "" {
		data, err := os.ReadFile(sqlFile)
		if err == nil {
			return string(data)
		}
	}
	if sqlStatement != "" {
		return sqlStatement
	}
	if len(args) > 0 {
		return strings.Join(args, " ")
	}
	return ""
}

func runSQLExec(cmd *cobra.Command, args []string) error {
	statement := resolveStatement(args)
	if strings.TrimSpace(statement) == "" {
		return ui.NewUsageError(fmt.Errorf("no SQL query provided. Specify query as argument, with -e \"SELECT ...\", or via -f/--file <path>"))
	}

	c := getClient()
	ui.PrintInfo("Casting query: %s", ui.StyleText(ui.TitleStyle, statement))

	if sqlExplainFlag {
		exp, err := c.ExplainSQL(client.SqlExplainRequest{SQL: statement})
		if err == nil && exp.OK && exp.Summary != "" {
			ui.PrintInfo("Plan Summary: %s", exp.Summary)
		}
	}

	var parsedParams []interface{}
	if sqlParams != "" {
		if err := json.Unmarshal([]byte(sqlParams), &parsedParams); err != nil {
			// If not array, try single object or scalar
			var singleParam interface{}
			if err2 := json.Unmarshal([]byte(sqlParams), &singleParam); err2 == nil {
				parsedParams = []interface{}{singleParam}
			} else {
				return ui.NewUsageError(fmt.Errorf("invalid --params JSON: %w", err))
			}
		}
	}

	req := client.StatementExecutionRequest{
		Statement:   statement,
		WarehouseID: sqlWarehouse,
		Catalog:     sqlCatalog,
		Schema:      sqlSchema,
		Parameters:  parsedParams,
		RowLimit:    sqlRowLimit,
		Format:      "JSON_ARRAY",
	}

	res, err := c.ExecuteStatement(req)
	if err != nil {
		// Fallback to legacy /api/query if statements API endpoint is unavailable (e.g. older server version)
		if strings.Contains(err.Error(), "HTTP 404") {
			return runLegacyQuery(statement, cmd)
		}
		return err
	}

	if res.Status.State == "FAILED" {
		errMsg := "query execution failed"
		if res.Status.Error != nil && res.Status.Error.Message != "" {
			errMsg = res.Status.Error.Message
		}
		return ui.NewCLIError(ui.ExitGeneralError, fmt.Errorf("%s", errMsg))
	}

	// Extract column headers and data rows
	var columns []string
	if res.Manifest != nil && len(res.Manifest.Schema.Columns) > 0 {
		for _, col := range res.Manifest.Schema.Columns {
			columns = append(columns, col.Name)
		}
	}

	var dataArray [][]interface{}
	if res.Result != nil && res.Result.DataArray != nil {
		dataArray = res.Result.DataArray
	}

	// If columns empty but rows exist, synthesize col headers
	if len(columns) == 0 && len(dataArray) > 0 {
		for j := range dataArray[0] {
			columns = append(columns, fmt.Sprintf("col_%d", j+1))
		}
	}

	activeFormat := ui.GetOutputFormat()
	formatFlag := sqlOutputFormat
	if cmd.Flags().Changed("format") {
		formatFlag = strings.ToLower(strings.TrimSpace(formatFlag))
	}

	// Handle CSV format
	if activeFormat == ui.FormatCSV || strings.EqualFold(formatFlag, ui.FormatCSV) {
		tableRows := make([][]string, len(dataArray))
		for i, row := range dataArray {
			tableRows[i] = make([]string, len(row))
			for j, val := range row {
				tableRows[i][j] = fmt.Sprintf("%v", val)
			}
		}
		return ui.RenderCSV(columns, tableRows)
	}

	// Handle NDJSON format
	if activeFormat == ui.FormatNDJSON || strings.EqualFold(formatFlag, ui.FormatNDJSON) {
		return ui.RenderNDJSON(columns, dataArray)
	}

	// Structured output: json, yaml, toon
	return ui.RenderOutput(res, func() {
		tableRows := make([][]string, len(dataArray))
		for i, row := range dataArray {
			tableRows[i] = make([]string, len(row))
			for j, val := range row {
				tableRows[i][j] = fmt.Sprintf("%v", val)
			}
		}

		if len(columns) > 0 || len(tableRows) > 0 {
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(columns, tableRows))
		}

		rowCount := 0
		if res.Result != nil {
			rowCount = res.Result.RowCount
		} else if res.Manifest != nil {
			rowCount = res.Manifest.TotalRowCount
		}

		execDuration := ""
		if res.ExecutionMs != nil {
			execDuration = fmt.Sprintf("%.2fms", *res.ExecutionMs)
		}

		if execDuration != "" {
			ui.PrintSuccess("Reeled in %d rows in %s (statement: %s).", rowCount, execDuration, res.StatementID)
		} else {
			ui.PrintSuccess("Reeled in %d rows (statement: %s).", rowCount, res.StatementID)
		}
	})
}

func runLegacyQuery(statement string, cmd *cobra.Command) error {
	c := getClient()
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
	if activeFormat == ui.FormatCSV || strings.EqualFold(sqlOutputFormat, ui.FormatCSV) {
		return ui.RenderCSV(res.Columns, tableRows)
	}
	if activeFormat == ui.FormatNDJSON || strings.EqualFold(sqlOutputFormat, ui.FormatNDJSON) {
		return ui.RenderNDJSON(res.Columns, res.Rows)
	}

	return ui.RenderOutput(res, func() {
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(res.Columns, tableRows))
		ui.PrintSuccess("Reeled in %d rows in %s.", res.RowCount, res.Duration)
	})
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var queryCmd = &cobra.Command{
	Use:     "query [SQL]",
	Aliases: []string{"cast"},
	Short:   "Execute SQL query against the data lake (alias: cast)",
	Long: `Execute SQL queries against DuckDB and S3 Parquet tables in Lake of Tears.
You can specify the query as an argument or using the -e / --execute flag.

Aliases:
  angler query    - Standard analytical query command
  angler cast     - Nautical Lake of Tears query command`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statement := resolveStatement(args)
		if strings.TrimSpace(statement) == "" {
			return ui.NewUsageError(fmt.Errorf("no SQL query provided. Specify query as argument or with -e \"SELECT ...\""))
		}
		return runSQLExec(cmd, args)
	},
}

func init() {
	// Root, exec, and query flags
	for _, c := range []*cobra.Command{sqlCmd, sqlExecCmd, queryCmd} {
		c.Flags().StringVarP(&sqlStatement, "execute", "e", "", "SQL query statement to execute")
		c.Flags().StringVarP(&sqlFile, "file", "f", "", "Path to file containing SQL query")
		c.Flags().StringVar(&sqlOutputFormat, "format", "table", "Output format: table, json, yaml, toon, csv, ndjson")
		c.Flags().IntVar(&sqlRowLimit, "limit", 1000, "Maximum number of rows to return")
		c.Flags().StringVar(&sqlParams, "params", "", "JSON parameters array or object")
		c.Flags().StringVar(&sqlWarehouse, "warehouse", "default", "Execution warehouse ID")
		c.Flags().StringVar(&sqlCatalog, "catalog", "main", "Catalog scope")
		c.Flags().StringVar(&sqlSchema, "schema", "default", "Schema scope")
		c.Flags().BoolVar(&sqlExplainFlag, "explain", false, "Explain execution plan prior to running")
	}

	// explain command flags
	sqlExplainCmd.Flags().StringVarP(&sqlStatement, "execute", "e", "", "SQL query statement to explain")
	sqlExplainCmd.Flags().StringVarP(&sqlFile, "file", "f", "", "Path to file containing SQL query")
	sqlExplainCmd.Flags().StringVar(&sqlModel, "model", "", "AI model override for plan explanation")

	// generate command flags
	sqlGenerateCmd.Flags().StringVar(&sqlCurrentSQL, "current-sql", "", "Current SQL context to refine")
	sqlGenerateCmd.Flags().StringVar(&sqlModel, "model", "", "AI model override")
	sqlGenerateCmd.Flags().BoolVarP(&sqlAutoExecute, "run", "x", false, "Execute the generated SQL statement automatically")

	// fix command flags
	sqlFixCmd.Flags().StringVarP(&sqlStatement, "statement", "s", "", "Failing SQL statement")
	sqlFixCmd.Flags().StringVarP(&sqlFile, "file", "f", "", "Path to file containing SQL statement")
	sqlFixCmd.Flags().StringVarP(&sqlErrorMsg, "error", "e", "", "Database error message from failure")
	sqlFixCmd.Flags().StringVar(&sqlModel, "model", "", "AI model override")
	sqlFixCmd.Flags().BoolVarP(&sqlAutoExecute, "run", "x", false, "Execute the repaired SQL statement automatically")

	// history command flags
	sqlHistoryCmd.Flags().IntVar(&sqlHistoryLimit, "limit", 20, "Maximum number of statements to list")
	sqlHistoryCmd.Flags().StringVar(&sqlHistoryStatus, "status", "", "Filter statements by execution state")

	// Register subcommands under sqlCmd
	sqlCmd.AddCommand(sqlExecCmd)
	sqlCmd.AddCommand(sqlExplainCmd)
	sqlCmd.AddCommand(sqlGenerateCmd)
	sqlCmd.AddCommand(sqlFixCmd)
	sqlCmd.AddCommand(sqlHistoryCmd)
	sqlCmd.AddCommand(sqlGetCmd)
	sqlCmd.AddCommand(sqlCancelCmd)

	// Add sqlCmd and queryCmd to root
	rootCmd.AddCommand(sqlCmd)
	rootCmd.AddCommand(queryCmd)
}
