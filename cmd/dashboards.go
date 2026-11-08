package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	dashTitle       string
	dashDescription string
	dashTemplate    string
	dashSpecFile    string
	dashPublished   bool
	dashForce       bool
	dashTimeRange   string
)

var dashboardsCmd = &cobra.Command{
	Use:     "dashboards",
	Aliases: []string{"dashboard", "dash"},
	Short:   "Lake of Tears BI Dashboards, Canvas Layouts, and Visual Widgets",
	Long: `Manage Lake of Tears native BI dashboards, canvas widget configurations,
and live aggregated analytical datasets.

Modeled after Databricks Dashboards, allowing data engineers, business analysts,
and autonomous LLM agents to inspect visualization layouts, extract live widget
query metrics, and programmatically deploy dashboards.

Aliases:
  angler dashboards
  angler dashboard
  angler dash

Examples:
  angler dashboards list
  angler dashboards get dash-123
  angler dashboards create --title "Executive KPIs" --template executive
  angler dashboards create --spec ./dashboards/sales_overview.json
  angler dashboards delete dash-123 --force
  angler dashboards widgets list dash-123
  angler dashboards widgets get-data dash-123 w_kpi_1 --time-range last_30_days`,
}

var dashboardsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all registered BI dashboards in the active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		dashboards, err := c.ListDashboards()
		if err != nil {
			return err
		}

		return ui.RenderOutput(dashboards, func() {
			if len(dashboards) == 0 {
				ui.PrintInfo("No dashboards found in current workspace.")
				return
			}

			headers := []string{"ID", "TITLE", "SLUG", "CHARTS", "PUBLISHED", "UPDATED"}
			rows := make([][]string, len(dashboards))
			for i, d := range dashboards {
				pub := "No"
				if d.IsPublished {
					pub = "Yes"
				}
				rows[i] = []string{
					d.ID,
					d.Title,
					d.Slug,
					fmt.Sprintf("%d", d.ChartsCount),
					pub,
					d.UpdatedAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total dashboards: %d", len(dashboards))
		})
	},
}

var dashboardsGetCmd = &cobra.Command{
	Use:   "get <dashboard_id>",
	Short: "Retrieve complete dashboard details, widgets, and layout configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dashboardID := args[0]
		c := getClient()

		dash, err := c.GetDashboard(dashboardID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(dash, func() {
			pubStatus := "Draft"
			if dash.IsPublished {
				pubStatus = "Published"
			}

			ui.PrintSuccess("Dashboard: %s (%s)", dash.Title, pubStatus)
			fmt.Fprintf(ui.OutWriter, "ID:          %s\n", dash.ID)
			fmt.Fprintf(ui.OutWriter, "Slug:        %s\n", dash.Slug)
			if dash.Description != "" {
				fmt.Fprintf(ui.OutWriter, "Description: %s\n", dash.Description)
			}
			fmt.Fprintf(ui.OutWriter, "Charts:      %d\n", dash.ChartsCount)
			fmt.Fprintf(ui.OutWriter, "Created At:  %s\n", dash.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(ui.OutWriter, "Updated At:  %s\n", dash.UpdatedAt.Format("2006-01-02 15:04:05"))

			if len(dash.LayoutConfig) > 0 {
				fmt.Fprintf(ui.OutWriter, "\n%s\n", ui.StyleText(ui.TitleStyle, "Widgets & Canvas Layout:"))
				headers := []string{"WIDGET ID", "TITLE", "TYPE", "QUERY / SOURCE"}
				rows := make([][]string, len(dash.LayoutConfig))
				for i, w := range dash.LayoutConfig {
					wID, _ := w["id"].(string)
					title, _ := w["title"].(string)
					wType, _ := w["type"].(string)
					query, _ := w["query"].(string)
					if query == "" {
						query, _ = w["dataset"].(string)
					}
					rows[i] = []string{wID, title, wType, query}
				}
				fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			}
		})
	},
}

var dashboardsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Deploy a new native BI dashboard from specification or template",
	Long: `Creates a new dashboard in the active workspace.
Optionally load a complete specification from a JSON/YAML file, or initialize
from built-in templates ('blank', 'executive', 'traffic').`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()

		var req client.CreateDashboardRequest
		req.Title = dashTitle
		req.Description = dashDescription
		req.IsPublished = dashPublished
		req.Template = dashTemplate

		if dashSpecFile != "" {
			var specBytes []byte
			var err error
			if dashSpecFile == "-" {
				specBytes, err = io.ReadAll(os.Stdin)
			} else {
				specBytes, err = os.ReadFile(dashSpecFile)
			}
			if err != nil {
				return fmt.Errorf("failed to read spec file: %w", err)
			}

			// Support both JSON and YAML spec formats
			var specData map[string]any
			if err := json.Unmarshal(specBytes, &specData); err != nil {
				if yamlErr := yaml.Unmarshal(specBytes, &specData); yamlErr != nil {
					return fmt.Errorf("failed to parse spec as JSON or YAML: %w", err)
				}
			}

			if title, ok := specData["title"].(string); ok && title != "" && req.Title == "" {
				req.Title = title
			}
			if desc, ok := specData["description"].(string); ok && desc != "" && req.Description == "" {
				req.Description = desc
			}
			if pub, ok := specData["is_published"].(bool); ok {
				req.IsPublished = pub
			}
			if tmpl, ok := specData["template"].(string); ok && tmpl != "" {
				req.Template = tmpl
			}
			if layout, ok := specData["layout_config"].([]any); ok {
				for _, item := range layout {
					if m, ok := item.(map[string]any); ok {
						req.LayoutConfig = append(req.LayoutConfig, m)
					}
				}
			}
		}

		if strings.TrimSpace(req.Title) == "" {
			return ui.NewUsageError(fmt.Errorf("--title is required to create a dashboard"))
		}

		dash, err := c.CreateDashboard(req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(dash, func() {
			ui.PrintSuccess("Dashboard created successfully: %s (%s)", dash.Title, dash.ID)
			fmt.Fprintf(ui.OutWriter, "Slug:        %s\n", dash.Slug)
			fmt.Fprintf(ui.OutWriter, "Charts:      %d\n", dash.ChartsCount)
			fmt.Fprintf(ui.OutWriter, "Published:   %v\n", dash.IsPublished)
		})
	},
}

