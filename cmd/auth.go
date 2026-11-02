package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	loginEmail    string
	loginPassword string
)

type AuthLoginResult struct {
	Status   string `json:"status" yaml:"status"`
	Context  string `json:"context" yaml:"context"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Email    string `json:"email" yaml:"email"`
	Role     string `json:"role,omitempty" yaml:"role,omitempty"`
}

type AuthLogoutResult struct {
	Status  string `json:"status" yaml:"status"`
	Context string `json:"context" yaml:"context"`
}

type AuthStatusResult struct {
	Authenticated bool   `json:"authenticated" yaml:"authenticated"`
	Context       string `json:"context" yaml:"context"`
	Endpoint      string `json:"endpoint" yaml:"endpoint"`
	Email         string `json:"email,omitempty" yaml:"email,omitempty"`
	TokenPreview  string `json:"token_preview,omitempty" yaml:"token_preview,omitempty"`
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Lake of Tears",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Lake of Tears account",
	RunE: func(cmd *cobra.Command, args []string) error {
		email := strings.TrimSpace(loginEmail)
		if email == "" {
			email = strings.TrimSpace(os.Getenv("ANGLER_EMAIL"))
		}

		password := strings.TrimSpace(loginPassword)
		if password == "" {
			password = strings.TrimSpace(os.Getenv("ANGLER_PASSWORD"))
		}

		if email == "" || password == "" {
			if ui.IsNonInteractive() {
				return ui.NewUsageError(fmt.Errorf("credentials required in non-interactive mode: provide --email and --password (or set ANGLER_EMAIL and ANGLER_PASSWORD)"))
			}

			reader := bufio.NewReader(os.Stdin)
			if email == "" {
				fmt.Fprint(ui.OutWriter, "Email: ")
				emailInput, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				email = strings.TrimSpace(emailInput)
			}

			if password == "" {
				fmt.Fprint(ui.OutWriter, "Password: ")
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return err
				}
				fmt.Fprintln(ui.OutWriter)
				password = strings.TrimSpace(string(bytePassword))
			}
		}

		c := getClient()
		ui.PrintInfo("Authenticating with %s...", c.Endpoint)

		resp, err := c.Login(email, password)
		if err != nil {
			return ui.NewAuthError(err)
		}

		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		currentCtx := cfg.Contexts[ctxName]
		if currentCtx.Endpoint == "" {
			currentCtx.Endpoint = c.Endpoint
		}
		currentCtx.Email = email
		currentCtx.Token = resp.Token
		cfg.Contexts[ctxName] = currentCtx

		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		result := AuthLoginResult{
			Status:   "authenticated",
			Context:  ctxName,
			Endpoint: c.Endpoint,
			Email:    email,
			Role:     resp.User.Role,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Successfully logged in as %s (%s)!", email, resp.User.Role)
		})
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		ctx, exists := cfg.Contexts[ctxName]
		if !exists || ctx.Token == "" {
			result := AuthLogoutResult{
				Status:  "already_logged_out",
				Context: ctxName,
			}
			return ui.RenderOutput(result, func() {
				ui.PrintInfo("Not currently logged in for context %q.", ctxName)
			})
		}

		ctx.Token = ""
		ctx.Email = ""
		cfg.Contexts[ctxName] = ctx

		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		result := AuthLogoutResult{
			Status:  "logged_out",
			Context: ctxName,
		}
		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Logged out successfully from context %q.", ctxName)
		})
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getActiveContext()
		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		if ctx.Token == "" {
			result := AuthStatusResult{
				Authenticated: false,
				Context:       ctxName,
				Endpoint:      ctx.Endpoint,
			}
			return ui.RenderOutput(result, func() {
				ui.PrintWarning("Context %q: Not logged in (endpoint: %s)", ctxName, ctx.Endpoint)
				fmt.Fprintln(ui.OutWriter, "Run 'angler auth login' to authenticate.")
			})
		}

		preview := fmt.Sprintf("%s...%s", ctx.Token[:min(len(ctx.Token), 8)], ctx.Token[max(0, len(ctx.Token)-4):])
		result := AuthStatusResult{
			Authenticated: true,
			Context:       ctxName,
			Endpoint:      ctx.Endpoint,
			Email:         ctx.Email,
			TokenPreview:  preview,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Context %q: Authenticated", ctxName)
			fmt.Fprintf(ui.OutWriter, "Endpoint: %s\n", ctx.Endpoint)
			if ctx.Email != "" {
				fmt.Fprintf(ui.OutWriter, "User:     %s\n", ctx.Email)
			}
			fmt.Fprintf(ui.OutWriter, "Token:    %s\n", preview)
		})
	},
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	loginCmd.Flags().StringVar(&loginEmail, "email", "", "account email address (or ANGLER_EMAIL)")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "account password (or ANGLER_PASSWORD)")
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
