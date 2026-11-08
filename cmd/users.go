package cmd

import (
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	userRole        string
	userDisplayName string
	userActive      bool
	userActiveSet   bool
)

var usersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "Lake of Tears User and Identity Management (IAM)",
	Long: `Manage Lake of Tears user accounts, global roles, active status, and credentials.
Modeled after Databricks Users and IAM controls.

Requires cluster superadmin privileges.

Aliases:
  angler users
  angler user

Examples:
  angler users list
  angler users get u-123
  angler users update u-123 --role superadmin
  angler users update u-123 --name "Jane Doe"`,
}

var usersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all registered user accounts across the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		users, err := c.ListUsers()
		if err != nil {
			return err
		}

		return ui.RenderOutput(users, func() {
			if len(users) == 0 {
				ui.PrintInfo("No users found.")
				return
			}

			headers := []string{"ID", "EMAIL", "ROLE", "ACTIVE", "DISPLAY NAME", "CREATED"}
			rows := make([][]string, len(users))
			for i, u := range users {
				name := ""
				if u.DisplayName != nil {
					name = *u.DisplayName
				}
				activeStr := "No"
				if u.IsActive {
					activeStr = "Yes"
				}
				rows[i] = []string{
					u.ID,
					u.Email,
					u.Role,
					activeStr,
					name,
					u.CreatedAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total users: %d", len(users))
		})
	},
}

var usersGetCmd = &cobra.Command{
	Use:   "get <user_id>",
	Short: "Inspect detailed user profile and authentication configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		c := getClient()

		user, err := c.GetUser(userID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(user, func() {
			name := "N/A"
			if user.DisplayName != nil && *user.DisplayName != "" {
				name = *user.DisplayName
			}
			activeStatus := "Active"
			if !user.IsActive {
				activeStatus = "Suspended / Inactive"
			}

			ui.PrintSuccess("User: %s (%s)", user.Email, user.Role)
			fmt.Fprintf(ui.OutWriter, "ID:           %s\n", user.ID)
			fmt.Fprintf(ui.OutWriter, "Display Name: %s\n", name)
			fmt.Fprintf(ui.OutWriter, "Status:       %s\n", activeStatus)
			fmt.Fprintf(ui.OutWriter, "Has Password: %v\n", user.HasPassword)
			if len(user.AuthProviders) > 0 {
				fmt.Fprintf(ui.OutWriter, "Auth Methods: %s\n", strings.Join(user.AuthProviders, ", "))
			}
			fmt.Fprintf(ui.OutWriter, "Created At:   %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
		})
	},
}

var usersUpdateCmd = &cobra.Command{
	Use:   "update <user_id>",
	Short: "Modify user role, active status, or display name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		c := getClient()

		var req client.UpdateUserRequest
		if userRole != "" {
			req.Role = &userRole
		}
		if userDisplayName != "" {
			req.DisplayName = &userDisplayName
		}
		if cmd.Flags().Changed("active") {
			req.IsActive = &userActive
		}

		user, err := c.UpdateUser(userID, req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(user, func() {
			ui.PrintSuccess("User %q updated successfully.", user.Email)
			fmt.Fprintf(ui.OutWriter, "Role:         %s\n", user.Role)
			fmt.Fprintf(ui.OutWriter, "Active:       %v\n", user.IsActive)
			if user.DisplayName != nil {
				fmt.Fprintf(ui.OutWriter, "Display Name: %s\n", *user.DisplayName)
			}
		})
	},
}

func init() {
	usersUpdateCmd.Flags().StringVar(&userRole, "role", "", "Role: 'superadmin' or 'user'")
	usersUpdateCmd.Flags().StringVar(&userDisplayName, "name", "", "User display name")
	usersUpdateCmd.Flags().BoolVar(&userActive, "active", true, "Account active status")

	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersGetCmd)
	usersCmd.AddCommand(usersUpdateCmd)

	rootCmd.AddCommand(usersCmd)
}
