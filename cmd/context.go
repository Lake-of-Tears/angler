package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

type ContextItem struct {
	Name        string `json:"name" yaml:"name"`
	Endpoint    string `json:"endpoint" yaml:"endpoint"`
	Workspace   string `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	Email       string `json:"email,omitempty" yaml:"email,omitempty"`
	Timeout     int    `json:"timeout" yaml:"timeout"`
	Active      bool   `json:"active" yaml:"active"`
}

type ContextActionResult struct {
	Status         string `json:"status" yaml:"status"`
	Context        string `json:"context" yaml:"context"`
	CurrentContext string `json:"current_context" yaml:"current_context"`
	Endpoint       string `json:"endpoint" yaml:"endpoint"`
	Workspace      string `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	WorkspaceID    string `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
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
				Name:        name,
				Endpoint:    c.Endpoint,
				Workspace:   c.Workspace,
				WorkspaceID: c.WorkspaceID,
				Email:       c.Email,
				Timeout:     c.Timeout,
				Active:      name == cfg.CurrentContext,
			})
		}

		return ui.RenderOutput(items, func() {
			ui.PrintInfo("Configured Lake of Tears contexts:")
			for _, item := range items {
				prefix := "  "
				if item.Active {
					prefix = "* "
				}
				details := []string{}
				if item.Workspace != "" {
					details = append(details, fmt.Sprintf("workspace: %s", item.Workspace))
				}
				if item.Email != "" {
					details = append(details, fmt.Sprintf("user: %s", item.Email))
				}
				detailsStr := ""
				if len(details) > 0 {
					detailsStr = fmt.Sprintf(" (%s)", strings.Join(details, ", "))
				}
				fmt.Fprintf(ui.OutWriter, "%s%-15s %s%s\n", prefix, item.Name, item.Endpoint, detailsStr)
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
	newEndpoint    string
	newWorkspace   string
	newWorkspaceID string
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
		if cmd.Flags().Changed("workspace") {
			current.Workspace = newWorkspace
		}
		if cmd.Flags().Changed("workspace-id") {
			current.WorkspaceID = newWorkspaceID
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
			Workspace:      current.Workspace,
			WorkspaceID:    current.WorkspaceID,
		}

		return ui.RenderOutput(result, func() {
			wsMsg := ""
			if current.Workspace != "" {
				wsMsg = fmt.Sprintf(" (workspace: %s)", current.Workspace)
			}
			ui.PrintSuccess("Saved context %q with endpoint %s%s.", name, current.Endpoint, wsMsg)
		})
	},
}

func init() {
	contextSetCmd.Flags().StringVarP(&newEndpoint, "endpoint", "e", "", "API endpoint URL")
	contextSetCmd.Flags().StringVarP(&newWorkspace, "workspace", "w", "", "workspace slug or name")
	contextSetCmd.Flags().StringVar(&newWorkspaceID, "workspace-id", "", "workspace ID or UUID")
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextSetCmd)
	rootCmd.AddCommand(contextCmd)
}
