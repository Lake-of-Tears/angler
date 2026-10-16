package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "local"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the angler version",
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			data, _ := json.MarshalIndent(map[string]string{
				"version":  Version,
				"commit":   Commit,
				"date":     Date,
				"built_by": BuiltBy,
			}, "", "  ")
			fmt.Println(string(data))
			return
		}

		ui.PrintBanner()
		fmt.Printf("angler version %s\n", ui.TitleStyle.Render(Version))
		fmt.Printf("git commit:     %s\n", Commit)
		fmt.Printf("build date:     %s\n", Date)
		fmt.Printf("built by:       %s\n", BuiltBy)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
