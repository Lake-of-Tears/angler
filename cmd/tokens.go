package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	tokenName         string
	tokenComment      string
	tokenLifetimeDays int
	tokenWorkspace    string
	tokenRole         string
	tokenScopes       []string
	tokenIncludeAll   bool
	tokenForce        bool
)

type TokenCreateResult struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Token       string            `json:"token" yaml:"token"`
	TokenPrefix string            `json:"token_prefix" yaml:"token_prefix"`
	WorkspaceID string            `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	Workspaces  map[string]string `json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
	CreatedAt   string            `json:"created_at" yaml:"created_at"`
	ExpiresAt   *string           `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
}

type TokenListItem struct {
	ID         string  `json:"id" yaml:"id"`
	Name       string  `json:"name" yaml:"name"`
	Prefix     string  `json:"prefix" yaml:"prefix"`
	Workspaces string  `json:"workspaces" yaml:"workspaces"`
	CreatedAt  string  `json:"created_at" yaml:"created_at"`
	ExpiresAt  *string `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty" yaml:"last_used_at,omitempty"`
	Status     string  `json:"status" yaml:"status"`
}

type TokenRevokeResult struct {
	Status  string `json:"status" yaml:"status"`
	TokenID string `json:"token_id" yaml:"token_id"`
}

