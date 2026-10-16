package cmd

import (
	"encoding/json"
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

		ui.PrintInfo("Checking Lake of Tears at %s ...", ui.TitleStyle.Render(ctx.Endpoint))

		health, err := c.Health()
		if err != nil {
			ui.PrintError("Connection failed: %v", err)
			return err
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(health, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		ui.PrintSuccess("Connected to Lake of Tears successfully!")
		fmt.Printf("Endpoint: %s\n", ctx.Endpoint)
		fmt.Printf("Status:   %s\n", ui.SuccessStyle.Render(health.Status))
		if health.Version != "" {
			fmt.Printf("Version:  %s\n", health.Version)
		}
		if len(health.Services) > 0 {
			fmt.Println("\nServices:")
			for svc, st := range health.Services {
				fmt.Printf("  • %-12s: %s\n", svc, st)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
