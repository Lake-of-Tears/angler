package cmd

import (
	"fmt"
	"os"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/config"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	ctxOverride string
	endpoint    string
	token       string
	jsonOutput  bool

	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "angler",
	Short: "The official CLI for Lake of Tears data lakehouse",
	Long: `🎣 angler: Cast lines into your Lake of Tears data lakehouse.
Query DuckDB over S3, manage cluster deployments, browse data catalogs, 
and run AI semantic searches directly from your terminal.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError("%v", err)
		os.Exit(1)
	}
}

func init() {
	config.SetupViper()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.angler/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&ctxOverride, "context", "c", "", "context profile to use")
	rootCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "", "Lake of Tears API endpoint")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "API authentication token")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results in JSON format")
}

func getActiveContext() config.Context {
	active := cfg.GetActiveContext(ctxOverride)
	if endpoint != "" {
		active.Endpoint = endpoint
	}
	if token != "" {
		active.Token = token
	}
	return active
}

func getClient() *client.Client {
	ctx := getActiveContext()
	return client.NewClient(ctx)
}