var tokensCmd = &cobra.Command{
	Use:     "tokens",
	Aliases: []string{"token", "pat"},
	Short:   "Manage Personal Access Tokens (PATs) for Lake of Tears",
	Long: `Create, list, and revoke Personal Access Tokens (PATs) for automated CI/CD pipelines,
autonomous agents, and headless scripts. Modeled after cloud token lifecycles.`,
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Generate a new Personal Access Token",
	Long: `Generate a new Personal Access Token (PAT) with scoped workspace permissions and lifetime.

The raw secret token (lot_pat_...) is displayed once upon creation and cannot be retrieved later.
Tokens are cryptographically hashed on the Lake of Tears server.

Examples:
  # Generate token with default 30-day lifetime in active workspace
  angler tokens create --name "ci-deployer"

  # Generate 90-day token with write role for a specific workspace
  angler tokens create --name "airflow-sync" --lifetime-days 90 --workspace default --role write

  # Machine-readable output returning token payload in JSON
  angler tokens create --name "agent-token" --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		if c.Token == "" {
			return ui.NewAuthError(fmt.Errorf("authentication required: run 'angler auth login' first"))
		}

		name := strings.TrimSpace(tokenName)
		if name == "" && strings.TrimSpace(tokenComment) != "" {
			name = strings.TrimSpace(tokenComment)
		}

		if name == "" {
			if ui.IsNonInteractive() {
				return ui.NewUsageError(fmt.Errorf("--name (or --comment) is required"))
			}

			reader := bufio.NewReader(os.Stdin)
			fmt.Fprint(ui.OutWriter, "Token Name / Description: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			name = strings.TrimSpace(input)
			if name == "" {
				return ui.NewUsageError(fmt.Errorf("token name cannot be empty"))
			}
		}

		if tokenLifetimeDays < 1 || tokenLifetimeDays > 3650 {
			return ui.NewUsageError(fmt.Errorf("--lifetime-days must be between 1 and 3650 (got %d)", tokenLifetimeDays))
		}

		role := strings.ToLower(strings.TrimSpace(tokenRole))
		if role == "" {
			role = "read"
		}
		if role != "read" && role != "write" && role != "admin" {
			return ui.NewUsageError(fmt.Errorf("invalid role %q: must be one of 'read', 'write', 'admin'", role))
		}

		// Resolve workspace to grant token access
		workspaces, err := c.ListWorkspaces()
		if err != nil {
			return err
		}

		var targetWorkspace *client.Workspace
		wsInput := strings.TrimSpace(tokenWorkspace)
		if wsInput == "" {
			wsInput = strings.TrimSpace(workspaceFlag)
		}
		if wsInput == "" {
			wsInput = strings.TrimSpace(getActiveContext().Workspace)
		}

		if wsInput != "" {
			for i := range workspaces {
				w := &workspaces[i]
				if w.ID == wsInput || strings.EqualFold(w.Slug, wsInput) || strings.EqualFold(w.Name, wsInput) {
					targetWorkspace = w
					break
				}
			}
			if targetWorkspace == nil {
				return ui.NewNotFoundError(fmt.Errorf("workspace %q not found or inaccessible", wsInput))
			}
		} else {
			if len(workspaces) == 0 {
				return ui.NewUsageError(fmt.Errorf("no accessible workspaces found to associate with token"))
			}
			// Look for default slug
			for i := range workspaces {
				if workspaces[i].Slug == "default" {
					targetWorkspace = &workspaces[i]
					break
				}
			}
			if targetWorkspace == nil {
				targetWorkspace = &workspaces[0]
			}
		}

		req := &client.TokenCreateRequest{
			Name:            name,
			LifetimeDays:    &tokenLifetimeDays,
			WorkspaceID:     &targetWorkspace.ID,
			WorkspaceIDs:    []string{targetWorkspace.ID},
			WorkspaceAccess: map[string]string{targetWorkspace.ID: role},
			Scopes:          tokenScopes,
		}

		created, err := c.CreateToken(req)
		if err != nil {
			return err
		}

		var expiresStr *string
		if created.ExpiresAt != nil {
			s := created.ExpiresAt.Format(time.RFC3339)
			expiresStr = &s
		}

		result := TokenCreateResult{
			ID:          created.ID,
			Name:        created.Name,
			Token:       created.Token,
			TokenPrefix: created.TokenPrefix,
			WorkspaceID: targetWorkspace.ID,
			Workspaces:  req.WorkspaceAccess,
			CreatedAt:   created.CreatedAt.Format(time.RFC3339),
			ExpiresAt:   expiresStr,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Personal Access Token generated successfully!")
			ui.PrintWarning("Copy this token now! It will NOT be shown again.")
			fmt.Fprintln(ui.OutWriter)
			fmt.Fprintf(ui.OutWriter, "Token:        %s\n", ui.StyleText(ui.SuccessStyle, created.Token))
			fmt.Fprintf(ui.OutWriter, "Token ID:     %s\n", created.ID)
			fmt.Fprintf(ui.OutWriter, "Name:         %s\n", created.Name)
			fmt.Fprintf(ui.OutWriter, "Workspace:    %s (%s)\n", targetWorkspace.Name, targetWorkspace.ID)
			fmt.Fprintf(ui.OutWriter, "Access Level: %s\n", role)
			if expiresStr != nil {
				fmt.Fprintf(ui.OutWriter, "Expires:      %s\n", *expiresStr)
			}
		})
	},
}

var tokensListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List active Personal Access Tokens registered to your account",
	Long: `List all Personal Access Tokens registered to the authenticated user.

Displays token ID, name, prefix, workspace associations, expiration, and last used timestamp.

Examples:
  angler tokens list
  angler tokens list --all
  angler tokens list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		if c.Token == "" {
			return ui.NewAuthError(fmt.Errorf("authentication required: run 'angler auth login' first"))
		}

		tokens, err := c.ListTokens(tokenIncludeAll)
		if err != nil {
			return err
		}

		now := time.Now()
		items := make([]TokenListItem, 0, len(tokens))
		tableRows := make([][]string, 0, len(tokens))

		for _, t := range tokens {
			status := "Active"
			if t.IsRevoked {
				status = "Revoked"
			} else if t.ExpiresAt != nil && t.ExpiresAt.Before(now) {
				status = "Expired"
			}

			wsSummary := "None"
			if len(t.WorkspaceNames) > 0 {
				wsSummary = strings.Join(t.WorkspaceNames, ", ")
			} else if t.WorkspaceName != nil {
				wsSummary = *t.WorkspaceName
			}

			var expiresStr *string
			expiresDisplay := "Never"
			if t.ExpiresAt != nil {
				s := t.ExpiresAt.Format("2006-01-02 15:04")
				expiresStr = &s
				expiresDisplay = s
			}

			var lastUsedStr *string
			lastUsedDisplay := "Never"
			if t.LastUsedAt != nil {
				s := t.LastUsedAt.Format("2006-01-02 15:04")
				lastUsedStr = &s
				lastUsedDisplay = s
			}

			createdDisplay := t.CreatedAt.Format("2006-01-02 15:04")

			items = append(items, TokenListItem{
				ID:         t.ID,
				Name:       t.Name,
				Prefix:     t.TokenPrefix + "…",
				Workspaces: wsSummary,
				CreatedAt:  createdDisplay,
				ExpiresAt:  expiresStr,
				LastUsedAt: lastUsedStr,
				Status:     status,
			})

			tableRows = append(tableRows, []string{
				t.ID,
				t.Name,
				t.TokenPrefix + "…",
				wsSummary,
				createdDisplay,
				expiresDisplay,
				lastUsedDisplay,
				status,
			})
		}

		return ui.RenderOutput(items, func() {
			if len(tableRows) == 0 {
				ui.PrintInfo("No personal access tokens found.")
				return
			}

			headers := []string{"ID", "Name", "Prefix", "Workspaces", "Created", "Expires", "Last Used", "Status"}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
		})
	},
}

