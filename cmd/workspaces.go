package cmd

import (
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	wsAdminName        string
	wsAdminSlug        string
	wsAdminDescription string
	wsMemberEmail      string
	wsMemberRole       string
	wsMemberForce      bool
)

var workspacesCmd = &cobra.Command{
	Use:     "workspaces",
	Aliases: []string{"ws-admin"},
	Short:   "Lake of Tears Multi-Tenant Workspace Provisioning and IAM Members",
	Long: `Manage Lake of Tears multi-tenant workspace environments, access memberships,
and tenant boundaries.

Aliases:
  angler workspaces
  angler ws-admin

Examples:
  angler workspaces list
  angler workspaces get ws-prod
  angler workspaces create --name "Risk Analytics" --slug risk-analytics
  angler workspaces members list ws-prod
  angler workspaces members add ws-prod --email analyst@lot.io --role editor
  angler workspaces members remove ws-prod u-123`,
}

var workspacesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all workspaces accessible to the current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		workspaces, err := c.ListAllWorkspaces()
		if err != nil {
			return err
		}

		return ui.RenderOutput(workspaces, func() {
			if len(workspaces) == 0 {
				ui.PrintInfo("No workspaces found.")
				return
			}

			headers := []string{"ID", "NAME", "SLUG", "ROLE", "STATUS", "CREATED"}
			rows := make([][]string, len(workspaces))
			for i, ws := range workspaces {
				role := "N/A"
				if ws.MyRole != nil {
					role = *ws.MyRole
				}
				rows[i] = []string{
					ws.ID,
					ws.Name,
					ws.Slug,
					role,
					ws.Status,
					ws.CreatedAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total workspaces: %d", len(workspaces))
		})
	},
}

var workspacesGetCmd = &cobra.Command{
	Use:   "get <workspace_id>",
	Short: "Inspect workspace details, storage boundaries, and configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := args[0]
		c := getClient()

		ws, err := c.GetWorkspace(workspaceID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(ws, func() {
			role := "N/A"
			if ws.MyRole != nil {
				role = *ws.MyRole
			}
			desc := "None"
			if ws.Description != nil && *ws.Description != "" {
				desc = *ws.Description
			}

			ui.PrintSuccess("Workspace: %s (%s)", ws.Name, ws.Status)
			fmt.Fprintf(ui.OutWriter, "ID:          %s\n", ws.ID)
			fmt.Fprintf(ui.OutWriter, "Slug:        %s\n", ws.Slug)
			fmt.Fprintf(ui.OutWriter, "My Role:     %s\n", role)
			fmt.Fprintf(ui.OutWriter, "Description: %s\n", desc)
			fmt.Fprintf(ui.OutWriter, "Is System:   %v\n", ws.IsSystem)
			fmt.Fprintf(ui.OutWriter, "Shared:      %v\n", ws.IsShared)
			fmt.Fprintf(ui.OutWriter, "Created At:  %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05"))
		})
	},
}

var workspacesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Provision a new analytical workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(wsAdminName) == "" {
			return ui.NewUsageError(fmt.Errorf("--name is required to create a workspace"))
		}

		c := getClient()
		req := client.CreateWorkspaceRequest{
			Name:        wsAdminName,
			Slug:        wsAdminSlug,
			Description: wsAdminDescription,
		}

		ws, err := c.CreateWorkspace(req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(ws, func() {
			ui.PrintSuccess("Workspace %q created successfully (%s)", ws.Name, ws.ID)
			fmt.Fprintf(ui.OutWriter, "Slug:   %s\n", ws.Slug)
			fmt.Fprintf(ui.OutWriter, "Status: %s\n", ws.Status)
		})
	},
}

// ── Members Subcommands ─────────────────────────────────────────────────────

var workspacesMembersCmd = &cobra.Command{
	Use:     "members",
	Aliases: []string{"member"},
	Short:   "Manage workspace user memberships and role assignments",
}

var workspacesMembersListCmd = &cobra.Command{
	Use:   "list <workspace_id>",
	Short: "List all members and roles in a workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := args[0]
		c := getClient()

		members, err := c.ListWorkspaceMembers(workspaceID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(members, func() {
			if len(members) == 0 {
				ui.PrintInfo("No members found in workspace %q.", workspaceID)
				return
			}

			headers := []string{"USER ID", "EMAIL", "ROLE", "GLOBAL ROLE", "JOINED"}
			rows := make([][]string, len(members))
			for i, m := range members {
				rows[i] = []string{
					m.UserID,
					m.Email,
					m.Role,
					m.GlobalRole,
					m.JoinedAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total members: %d", len(members))
		})
	},
}

var workspacesMembersAddCmd = &cobra.Command{
	Use:   "add <workspace_id>",
	Short: "Assign a user to a workspace with a specific role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := args[0]
		if strings.TrimSpace(wsMemberEmail) == "" {
			return ui.NewUsageError(fmt.Errorf("--email is required"))
		}
		if strings.TrimSpace(wsMemberRole) == "" {
			wsMemberRole = "viewer"
		}

		c := getClient()
		req := client.AddWorkspaceMemberRequest{
			Email: wsMemberEmail,
			Role:  wsMemberRole,
		}

		if err := c.AddWorkspaceMember(workspaceID, req); err != nil {
			return err
		}

		ui.PrintSuccess("User %q added to workspace %q with role %q", wsMemberEmail, workspaceID, wsMemberRole)
		return nil
	},
}

var workspacesMembersRemoveCmd = &cobra.Command{
	Use:     "remove <workspace_id> <user_id>",
	Aliases: []string{"rm"},
	Short:   "Remove a user from a workspace",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := args[0]
		userID := args[1]
		c := getClient()

		if !wsMemberForce && !confirmAction(fmt.Sprintf("Remove user '%s' from workspace '%s'?", userID, workspaceID)) {
			ui.PrintInfo("Operation cancelled.")
			return nil
		}

		if err := c.RemoveWorkspaceMember(workspaceID, userID); err != nil {
			return err
		}

		ui.PrintSuccess("User '%s' removed from workspace '%s'", userID, workspaceID)
		return nil
	},
}

func init() {
	// Create workspace flags
	workspacesCreateCmd.Flags().StringVar(&wsAdminName, "name", "", "Workspace name")
	workspacesCreateCmd.Flags().StringVar(&wsAdminSlug, "slug", "", "Workspace slug URL identifier")
	workspacesCreateCmd.Flags().StringVar(&wsAdminDescription, "description", "", "Workspace description")

	// Member add flags
	workspacesMembersAddCmd.Flags().StringVar(&wsMemberEmail, "email", "", "User email address to assign")
	workspacesMembersAddCmd.Flags().StringVar(&wsMemberRole, "role", "viewer", "Workspace role: admin, editor, or viewer")

	// Member remove flags
	workspacesMembersRemoveCmd.Flags().BoolVarP(&wsMemberForce, "force", "f", false, "Remove without confirmation")

	// Members subcommands
	workspacesMembersCmd.AddCommand(workspacesMembersListCmd)
	workspacesMembersCmd.AddCommand(workspacesMembersAddCmd)
	workspacesMembersCmd.AddCommand(workspacesMembersRemoveCmd)

	// Workspaces subcommands
	workspacesCmd.AddCommand(workspacesListCmd)
	workspacesCmd.AddCommand(workspacesGetCmd)
	workspacesCmd.AddCommand(workspacesCreateCmd)
	workspacesCmd.AddCommand(workspacesMembersCmd)

	rootCmd.AddCommand(workspacesCmd)
}
