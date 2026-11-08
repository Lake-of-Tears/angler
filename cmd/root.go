package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile           string
	ctxOverride       string
	endpoint          string
	token             string
	workspaceFlag     string
	workspaceIDFlag   string
	jsonOutput        bool
	outputFlag        string
	nonInteractive    bool
	assumeYes         bool
	forceExec         bool
	runtimeConfigured bool

	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "angler",
	Short: "The official CLI for Lake of Tears data lakehouse",
	Long: `🎣 angler: Cast lines into your Lake of Tears data lakehouse.
Query DuckDB over S3, manage cluster deployments, browse data catalogs, 
and run AI semantic searches directly from your terminal.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := configureRuntimeMode(cmd); err != nil {
			return err
		}

		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		return nil
	},
}

func configureRuntimeMode(cmd *cobra.Command) error {
	runtimeConfigured = true
	ui.SetNonInteractive(nonInteractive || assumeYes || forceExec)

	resolved := ""
	allowTabular := cmd != nil && (cmd.Name() == "sql" || cmd.Name() == "exec" || cmd.Name() == "query" || cmd.Name() == "cast")

	if jsonOutput {
		resolved = ui.FormatJSON
	} else if outputFlag != "" {
		resolved = strings.ToLower(strings.TrimSpace(outputFlag))
	} else if allowTabular && cmd.Flags().Lookup("format") != nil && cmd.Flags().Changed("format") {
		resolved = strings.ToLower(strings.TrimSpace(cmd.Flags().Lookup("format").Value.String()))
	} else if !ui.IsStdoutTTY() {
		// Automatically switch to clean JSON when run by an AI agent, pipe, or script
		resolved = ui.FormatJSON
	} else {
		resolved = ui.FormatTable
	}

	if err := ui.ValidateFormat(resolved, allowTabular); err != nil {
		// When format is invalid, default error emission to JSON if --json or non-TTY
		if jsonOutput || !ui.IsStdoutTTY() {
			ui.SetOutputFormat(ui.FormatJSON)
		} else {
			ui.SetOutputFormat(ui.FormatTable)
		}
		return err
	}

	ui.SetOutputFormat(resolved)
	return nil
}

// ExecuteRoot runs the root command and returns the deterministic exit code.
func ExecuteRoot() int {
	runtimeConfigured = false
	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		if !runtimeConfigured {
			_ = configureRuntimeMode(executedCmd)
		}
		return ui.EmitError(err)
	}
	return ui.ExitOK
}

func Execute() {
	if code := ExecuteRoot(); code != ui.ExitOK {
		os.Exit(code)
	}
}

func init() {
	config.SetupViper()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.angler/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&ctxOverride, "context", "c", "", "context profile to use")
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "Lake of Tears API endpoint")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "API authentication token")
	rootCmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Lake of Tears workspace slug or name")
	rootCmd.PersistentFlags().StringVar(&workspaceIDFlag, "workspace-id", "", "explicit Lake of Tears workspace ID")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results in JSON format")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "output format: table, json, yaml, toon")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false, "automatic yes to prompts; run non-interactively")
	rootCmd.PersistentFlags().BoolVar(&forceExec, "force", false, "bypass confirmation prompts")
}

func getActiveContext() config.Context {
	active := cfg.GetActiveContext(ctxOverride)
	if endpoint != "" {
		active.Endpoint = endpoint
	}
	if token != "" {
		active.Token = token
	}
	if workspaceFlag != "" {
		active.Workspace = workspaceFlag
	}
	if workspaceIDFlag != "" {
		active.WorkspaceID = workspaceIDFlag
	}
	return active
}

func getClient() *client.Client {
	ctx := getActiveContext()
	return client.NewClient(ctx)
}