var tokensRevokeCmd = &cobra.Command{
	Use:     "revoke <token_id>",
	Aliases: []string{"rm", "delete"},
	Short:   "Revoke a Personal Access Token immediately",
	Long: `Immediately invalidate an active Personal Access Token.

Any scripts, workers, or CI/CD pipelines using the revoked token will immediately lose access.

Examples:
  angler tokens revoke 99999999-8888-7777-6666-555555555555
  angler tokens revoke 99999999-8888-7777-6666-555555555555 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		if c.Token == "" {
			return ui.NewAuthError(fmt.Errorf("authentication required: run 'angler auth login' first"))
		}

		tokenID := strings.TrimSpace(args[0])
		if tokenID == "" {
			return ui.NewUsageError(fmt.Errorf("token ID is required"))
		}

		if !tokenForce && !assumeYes && !forceExec {
			if ui.IsNonInteractive() {
				return ui.NewUsageError(fmt.Errorf("confirmation required in non-interactive mode: pass --yes or --force"))
			}

			fmt.Fprintf(ui.OutWriter, "Revoke Personal Access Token %q? Any clients using it will lose access immediately. [y/N]: ", tokenID)
			reader := bufio.NewReader(os.Stdin)
			confirmation, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			conf := strings.ToLower(strings.TrimSpace(confirmation))
			if conf != "y" && conf != "yes" {
				ui.PrintInfo("Revocation cancelled.")
				return nil
			}
		}

		rev, err := c.RevokeToken(tokenID)
		if err != nil {
			return err
		}

		result := TokenRevokeResult{
			Status:  "revoked",
			TokenID: rev.RevokedID,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Successfully revoked Personal Access Token: %s", rev.RevokedID)
		})
	},
}

func init() {
	tokensCreateCmd.Flags().StringVarP(&tokenName, "name", "n", "", "name or description of the token")
	tokensCreateCmd.Flags().StringVar(&tokenComment, "comment", "", "alias for --name")
	tokensCreateCmd.Flags().IntVarP(&tokenLifetimeDays, "lifetime-days", "d", 30, "token lifespan in days (1 to 3650)")
	tokensCreateCmd.Flags().StringVarP(&tokenWorkspace, "workspace", "w", "", "workspace name, slug, or ID (defaults to active/default workspace)")
	tokensCreateCmd.Flags().StringVarP(&tokenRole, "role", "r", "read", "workspace access role ('read', 'write', 'admin')")
	tokensCreateCmd.Flags().StringSliceVarP(&tokenScopes, "scopes", "s", []string{}, "optional access scopes")

	tokensListCmd.Flags().BoolVarP(&tokenIncludeAll, "all", "a", false, "include revoked and expired tokens")

	tokensRevokeCmd.Flags().BoolVarP(&tokenForce, "force", "f", false, "bypass confirmation prompt")

	tokensCmd.AddCommand(tokensCreateCmd)
	tokensCmd.AddCommand(tokensListCmd)
	tokensCmd.AddCommand(tokensRevokeCmd)

	rootCmd.AddCommand(tokensCmd)
}
