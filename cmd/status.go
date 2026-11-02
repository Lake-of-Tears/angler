package cmd

import (
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"ping"},
	Short:   "Check the health and status of the Lake of Tears instance",
	Long:    "Ping the Lake of Tears API endpoint and verify cluster connectivity and status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		ctx := getActiveContext()

		ui.PrintInfo("Checking Lake of Tears at %s ...", ui.StyleText(ui.TitleStyle, ctx.Endpoint))

		health, err := c.Health()
		if err != nil {
			return ui.NewConnectionError(err)
		}

		return ui.RenderOutput(health, func() {
			ui.PrintSuccess("Connected to Lake of Tears successfully!")
			fmt.Fprintf(ui.OutWriter, "Endpoint: %s\n", ctx.Endpoint)
			fmt.Fprintf(ui.OutWriter, "Status:   %s\n", ui.StyleText(ui.SuccessStyle, health.Status))
			if health.Version != "" {
				fmt.Fprintf(ui.OutWriter, "Version:  %s\n", health.Version)
			}
			if len(health.Services) > 0 {
				fmt.Fprintln(ui.OutWriter, "\nServices:")
				for svc, st := range health.Services {
					fmt.Fprintf(ui.OutWriter, "  • %-12s: %s\n", svc, st)
				}
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
