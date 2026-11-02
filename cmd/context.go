package cmd

import (
	"fmt"
	"sort"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

type ContextItem struct {
	Name     string `json:"name" yaml:"name"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Email    string `json:"email,omitempty" yaml:"email,omitempty"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Active   bool   `json:"active" yaml:"active"`
}

type ContextActionResult struct {
	Status         string `json:"status" yaml:"status"`
	Context        string `json:"context" yaml:"context"`
	CurrentContext string `json:"current_context" yaml:"current_context"`
	Endpoint       string `json:"endpoint" yaml:"endpoint"`
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage connection environments and profiles",
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := make([]string, 0, len(cfg.Contexts))
		for name := range cfg.Contexts {
			names = append(names, name)
		}
		sort.Strings(names)

		items := make([]ContextItem, 0, len(names))
		for _, name := range names {
			c := cfg.Contexts[name]
			items = append(items, ContextItem{
				Name:     name,
				Endpoint: c.Endpoint,
				Email:    c.Email,
				Timeout:  c.Timeout,
				Active:   name == cfg.CurrentContext,
			})
		}

		return ui.RenderOutput(items, func() {
			ui.PrintInfo("Configured Lake of Tears contexts:")
			for _, item := range items {
				prefix := "  "
				if item.Active {
					prefix = "* "
				}
				fmt.Fprintf(ui.OutWriter, "%s%-15s %s (user: %s)\n", prefix, item.Name, item.Endpoint, item.Email)
			}
		})
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active context profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		ctx, exists := cfg.Contexts[target]
		if !exists {
			return ui.NewNotFoundError(fmt.Errorf("context %q not found", target))
		}

		cfg.CurrentContext = target
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		result := ContextActionResult{
			Status:         "switched",
			Context:        target,
			CurrentContext: cfg.CurrentContext,
			Endpoint:       ctx.Endpoint,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Switched to context %q.", target)
		})
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
		if current.Timeout == 0 {
			current.Timeout = 30
		}

		cfg.Contexts[name] = current
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		result := ContextActionResult{
			Status:         "saved",
			Context:        name,
			CurrentContext: cfg.CurrentContext,
			Endpoint:       current.Endpoint,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Saved context %q with endpoint %s.", name, current.Endpoint)
		})
	},
}

func init() {
	contextSetCmd.Flags().StringVarP(&newEndpoint, "endpoint", "e", "", "API endpoint URL")
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextSetCmd)
	rootCmd.AddCommand(contextCmd)
}
