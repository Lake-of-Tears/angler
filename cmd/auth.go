package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	loginEmail      string
	loginPassword   string
	loginToken      string
	loginTokenStdin bool
)

type AuthLoginResult struct {
	Status     string `json:"status" yaml:"status"`
	Context    string `json:"context" yaml:"context"`
	Endpoint   string `json:"endpoint" yaml:"endpoint"`
	Email      string `json:"email,omitempty" yaml:"email,omitempty"`
	Role       string `json:"role,omitempty" yaml:"role,omitempty"`
	AuthMethod string `json:"auth_method" yaml:"auth_method"`
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
	TokenType     string `json:"token_type,omitempty" yaml:"token_type,omitempty"`
	TokenPreview  string `json:"token_preview,omitempty" yaml:"token_preview,omitempty"`
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Lake of Tears",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Lake of Tears account via Personal Access Token or credentials",
	Long: `Log in to Lake of Tears using a Personal Access Token (PAT) or email and password.

Recommended:
  Use a Personal Access Token created in the Lake of Tears Web UI (Account Settings -> Personal Access Tokens).
  This avoids storing raw passwords and supports granular workspace-level access.

Examples:
  # Authenticate with a PAT (interactive prompt or flag)
  angler auth login --token lot_pat_abcdef0123456789...

  # Pipe token via stdin (ideal for CI/CD)
  echo "$LOT_TOKEN" | angler auth login --token-stdin

  # Legacy interactive login
  angler auth login`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		activeCtx := getActiveContext()
		endpointURL := activeCtx.Endpoint
		if endpoint != "" {
			endpointURL = endpoint
		}

		var tokenToUse string
		if loginTokenStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read token from stdin: %w", err)
			}
			tokenToUse = strings.TrimSpace(string(data))
			if tokenToUse == "" {
				return ui.NewUsageError(fmt.Errorf("empty token provided via stdin"))
			}
		} else if strings.TrimSpace(loginToken) != "" {
			tokenToUse = strings.TrimSpace(loginToken)
		}

		email := strings.TrimSpace(loginEmail)
		if email == "" {
			email = strings.TrimSpace(os.Getenv("ANGLER_EMAIL"))
		}

		password := strings.TrimSpace(loginPassword)
		if password == "" {
			password = strings.TrimSpace(os.Getenv("ANGLER_PASSWORD"))
		}

		// Check ANGLER_TOKEN if nothing provided
		if tokenToUse == "" && email == "" && password == "" {
			if envTok := strings.TrimSpace(os.Getenv("ANGLER_TOKEN")); envTok != "" {
				tokenToUse = envTok
			}
		}

		// If still nothing provided, prompt or fail if non-interactive
		if tokenToUse == "" && (email == "" || password == "") {
			if ui.IsNonInteractive() {
				return ui.NewUsageError(fmt.Errorf("credentials required in non-interactive mode: provide --token, --token-stdin, or --email and --password (or set ANGLER_TOKEN, ANGLER_EMAIL, and ANGLER_PASSWORD)"))
			}

			reader := bufio.NewReader(os.Stdin)
			fmt.Fprintln(ui.OutWriter, "Select an authentication method:")
			fmt.Fprintln(ui.OutWriter, "  1) Personal Access Token (PAT) [Recommended]")
			fmt.Fprintln(ui.OutWriter, "  2) Email and Password")
			fmt.Fprint(ui.OutWriter, "Choice [1]: ")
			choice, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			choice = strings.TrimSpace(choice)

			if choice == "2" {
				// Email and password flow
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
			} else {
				// PAT flow
				fmt.Fprint(ui.OutWriter, "Personal Access Token (lot_pat_...): ")
				var patInput string
				if term.IsTerminal(int(syscall.Stdin)) {
					byteToken, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return err
					}
					fmt.Fprintln(ui.OutWriter)
					patInput = string(byteToken)
				} else {
					line, err := reader.ReadString('\n')
					if err != nil {
						return err
					}
					patInput = line
				}
				tokenToUse = strings.TrimSpace(patInput)
				if tokenToUse == "" {
					return ui.NewUsageError(fmt.Errorf("personal access token cannot be empty"))
				}
			}
		}

		// Execute Token authentication
		if tokenToUse != "" {
			ui.PrintInfo("Validating Personal Access Token with %s...", endpointURL)
			testClient := client.NewClient(config.Context{
				Endpoint: endpointURL,
				Token:    tokenToUse,
				Timeout:  activeCtx.Timeout,
			})

			user, err := testClient.CurrentUser()
			if err != nil {
				return ui.NewAuthError(err)
			}

			currentCtx := cfg.Contexts[ctxName]
			currentCtx.Endpoint = endpointURL
			currentCtx.Token = tokenToUse
			currentCtx.Email = user.Email
			cfg.Contexts[ctxName] = currentCtx

			if err := config.SaveConfig(cfg); err != nil {
				return err
			}

			result := AuthLoginResult{
				Status:     "authenticated",
				Context:    ctxName,
				Endpoint:   endpointURL,
				Email:      user.Email,
				Role:       user.Role,
				AuthMethod: "personal_access_token",
			}

			return ui.RenderOutput(result, func() {
				ui.PrintSuccess("Successfully authenticated via Personal Access Token as %s (%s)!", user.Email, user.Role)
			})
		}

		// Execute Email/Password authentication
		c := getClient()
		ui.PrintInfo("Authenticating with %s...", c.Endpoint)

		resp, err := c.Login(email, password)
		if err != nil {
			return ui.NewAuthError(err)
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
			Status:     "authenticated",
			Context:    ctxName,
			Endpoint:   c.Endpoint,
			Email:      email,
			Role:       resp.User.Role,
			AuthMethod: "password",
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

		tokenType := "session_token"
		tokenTypeDisplay := "Session Token (JWT)"
		if strings.HasPrefix(ctx.Token, "lot_pat_") {
			tokenType = "personal_access_token"
			tokenTypeDisplay = "Personal Access Token (PAT)"
		}

		preview := fmt.Sprintf("%s...%s", ctx.Token[:min(len(ctx.Token), 8)], ctx.Token[max(0, len(ctx.Token)-4):])
		result := AuthStatusResult{
			Authenticated: true,
			Context:       ctxName,
			Endpoint:      ctx.Endpoint,
			Email:         ctx.Email,
			TokenType:     tokenType,
			TokenPreview:  preview,
		}

		return ui.RenderOutput(result, func() {
			ui.PrintSuccess("Context %q: Authenticated", ctxName)
			fmt.Fprintf(ui.OutWriter, "Endpoint:   %s\n", ctx.Endpoint)
			if ctx.Email != "" {
				fmt.Fprintf(ui.OutWriter, "User:       %s\n", ctx.Email)
			}
			fmt.Fprintf(ui.OutWriter, "Token Type: %s\n", tokenTypeDisplay)
			fmt.Fprintf(ui.OutWriter, "Token:      %s\n", preview)
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
	loginCmd.Flags().StringVarP(&loginToken, "token", "t", "", "authenticate using Personal Access Token (or ANGLER_TOKEN)")
	loginCmd.Flags().BoolVar(&loginTokenStdin, "token-stdin", false, "read Personal Access Token from standard input")
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
