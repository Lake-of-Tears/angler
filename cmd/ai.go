package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	aiStream   bool
	aiNoStream bool
	aiMode     string
	aiModel    string
	aiKey      string
)

var aiCmd = &cobra.Command{
	Use:     "ai",
	Aliases: []string{"genie"},
	Short:   "Lake of Tears Genie AI, Semantic Gateway, and Vector Indexes (alias: genie)",
	Long: `Interact with Lake of Tears' built-in Genie semantic engine, conversational AI,
vector indexes, and LLM gateway.

Enables human engineers and autonomous LLM agents to converse with Genie AI grounded in
lakehouse catalogs and schemas, stream execution steps, inspect active grounding contexts,
and manage vector embeddings.

Aliases:
  angler ai
  angler genie

Examples:
  angler ai chat "Which tables have customer transaction data?"
  angler ai chat --no-stream "Count total orders in 2026"
  angler ai context
  angler ai vector-index list
  angler ai vector-index search "high value customers"
  angler ai test-key --model gemini-2.0-flash`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return runAIChat(cmd, args)
		}
		return cmd.Help()
	},
}

var aiChatCmd = &cobra.Command{
	Use:   "chat [prompt]",
	Short: "Chat with Lake of Tears Genie AI grounded in catalog tables and metadata",
	Long: `Executes a single question or initiates an interactive chat session with Genie AI.
Grounded in workspace catalogs, Parquet schemas, and SQL AI functions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAIChat(cmd, args)
	},
}

func runAIChat(cmd *cobra.Command, args []string) error {
	c := getClient()

	prompt := strings.Join(args, " ")
	prompt = strings.TrimSpace(prompt)

	shouldStream := !aiNoStream && (aiStream || ui.IsStdoutTTY())
	if jsonOutput || outputFlag == "json" {
		shouldStream = false
	}

	// Interactive REPL loop when no prompt argument is provided and TTY is active
	if prompt == "" {
		if !ui.IsStdoutTTY() || nonInteractive || assumeYes {
			return ui.NewUsageError(fmt.Errorf("please provide a question or prompt for ai chat"))
		}

		ui.PrintSuccess("Lake of Tears Genie AI (Interactive Session)")
		ui.PrintInfo("Type your question and press Enter. Type 'exit' or 'quit' to end.\n")

		var history []client.GenieChatMessage
		scanner := bufio.NewScanner(os.Stdin)

		for {
			fmt.Fprint(ui.OutWriter, ui.StyleText(ui.TitleStyle, "genie> "))
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				ui.PrintInfo("Goodbye.")
				return nil
			}

			req := client.GenieChatRequest{
				Question: input,
				Mode:     aiMode,
				Model:    aiModel,
				History:  history,
			}

			if shouldStream {
				var finalResp client.GenieChatResponse
				err := c.GenieChatStream(req, func(event string, raw []byte) error {
					if event == "step" {
						var stepEvt map[string]any
						_ = json.Unmarshal(raw, &stepEvt)
						msg, _ := stepEvt["message"].(string)
						if msg != "" {
							fmt.Fprintf(ui.OutWriter, "%s %s\n", ui.StyleText(ui.MutedStyle, "•"), ui.StyleText(ui.MutedStyle, msg))
						}
					} else if event == "result" {
						_ = json.Unmarshal(raw, &finalResp)
					}
					return nil
				})
				if err != nil {
					ui.PrintError("Genie error: %v", err)
					continue
				}

				if finalResp.Answer != "" {
					fmt.Fprintln(ui.OutWriter, "\n"+finalResp.Answer)
				}
				if finalResp.SQL != "" {
					fmt.Fprintf(ui.OutWriter, "\n%s\n%s\n", ui.StyleText(ui.TitleStyle, "Generated SQL:"), ui.StyleText(ui.MutedStyle, finalResp.SQL))
				}
				fmt.Fprintln(ui.OutWriter)

				history = append(history, client.GenieChatMessage{Role: "user", Content: input})
				history = append(history, client.GenieChatMessage{Role: "assistant", Content: finalResp.Answer})
			} else {
				resp, err := c.GenieChat(req)
				if err != nil {
					ui.PrintError("Genie error: %v", err)
					continue
				}
				if resp.Answer != "" {
					fmt.Fprintln(ui.OutWriter, "\n"+resp.Answer)
				}
				if resp.SQL != "" {
					fmt.Fprintf(ui.OutWriter, "\n%s\n%s\n", ui.StyleText(ui.TitleStyle, "Generated SQL:"), ui.StyleText(ui.MutedStyle, resp.SQL))
				}
				fmt.Fprintln(ui.OutWriter)

				history = append(history, client.GenieChatMessage{Role: "user", Content: input})
				history = append(history, client.GenieChatMessage{Role: "assistant", Content: resp.Answer})
			}
		}
		return scanner.Err()
	}

	// Single-turn question execution
	req := client.GenieChatRequest{
		Question: prompt,
		Mode:     aiMode,
		Model:    aiModel,
	}

	if shouldStream {
		var finalResp client.GenieChatResponse
		err := c.GenieChatStream(req, func(event string, raw []byte) error {
			if event == "step" {
				var stepEvt map[string]any
				_ = json.Unmarshal(raw, &stepEvt)
				msg, _ := stepEvt["message"].(string)
				if msg != "" {
					fmt.Fprintf(ui.OutWriter, "%s %s\n", ui.StyleText(ui.MutedStyle, "•"), ui.StyleText(ui.MutedStyle, msg))
				}
			} else if event == "result" {
				_ = json.Unmarshal(raw, &finalResp)
			}
			return nil
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(finalResp, func() {
			if finalResp.Answer != "" {
				fmt.Fprintln(ui.OutWriter, "\n"+finalResp.Answer)
			}
			if finalResp.SQL != "" {
				fmt.Fprintf(ui.OutWriter, "\n%s\n%s\n", ui.StyleText(ui.TitleStyle, "Generated SQL:"), ui.StyleText(ui.MutedStyle, finalResp.SQL))
			}
			if len(finalResp.Data) > 0 {
				fmt.Fprintf(ui.OutWriter, "\n%s (%d rows)\n", ui.StyleText(ui.SuccessStyle, "Result Data:"), len(finalResp.Data))
				if len(finalResp.Columns) > 0 {
					headers := finalResp.Columns
					rows := make([][]string, len(finalResp.Data))
					for i, row := range finalResp.Data {
						rows[i] = make([]string, len(headers))
						for j, col := range headers {
							rows[i][j] = fmt.Sprintf("%v", row[col])
						}
					}
					fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
				}
			}
		})
	}

	resp, err := c.GenieChat(req)
	if err != nil {
		return err
	}

	return ui.RenderOutput(resp, func() {
		if resp.Answer != "" {
			fmt.Fprintln(ui.OutWriter, resp.Answer)
		}
		if resp.SQL != "" {
			fmt.Fprintf(ui.OutWriter, "\n%s\n%s\n", ui.StyleText(ui.TitleStyle, "Generated SQL:"), ui.StyleText(ui.MutedStyle, resp.SQL))
		}
		if len(resp.Data) > 0 {
			fmt.Fprintf(ui.OutWriter, "\n%s (%d rows)\n", ui.StyleText(ui.SuccessStyle, "Result Data:"), len(resp.Data))
			if len(resp.Columns) > 0 {
				headers := resp.Columns
				rows := make([][]string, len(resp.Data))
				for i, row := range resp.Data {
					rows[i] = make([]string, len(headers))
					for j, col := range headers {
						rows[i][j] = fmt.Sprintf("%v", row[col])
					}
				}
				fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			}
		}
	})
}

var aiContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Inspect active system data context, grounding prompt, and registered UDFs",
	Long:  `Fetches schemas, datasets, and SQL AI function definitions exposed to the AI engine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		resp, err := c.GetAIContext()
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("AI System Context: %s (Slug: %s, Role: %s)", resp.WorkspaceName, resp.WorkspaceSlug, resp.Role)

			fmt.Fprintf(ui.OutWriter, "\n%s (%d registered):\n", ui.StyleText(ui.TitleStyle, "Catalogs"), len(resp.Catalogs))
			for _, cat := range resp.Catalogs {
				name, _ := cat["name"].(string)
				schemas, _ := cat["schemas"].([]any)
				fmt.Fprintf(ui.OutWriter, "  • %s (%d schemas)\n", name, len(schemas))
			}

			if len(resp.DefaultDatasets) > 0 {
				fmt.Fprintf(ui.OutWriter, "\n%s (%d available):\n", ui.StyleText(ui.TitleStyle, "Default Lakehouse Datasets"), len(resp.DefaultDatasets))
				for _, ds := range resp.DefaultDatasets {
					name, _ := ds["name"].(string)
					table, _ := ds["table"].(string)
					fmt.Fprintf(ui.OutWriter, "  • %s (table: %s)\n", name, table)
				}
			}

			if len(resp.SQLAIFunctions) > 0 {
				fmt.Fprintf(ui.OutWriter, "\n%s (%d available):\n", ui.StyleText(ui.TitleStyle, "Registered SQL AI Functions"), len(resp.SQLAIFunctions))
				for _, fn := range resp.SQLAIFunctions {
					name, _ := fn["name"].(string)
					desc, _ := fn["description"].(string)
					fmt.Fprintf(ui.OutWriter, "  • %s: %s\n", name, desc)
				}
			}
		})
	},
}