var dashboardsDeleteCmd = &cobra.Command{
	Use:   "delete <dashboard_id>",
	Short: "Delete a dashboard from the current workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dashboardID := args[0]
		c := getClient()

		if !dashForce && !confirmAction(fmt.Sprintf("Permanently delete dashboard '%s'?", dashboardID)) {
			ui.PrintInfo("Deletion aborted.")
			return nil
		}

		if err := c.DeleteDashboard(dashboardID); err != nil {
			return err
		}

		ui.PrintSuccess("Dashboard %q deleted successfully.", dashboardID)
		return nil
	},
}

var dashboardsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display aggregate dashboard and chart metrics for the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		stats, err := c.GetDashboardStats()
		if err != nil {
			return err
		}

		return ui.RenderOutput(stats, func() {
			onlineStatus := "Online"
			if !stats.SupersetOnline {
				onlineStatus = "Offline"
			}
			ui.PrintSuccess("Workspace Dashboard Statistics")
			fmt.Fprintf(ui.OutWriter, "Total Dashboards:     %d\n", stats.TotalDashboards)
			fmt.Fprintf(ui.OutWriter, "Published Dashboards: %d\n", stats.PublishedDashboards)
			fmt.Fprintf(ui.OutWriter, "Total Charts/Widgets: %d\n", stats.TotalCharts)
			fmt.Fprintf(ui.OutWriter, "Engine Health:        %s\n", onlineStatus)
		})
	},
}

// ── Widgets Subcommand Group ────────────────────────────────────────────────

var dashboardsWidgetsCmd = &cobra.Command{
	Use:     "widgets",
	Aliases: []string{"widget"},
	Short:   "Inspect dashboard widget layout and execute live visualization queries",
}

