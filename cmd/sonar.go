package cmd

import (
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var sonarCmd = &cobra.Command{
	Use:     "ask [QUESTION]",
	Aliases: []string{"sonar"},
	Short:   "Ask questions in natural language with AI search (alias: sonar)",
	Long: `Transmit a sonar pulse into your Lake of Tears data.
Uses vector embeddings and Gemini RAG to answer questions grounded in your data.

Examples:
  angler ask "Which region had the highest revenue drop last quarter?"
  angler sonar "Show anomalies detected in server telemetry yesterday"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(strings.Join(args, " ")) == "" {
			return ui.NewUsageError(fmt.Errorf("please provide a question or prompt for sonar"))
		}

		prompt := strings.Join(args, " ")
		c := getClient()

		ui.PrintInfo("Sending sonar pulse: %q", prompt)

		resp, err := c.AIQuery(prompt)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			fmt.Fprintln(ui.OutWriter)
			ui.PrintSuccess("Echo received from data lake:")
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.CardStyle, resp.Answer))

			if resp.SQLQuery != "" {
				fmt.Fprintln(ui.OutWriter, "\nGenerated SQL:")
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.MutedStyle, resp.SQLQuery))
			}

			if len(resp.Sources) > 0 {
				fmt.Fprintln(ui.OutWriter, "\nGrounded Sources:")
				for _, src := range resp.Sources {
					fmt.Fprintf(ui.OutWriter, "  • %s\n", src)
				}
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(sonarCmd)
}
