package cmd

import (
	"encoding/json"
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
		if len(args) == 0 {
			return fmt.Errorf("please provide a question or prompt for sonar")
		}

		prompt := strings.Join(args, " ")
		c := getClient()

		ui.PrintInfo("Sending sonar pulse: %q", prompt)

		resp, err := c.AIQuery(prompt)
		if err != nil {
			ui.PrintError("Sonar failed: %v", err)
			return err
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println()
		ui.PrintSuccess("Echo received from data lake:")
		fmt.Println(ui.CardStyle.Render(resp.Answer))

		if resp.SQLQuery != "" {
			fmt.Println("\nGenerated SQL:")
			fmt.Println(ui.MutedStyle.Render(resp.SQLQuery))
		}

		if len(resp.Sources) > 0 {
			fmt.Println("\nGrounded Sources:")
			for _, src := range resp.Sources {
				fmt.Printf("  • %s\n", src)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sonarCmd)
}
