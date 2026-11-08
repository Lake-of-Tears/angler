package cmd

import (
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	adminSessionTimeout int
	adminMaxHours       int
	adminGeminiModel    string
	adminGeminiKey      string
	adminForce          bool
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Lake of Tears System Administration, Settings, and Session Auditing",
	Long: `Administrative command suite for cluster superadmins.
Inspect and update cluster-wide runtime settings, review active sessions,
and revoke compromised tokens or connections.

Requires cluster superadmin privileges.

Examples:
  angler admin settings get
  angler admin settings update --session-timeout 120 --gemini-model gemini-2.0-flash
  angler admin sessions list
  angler admin sessions revoke sess-123 --force`,
}

// ── Settings Subcommands ────────────────────────────────────────────────────

var adminSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Inspect and update system-wide configuration settings",
}

var adminSettingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve active cluster-wide configuration parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		settings, err := c.GetSystemSettings()
		if err != nil {
			return err
		}

		return ui.RenderOutput(settings, func() {
			geminiConfigured := "No"
			if settings.GeminiAPIKeyConfigured {
				geminiConfigured = "Yes"
			}
			model := "None"
			if settings.GeminiDefaultModel != nil && *settings.GeminiDefaultModel != "" {
				model = *settings.GeminiDefaultModel
			}

			ui.PrintSuccess("Cluster System Settings")
			fmt.Fprintf(ui.OutWriter, "Catalog Soft-Delete Days:       %d\n", settings.CatalogSoftDeleteDays)
			fmt.Fprintf(ui.OutWriter, "Workspace Inactive Grace Days:  %d\n", settings.WorkspaceInactiveGracePeriodDays)
			fmt.Fprintf(ui.OutWriter, "Session Inactivity Timeout:     %d minutes\n", settings.SessionInactivityTimeoutMinutes)
			fmt.Fprintf(ui.OutWriter, "Session Maximum Lifetime:       %d hours\n", settings.SessionMaxHours)
			fmt.Fprintf(ui.OutWriter, "Gemini API Key Configured:      %s\n", geminiConfigured)
			if settings.GeminiAPIKeyMasked != nil {
				fmt.Fprintf(ui.OutWriter, "Gemini Key Masked:              %s\n", *settings.GeminiAPIKeyMasked)
			}
			fmt.Fprintf(ui.OutWriter, "Default AI Model:               %s\n", model)
		})
	},
}

var adminSettingsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update cluster-wide configuration settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()

		var req client.UpdateSystemSettingsRequest
		if cmd.Flags().Changed("session-timeout") {
			req.SessionInactivityTimeoutMinutes = &adminSessionTimeout
		}
		if cmd.Flags().Changed("max-hours") {
			req.SessionMaxHours = &adminMaxHours
		}
		if cmd.Flags().Changed("gemini-model") {
			req.GeminiDefaultModel = &adminGeminiModel
		}
		if cmd.Flags().Changed("gemini-key") {
			req.GeminiAPIKey = &adminGeminiKey
		}

		settings, err := c.UpdateSystemSettings(req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(settings, func() {
			ui.PrintSuccess("System settings updated successfully.")
			fmt.Fprintf(ui.OutWriter, "Session Inactivity Timeout: %d minutes\n", settings.SessionInactivityTimeoutMinutes)
			fmt.Fprintf(ui.OutWriter, "Session Maximum Lifetime:   %d hours\n", settings.SessionMaxHours)
			if settings.GeminiDefaultModel != nil {
				fmt.Fprintf(ui.OutWriter, "Default AI Model:           %s\n", *settings.GeminiDefaultModel)
			}
		})
	},
}

// ── Sessions Subcommands ────────────────────────────────────────────────────

var adminSessionsCmd = &cobra.Command{
	Use:     "sessions",
	Aliases: []string{"session"},
	Short:   "Audit and manage active user and API sessions",
}

var adminSessionsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List active user sessions across the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		sessions, err := c.ListSessions()
		if err != nil {
			return err
		}

		return ui.RenderOutput(sessions, func() {
			if len(sessions) == 0 {
				ui.PrintInfo("No active sessions found.")
				return
			}

			headers := []string{"SESSION ID", "USER", "IP ADDRESS", "CURRENT", "EXPIRES AT"}
			rows := make([][]string, len(sessions))
			for i, s := range sessions {
				userStr := s.UserID
				if s.Email != nil && *s.Email != "" {
					userStr = *s.Email
				}
				ipStr := "N/A"
				if s.IPAddress != nil && *s.IPAddress != "" {
					ipStr = *s.IPAddress
				}
				currentStr := "No"
				if s.IsCurrent {
					currentStr = "Yes"
				}

				rows[i] = []string{
					s.ID,
					userStr,
					ipStr,
					currentStr,
					s.ExpiresAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total active sessions: %d", len(sessions))
		})
	},
}

var adminSessionsRevokeCmd = &cobra.Command{
	Use:   "revoke <session_id>",
	Short: "Revoke an active user or service session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		c := getClient()

		if !adminForce && !confirmAction(fmt.Sprintf("Revoke session '%s'?", sessionID)) {
			ui.PrintInfo("Operation cancelled.")
			return nil
		}

		if err := c.RevokeSession(sessionID); err != nil {
			return err
		}

		ui.PrintSuccess("Session '%s' revoked successfully.", sessionID)
		return nil
	},
}

func init() {
	// Settings update flags
	adminSettingsUpdateCmd.Flags().IntVar(&adminSessionTimeout, "session-timeout", 60, "Session inactivity timeout in minutes")
	adminSettingsUpdateCmd.Flags().IntVar(&adminMaxHours, "max-hours", 24, "Maximum session duration in hours")
	adminSettingsUpdateCmd.Flags().StringVar(&adminGeminiModel, "gemini-model", "", "Default Gemini AI model")
	adminSettingsUpdateCmd.Flags().StringVar(&adminGeminiKey, "gemini-key", "", "Gemini API key")

	// Session revoke flags
	adminSessionsRevokeCmd.Flags().BoolVarP(&adminForce, "force", "f", false, "Revoke session without confirmation prompt")

	// Settings hierarchy
	adminSettingsCmd.AddCommand(adminSettingsGetCmd)
	adminSettingsCmd.AddCommand(adminSettingsUpdateCmd)

	// Sessions hierarchy
	adminSessionsCmd.AddCommand(adminSessionsListCmd)
	adminSessionsCmd.AddCommand(adminSessionsRevokeCmd)

	// Admin hierarchy
	adminCmd.AddCommand(adminSettingsCmd)
	adminCmd.AddCommand(adminSessionsCmd)

	rootCmd.AddCommand(adminCmd)
}