// ── Vector Index Subcommands ────────────────────────────────────────────────

var aiVectorIndexCmd = &cobra.Command{
	Use:     "vector-index",
	Aliases: []string{"vector", "vectors"},
	Short:   "Manage and search table vector indexes and semantic embeddings",
}

var aiVectorIndexListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List active table vector embeddings and indexes across workspace catalogs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		summaries, err := c.ListVectorIndexes()
		if err != nil {
			return err
		}

		return ui.RenderOutput(summaries, func() {
			if len(summaries) == 0 {
				ui.PrintInfo("No tables with vector indexes found in active catalogs.")
				return
			}

			headers := []string{"TABLE", "SOURCE COLUMN", "VECTORS", "STATUS", "S3 VECTOR PATH"}
			rows := make([][]string, len(summaries))
			for i, v := range summaries {
				rows[i] = []string{
					v.TableName,
					v.SourceColumn,
					fmt.Sprintf("%d", v.TotalVectors),
					v.Status,
					v.S3VectorPath,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total vector indexes: %d", len(summaries))
		})
	},
}

var aiVectorIndexSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Execute semantic vector search against registered documentation and tables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		c := getClient()

		req := client.GenieChatRequest{
			Question: query,
			Mode:     "rag",
			Model:    aiModel,
		}

		resp, err := c.GenieChat(req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			ui.PrintSuccess("Vector search result for: %q", query)
			if resp.Answer != "" {
				fmt.Fprintln(ui.OutWriter, "\n"+resp.Answer)
			}
			if len(resp.GroundedDocs) > 0 {
				fmt.Fprintln(ui.OutWriter, "\nGrounded Documents:")
				for _, doc := range resp.GroundedDocs {
					fmt.Fprintf(ui.OutWriter, "  • %s\n", doc)
				}
			}
		})
	},
}

