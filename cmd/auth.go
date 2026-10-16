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

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Lake of Tears",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Lake of Tears account",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Email: ")
		emailInput, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		email := strings.TrimSpace(emailInput)

		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		password := strings.TrimSpace(string(bytePassword))

		c := getClient()
		ui.PrintInfo("Authenticating with %s...", c.Endpoint)

		resp, err := c.Login(email, password)
		if err != nil {
			ui.PrintError("Login failed: %v", err)
			return err
		}

		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		currentCtx := cfg.Contexts[ctxName]
		currentCtx.Email = email
		currentCtx.Token = resp.Token
		cfg.Contexts[ctxName] = currentCtx

		if err := config.SaveConfig(cfg); err != nil {
			ui.PrintError("Failed to save credentials: %v", err)
			return err
		}

		ui.PrintSuccess("Successfully logged in as %s (%s)!", email, resp.User.Role)
		return nil
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
			ui.PrintInfo("Not currently logged in for context %q.", ctxName)
			return nil
		}

		ctx.Token = ""
		ctx.Email = ""
		cfg.Contexts[ctxName] = ctx

		if err := config.SaveConfig(cfg); err != nil {
			return err
		}

		ui.PrintSuccess("Logged out successfully from context %q.", ctxName)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := getActiveContext()
		ctxName := cfg.CurrentContext
		if ctxOverride != "" {
			ctxName = ctxOverride
		}

		if ctx.Token == "" {
			ui.PrintWarning("Context %q: Not logged in (endpoint: %s)", ctxName, ctx.Endpoint)
			fmt.Println("Run 'angler auth login' to authenticate.")
			return
		}

		ui.PrintSuccess("Context %q: Authenticated", ctxName)
		fmt.Printf("Endpoint: %s\n", ctx.Endpoint)
		if ctx.Email != "" {
			fmt.Printf("User:     %s\n", ctx.Email)
		}
		fmt.Printf("Token:    %s...%s\n", ctx.Token[:min(len(ctx.Token), 8)], ctx.Token[max(0, len(ctx.Token)-4):])
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
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