var dashboardsWidgetsListCmd = &cobra.Command{
	Use:   "list <dashboard_id>",
	Short: "List all widgets and visualization queries in a dashboard",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dashboardID := args[0]
		c := getClient()

		dash, err := c.GetDashboard(dashboardID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(dash.LayoutConfig, func() {
			if len(dash.LayoutConfig) == 0 {
				ui.PrintInfo("No widgets configured in dashboard %q.", dash.Title)
				return
			}

			headers := []string{"WIDGET ID", "TITLE", "TYPE", "WIDTH", "QUERY / SOURCE"}
			rows := make([][]string, len(dash.LayoutConfig))
			for i, w := range dash.LayoutConfig {
				wID, _ := w["id"].(string)
				title, _ := w["title"].(string)
				wType, _ := w["type"].(string)
				width, _ := w["width"].(string)
				if width == "" {
					width = "half"
				}
				query, _ := w["query"].(string)
				if query == "" {
					query, _ = w["dataset"].(string)
				}
				rows[i] = []string{wID, title, wType, width, query}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total widgets: %d", len(dash.LayoutConfig))
		})
	},
}

var dashboardsWidgetsGetDataCmd = &cobra.Command{
	Use:   "get-data <dashboard_id> <widget_id>",
	Short: "Execute widget analytical query and return live aggregated chart data",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dashboardID := args[0]
		widgetID := args[1]
		c := getClient()

		resp, err := c.GetWidgetData(dashboardID, widgetID, dashTimeRange)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Widget Data: %s (ID: %s, Type: %s)", resp.Title, resp.WidgetID, resp.Type)

			// If data contains rows and columns array
			if rowsRaw, ok := resp.Data["rows"].([]any); ok && len(rowsRaw) > 0 {
				if colsRaw, ok := resp.Data["columns"].([]any); ok && len(colsRaw) > 0 {
					headers := make([]string, len(colsRaw))
					for i, col := range colsRaw {
						headers[i] = fmt.Sprintf("%v", col)
					}
					rows := make([][]string, len(rowsRaw))
					for i, r := range rowsRaw {
						if rowSlice, ok := r.([]any); ok {
							rows[i] = make([]string, len(rowSlice))
							for j, val := range rowSlice {
								rows[i][j] = fmt.Sprintf("%v", val)
							}
						}
					}
					fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
					return
				}
			}

			// Fallback: pretty-print data keys and values
			for k, v := range resp.Data {
				valStr := fmt.Sprintf("%v", v)
				if f, ok := v.(float64); ok && f == float64(int64(f)) {
					valStr = fmt.Sprintf("%.0f", f)
				}
				fmt.Fprintf(ui.OutWriter, "  • %s: %s\n", ui.StyleText(ui.TitleStyle, k), valStr)
			}
		})
	},
}

func init() {
	// Create flags
	dashboardsCreateCmd.Flags().StringVar(&dashTitle, "title", "", "Dashboard title")
	dashboardsCreateCmd.Flags().StringVar(&dashDescription, "description", "", "Dashboard description")
	dashboardsCreateCmd.Flags().StringVar(&dashTemplate, "template", "blank", "Starter template: blank, executive, traffic")
	dashboardsCreateCmd.Flags().StringVar(&dashSpecFile, "spec", "", "Path to dashboard JSON or YAML spec file (or '-' for stdin)")
	dashboardsCreateCmd.Flags().BoolVar(&dashPublished, "published", true, "Publish status (default true)")

	// Delete flags
	dashboardsDeleteCmd.Flags().BoolVarP(&dashForce, "force", "f", false, "Delete without confirmation")

	// Widget get-data flags
	dashboardsWidgetsGetDataCmd.Flags().StringVar(&dashTimeRange, "time-range", "", "Time window filter (e.g. 'last_30_days', 'last_7_days')")

	// Widgets subcommand hierarchy
	dashboardsWidgetsCmd.AddCommand(dashboardsWidgetsListCmd)
	dashboardsWidgetsCmd.AddCommand(dashboardsWidgetsGetDataCmd)

	// Dashboards subcommand hierarchy
	dashboardsCmd.AddCommand(dashboardsListCmd)
	dashboardsCmd.AddCommand(dashboardsGetCmd)
	dashboardsCmd.AddCommand(dashboardsCreateCmd)
	dashboardsCmd.AddCommand(dashboardsDeleteCmd)
	dashboardsCmd.AddCommand(dashboardsStatsCmd)
	dashboardsCmd.AddCommand(dashboardsWidgetsCmd)

	rootCmd.AddCommand(dashboardsCmd)
}
