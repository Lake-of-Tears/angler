package cmd

import (
	"fmt"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	Version = "0.2.4"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "local"
)

type VersionInfo struct {
	Version string `json:"version" yaml:"version"`
	Commit  string `json:"commit" yaml:"commit"`
	Date    string `json:"date" yaml:"date"`
	BuiltBy string `json:"built_by" yaml:"built_by"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the angler version",
	RunE: func(cmd *cobra.Command, args []string) error {
		info := VersionInfo{
			Version: Version,
			Commit:  Commit,
			Date:    Date,
			BuiltBy: BuiltBy,
		}

		return ui.RenderOutput(info, func() {
			ui.PrintBanner()
			fmt.Fprintf(ui.OutWriter, "angler version %s\n", ui.StyleText(ui.TitleStyle, Version))
			fmt.Fprintf(ui.OutWriter, "git commit:     %s\n", Commit)
			fmt.Fprintf(ui.OutWriter, "build date:     %s\n", Date)
			fmt.Fprintf(ui.OutWriter, "built by:       %s\n", BuiltBy)
		})
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