var aiVectorIndexSyncCmd = &cobra.Command{
	Use:   "sync <catalog_id> <schema_id> <table_id>",
	Short: "Trigger incremental or full vector index synchronization for a table",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]
		tableID := args[2]

		c := getClient()
		if err := c.SyncTableVectorIndex(catalogID, schemaID, tableID); err != nil {
			return err
		}

		ui.PrintSuccess("Vector index synchronization initiated for table '%s'", tableID)
		return nil
	},
}

var aiTestKeyCmd = &cobra.Command{
	Use:   "test-key",
	Short: "Validate configured LLM provider credentials (e.g. Gemini, OpenAI)",
	Long:  `Validates configured or candidate LLM API keys against the backend provider verification endpoint.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()

		resp, err := c.TestGeminiKey(aiKey, aiModel)
		if err != nil {
			return err
		}

		return ui.RenderOutput(resp, func() {
			if resp.OK {
				ui.PrintSuccess("LLM Provider verification succeeded (Model: %s)", resp.Model)
				fmt.Fprintf(ui.OutWriter, "Message: %s\n", resp.Message)
			} else {
				ui.PrintError("LLM Provider verification failed (Model: %s)", resp.Model)
				fmt.Fprintf(ui.OutWriter, "Message: %s\n", resp.Message)
			}
		})
	},
}

func init() {
	// chat flags
	aiChatCmd.Flags().BoolVar(&aiStream, "stream", false, "Stream response steps in real time")
	aiChatCmd.Flags().BoolVar(&aiNoStream, "no-stream", false, "Disable streaming and return complete response")
	aiChatCmd.Flags().StringVarP(&aiMode, "mode", "m", "auto", "Query mode: auto, sql, rag, clarify")
	aiChatCmd.Flags().StringVar(&aiModel, "model", "", "LLM model override (e.g. gemini-2.0-flash)")

	// test-key flags
	aiTestKeyCmd.Flags().StringVar(&aiKey, "key", "", "Candidate API key to test (uses configured key if omitted)")
	aiTestKeyCmd.Flags().StringVar(&aiModel, "model", "gemini-2.0-flash", "LLM model to verify")

	// vector-index subcommands
	aiVectorIndexCmd.AddCommand(aiVectorIndexListCmd)
	aiVectorIndexCmd.AddCommand(aiVectorIndexSearchCmd)
	aiVectorIndexCmd.AddCommand(aiVectorIndexSyncCmd)

	// ai subcommands
	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiContextCmd)
	aiCmd.AddCommand(aiVectorIndexCmd)
	aiCmd.AddCommand(aiTestKeyCmd)

	rootCmd.AddCommand(aiCmd)
}
