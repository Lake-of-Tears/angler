package cmd

import (
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage connection environments and profiles",
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured contexts",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintInfo("Configured Lake of Tears contexts:")
		for name, c := range cfg.Contexts {
			prefix := "  "
			if name == cfg.CurrentContext {
				prefix = "* "
			}
			fmt.Printf("%s%-15s %s (user: %s)\n", prefix, name, c.Endpoint, c.Email)
		}
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active context profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		if _, exists := cfg.Contexts[target]; !exists {
			return fmt.Errorf("context %q not found", target)
		}

		cfg.CurrentContext = target
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		ui.PrintSuccess("Switched to context %q.", target)
		return nil
	},
}

var (
	newEndpoint string
)

var contextSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Create or update a context profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		current := cfg.Contexts[name]

		if newEndpoint != "" {
			current.Endpoint = newEndpoint
		}
		if current.Endpoint == "" {
			current.Endpoint = config.DefaultEndpoint
		}

		cfg.Contexts[name] = current
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		ui.PrintSuccess("Saved context %q with endpoint %s.", name, current.Endpoint)
		return nil
	},
}

func init() {
	contextSetCmd.Flags().StringVarP(&newEndpoint, "endpoint", "e", "", "API endpoint URL")
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextSetCmd)
	rootCmd.AddCommand(contextCmd)
}
