package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	lakesetCatalog     string
	lakesetSchema      string
	lakesetTier        string
	lakesetSearch      string
	lakesetForce       bool
	lakesetDataset     string
	lakesetDashboardID string
	lakesetTTL         int
	lakesetRawToken    bool
	lakesetSQL         string
	lakesetChartType   string
	lakesetDimension   string
	lakesetMetric      string
	lakesetAgg         string
	lakesetLimit       int
	lakesetTab         string
	lakesetNoBrowser   bool
	lakesetPrintOnly   bool

	// browserLauncher is injectable for unit tests.
	browserLauncher = openBrowserURL
)

var lakesetCmd = &cobra.Command{
	Use:     "lakeset",
	Aliases: []string{"bi"},
	Short:   "Native Zero-Login Lakeset BI Studio, DuckDB Semantic Sync & Embedded Analytics (alias: bi)",
	Long: `Inspect Lakeset BI Studio health, browse governed workspace and shared catalog datasets
with semantic column descriptions, trigger automated DuckDB + MinIO S3 dataset synchronization
in Superset, mint RLS-scoped Guest Tokens, execute live DuckDB S3 Chart Builder aggregations,
and open zero-login deep-links into Lakeset Studio.

Aliases:
  angler lakeset
  angler bi

Examples:
  angler lakeset status
  angler lakeset datasets --tier gold --search revenue
  angler lakeset sync --catalog main --force
  angler lakeset guest-token --dataset main.gold.revenue_summary --ttl 3600
  angler lakeset explore --dataset main.gold.revenue_summary --dimension region --metric amount --agg SUM
  angler lakeset open --dataset main.gold.revenue_summary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLakesetStatus(cmd, args)
	},
}

var lakesetStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display Lakeset BI Studio health, DuckDB Lakehouse status, and dataset counts",
	RunE:  runLakesetStatus,
}

func runLakesetStatus(cmd *cobra.Command, args []string) error {
	c := getClient()
	st, err := c.GetLakesetStatus()
	if err != nil {
		return err
	}

	return ui.RenderOutput(st, func() {
		supersetStr := "Offline"
		if st.SupersetOnline {
			supersetStr = "Online"
		}
		lastSync := "Never"
		if st.LastSyncedAt != nil && *st.LastSyncedAt != "" {
			lastSync = *st.LastSyncedAt
		}

		ui.PrintSuccess("Lakeset BI Studio Status (%s)", st.WorkspaceSlug)
		fmt.Fprintf(ui.OutWriter, "  Superset Engine:      %s\n", supersetStr)
		fmt.Fprintf(ui.OutWriter, "  Lakehouse Database:   %s (%s)\n", st.DatabaseName, st.Engine)
		fmt.Fprintf(ui.OutWriter, "  Workspace:            %s (%s)\n", st.WorkspaceName, st.WorkspaceSlug)
		fmt.Fprintf(ui.OutWriter, "  Caller Role:          %s (%s)\n", st.Role, st.UserEmail)
		fmt.Fprintf(ui.OutWriter, "  Authorized Datasets:  %d (Synced: %d)\n", st.DatasetsCount, st.SyncedDatasetsCount)
		fmt.Fprintf(ui.OutWriter, "  Workspace Dashboards: %d\n", st.DashboardsCount)
		fmt.Fprintf(ui.OutWriter, "  Last Synced At:       %s\n", lastSync)
	})
}

var lakesetDatasetsCmd = &cobra.Command{
	Use:     "datasets",
	Aliases: []string{"ls", "list"},
	Short:   "List RBAC-authorized workspace and shared catalog datasets in Lakeset BI Studio",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resp, err := c.ListLakesetDatasets(client.LakesetDatasetsFilter{
			Catalog: lakesetCatalog,
			Schema:  lakesetSchema,
			Tier:    lakesetTier,
			Search:  lakesetSearch,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			if len(resp.Datasets) == 0 {
				ui.PrintInfo("No Lakeset datasets found matching filter.")
				return
			}

			headers := []string{"DATASET", "SQL VIEW", "TIER", "COLUMNS", "SYNCED", "DESCRIPTION"}
			rows := make([][]string, len(resp.Datasets))
			for i, d := range resp.Datasets {
				tier := strings.ToUpper(d.Tier)
				if tier == "" {
					tier = "-"
				}
				synced := "No"
				if d.Synced {
					synced = "Yes"
				}
				desc := "-"
				if d.Description != nil && *d.Description != "" {
					desc = *d.Description
					if len(desc) > 48 {
						desc = desc[:45] + "..."
					}
				}
				rows[i] = []string{
					d.FullName,
					d.SQLView,
					tier,
					fmt.Sprintf("%d", d.ColumnCount),
					synced,
					desc,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total Lakeset datasets: %d", resp.Total)
		})
	},
}

var lakesetSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Register DuckDB Lakehouse database in Superset and sync catalog datasets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resp, err := c.SyncLakeset(client.LakesetSyncRequest{
			CatalogSlug: lakesetCatalog,
			SchemaSlug:  lakesetSchema,
			Force:       lakesetForce,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Synced %d dataset(s) to Lakeset BI Studio (%s)", resp.SyncedDatasetsCount, resp.DatabaseName)
			fmt.Fprintf(ui.OutWriter, "  Workspace:   %s\n", resp.WorkspaceSlug)
			fmt.Fprintf(ui.OutWriter, "  Database ID: %d (%s)\n", resp.DatabaseID, resp.Engine)
			fmt.Fprintf(ui.OutWriter, "  Synced At:   %s\n", resp.SyncedAt)
		})
	},
}

var lakesetGuestTokenCmd = &cobra.Command{
	Use:   "guest-token",
	Short: "Mint a short-lived Superset Guest Token (JWT) with workspace RLS rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resp, err := c.CreateLakesetGuestToken(client.LakesetGuestTokenRequest{
			Dataset:     lakesetDataset,
			DashboardID: lakesetDashboardID,
			TTLSeconds:  lakesetTTL,
		})
		if err != nil {
			return err
		}

		if lakesetRawToken {
			fmt.Fprintln(ui.OutWriter, resp.Token)
			return nil
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Minted Lakeset Guest Token for %s (Workspace: %s)", resp.UserEmail, resp.WorkspaceSlug)
			fmt.Fprintf(ui.OutWriter, "  Token:      %s\n", resp.Token)
			fmt.Fprintf(ui.OutWriter, "  Expires In: %ds\n", resp.ExpiresIn)
			if resp.ExpiresAt != "" {
				fmt.Fprintf(ui.OutWriter, "  Expires At: %s\n", resp.ExpiresAt)
			}
			fmt.Fprintf(ui.OutWriter, "  RLS Rules:  %d\n", len(resp.RLS))
		})
	},
}

var lakesetExploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Execute live DuckDB S3 Chart Builder aggregations and SQL preview",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resp, err := c.ExploreLakesetData(client.LakesetExploreRequest{
			Dataset:   lakesetDataset,
			SQL:       lakesetSQL,
			ChartType: lakesetChartType,
			Dimension: lakesetDimension,
			Metric:    lakesetMetric,
			MetricAgg: lakesetAgg,
			Limit:     lakesetLimit,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Lakeset Chart Builder Query (%s — %s)", resp.Dataset, strings.ToUpper(resp.ChartType))
			fmt.Fprintf(ui.OutWriter, "  Dimension: %s | Metric: %s\n", resp.Dimension, resp.Metric)
			fmt.Fprintf(ui.OutWriter, "  SQL:       %s\n", resp.SQL)

			if resp.Data != nil {
				if val, ok := resp.Data["value"]; ok {
					fmt.Fprintf(ui.OutWriter, "  KPI Value: %v\n", val)
					if trend, ok := resp.Data["trend"]; ok {
						fmt.Fprintf(ui.OutWriter, "  Trend:     %v\n", trend)
					}
					return
				}

				catsRaw, hasCats := resp.Data["categories"].([]any)
				seriesRaw, hasSeries := resp.Data["series"].([]any)
				if hasCats && hasSeries && len(seriesRaw) > 0 {
					firstSeries, _ := seriesRaw[0].(map[string]any)
					metricName := resp.Metric
					if n, ok := firstSeries["name"].(string); ok && n != "" {
						metricName = n
					}
					valsRaw, _ := firstSeries["data"].([]any)

					headers := []string{strings.ToUpper(resp.Dimension), strings.ToUpper(metricName)}
					rows := make([][]string, len(catsRaw))
					for i, cat := range catsRaw {
						valStr := "-"
						if i < len(valsRaw) {
							valStr = fmt.Sprintf("%v", valsRaw[i])
						}
						rows[i] = []string{fmt.Sprintf("%v", cat), valStr}
					}
					fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
				} else if hasSeries && len(seriesRaw) > 0 {
					// Pie chart format: [{"name": "...", "value": ...}]
					headers := []string{strings.ToUpper(resp.Dimension), strings.ToUpper(resp.Metric)}
					var rows [][]string
					for _, item := range seriesRaw {
						if m, ok := item.(map[string]any); ok {
							rows = append(rows, []string{
								fmt.Sprintf("%v", m["name"]),
								fmt.Sprintf("%v", m["value"]),
							})
						}
					}
					if len(rows) > 0 {
						fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
					}
				}
			}
		})
	},
}

var lakesetOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open or print zero-login deep-links into the Lakeset BI Studio web UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getActiveContext()
		baseURL := strings.TrimRight(ctx.Endpoint, "/")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}

		params := url.Values{}
		tab := strings.ToLower(strings.TrimSpace(lakesetTab))
		if lakesetSQL != "" {
			if tab == "" {
				tab = "sqllab"
			}
			params.Set("tab", tab)
			params.Set("sql", lakesetSQL)
		} else if lakesetDataset != "" {
			if tab == "" {
				tab = "explore"
			}
			params.Set("tab", tab)
			params.Set("dataset", lakesetDataset)
		} else if tab != "" {
			params.Set("tab", tab)
		}

		targetURL := baseURL + "/lakeset"
		if len(params) > 0 {
			targetURL += "?" + params.Encode()
		}

		launched := false
		shouldLaunch := !lakesetNoBrowser && !lakesetPrintOnly && !ui.IsNonInteractive() && ui.IsStdoutTTY()
		if shouldLaunch {
			if err := browserLauncher(targetURL); err == nil {
				launched = true
			}
		}

		payload := map[string]any{
			"url":      targetURL,
			"tab":      tab,
			"dataset":  lakesetDataset,
			"sql":      lakesetSQL,
			"launched": launched,
		}

		return ui.RenderOutput(payload, func() {
			ui.PrintSuccess("Lakeset BI Studio URL: %s", targetURL)
			fmt.Fprintln(ui.OutWriter, targetURL)
		})
	},
}

func openBrowserURL(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	return cmd.Start()
}

func init() {
	// datasets flags
	lakesetDatasetsCmd.Flags().StringVar(&lakesetCatalog, "catalog", "", "Filter datasets by catalog slug or name")
	lakesetDatasetsCmd.Flags().StringVar(&lakesetSchema, "schema", "", "Filter datasets by schema slug or name")
	lakesetDatasetsCmd.Flags().StringVar(&lakesetTier, "tier", "", "Filter datasets by medallion tier (bronze, silver, gold)")
	lakesetDatasetsCmd.Flags().StringVarP(&lakesetSearch, "search", "q", "", "Search dataset names, table descriptions, and columns")

	// sync flags
	lakesetSyncCmd.Flags().StringVar(&lakesetCatalog, "catalog", "", "Scope sync output to a specific catalog")
	lakesetSyncCmd.Flags().StringVar(&lakesetSchema, "schema", "", "Scope sync output to a specific schema")
	lakesetSyncCmd.Flags().BoolVar(&lakesetForce, "force", true, "Force refresh of existing dataset metadata and column definitions")

	// guest-token flags
	lakesetGuestTokenCmd.Flags().StringVar(&lakesetDataset, "dataset", "", "Optional target dataset full name (catalog.schema.table)")
	lakesetGuestTokenCmd.Flags().StringVar(&lakesetDashboardID, "dashboard-id", "", "Optional target dashboard UUID/ID")
	lakesetGuestTokenCmd.Flags().IntVar(&lakesetTTL, "ttl", 3600, "Token lifetime in seconds (60..86400)")
	lakesetGuestTokenCmd.Flags().BoolVar(&lakesetRawToken, "raw", false, "Print raw JWT guest token string to stdout")

	// explore flags
	lakesetExploreCmd.Flags().StringVar(&lakesetDataset, "dataset", "", "Target governed dataset (catalog.schema.table or schema.table)")
	lakesetExploreCmd.Flags().StringVar(&lakesetSQL, "sql", "", "Custom analytical DuckDB SQL query override")
	lakesetExploreCmd.Flags().StringVar(&lakesetChartType, "chart-type", "bar", "Chart aggregation format (bar, line, area, pie, kpi, table)")
	lakesetExploreCmd.Flags().StringVar(&lakesetDimension, "dimension", "", "Grouping dimension column")
	lakesetExploreCmd.Flags().StringVar(&lakesetMetric, "metric", "count", "Metric column")
	lakesetExploreCmd.Flags().StringVar(&lakesetAgg, "agg", "count", "Aggregation function (COUNT, SUM, AVG, MIN, MAX)")
	lakesetExploreCmd.Flags().IntVar(&lakesetLimit, "limit", 50, "Maximum series categories/rows (1..500)")

	// open flags
	lakesetOpenCmd.Flags().StringVar(&lakesetDataset, "dataset", "", "Dataset to open in Lakeset Explore (catalog.schema.table)")
	lakesetOpenCmd.Flags().StringVar(&lakesetSQL, "sql", "", "SQL query to open in Lakeset SQL Lab")
	lakesetOpenCmd.Flags().StringVar(&lakesetTab, "tab", "", "Target studio tab (datasets, explore, dashboards, sqllab)")
	lakesetOpenCmd.Flags().BoolVar(&lakesetNoBrowser, "no-browser", false, "Do not launch the default OS browser")
	lakesetOpenCmd.Flags().BoolVar(&lakesetPrintOnly, "print", false, "Print the deep-link URL only without launching a browser")

	lakesetCmd.AddCommand(lakesetStatusCmd)
	lakesetCmd.AddCommand(lakesetDatasetsCmd)
	lakesetCmd.AddCommand(lakesetSyncCmd)
	lakesetCmd.AddCommand(lakesetGuestTokenCmd)
	lakesetCmd.AddCommand(lakesetExploreCmd)
	lakesetCmd.AddCommand(lakesetOpenCmd)

	rootCmd.AddCommand(lakesetCmd)
}
