package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	webhooksSecretRaw       bool
	webhooksForce           bool
	webhooksPayload         string
	webhooksPayloadFile     string
	webhooksTriggerPipeline string
	webhooksSecretToken     string
	webhooksUseHMAC         bool
)

var webhooksCmd = &cobra.Command{
	Use:     "webhooks",
	Aliases: []string{"webhook"},
	Short:   "Workspace Webhook Ingestion, HMAC Signing, Secret Rotation & Downstream Pipeline Triggering",
	Long: `Manage workspace webhook secret tokens and stream real-time JSON event payloads
into MinIO Parquet raw storage with optional atomic downstream pipeline triggering.

Examples:
  angler webhooks secret get
  angler webhooks secret rotate --force
  angler webhooks send --payload '{"event":"order_placed","amount":99.5}'
  angler webhooks send -f events.json --trigger-pipeline bronze_to_silver_etl --hmac`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWebhooksSecretGet(cmd, args)
	},
}

var webhooksSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Retrieve or rotate the active workspace's webhook secret token",
	RunE:  runWebhooksSecretGet,
}

var webhooksSecretGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the active workspace's webhook secret token and raw ingestion URL",
	RunE:  runWebhooksSecretGet,
}

func runWebhooksSecretGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	res, err := c.GetWebhookSecret(c.Workspace)
	if err != nil {
		return err
	}

	if webhooksSecretRaw {
		fmt.Fprintln(ui.OutWriter, res.WebhookSecret)
		return nil
	}

	fullURL := strings.TrimRight(c.Endpoint, "/") + res.WebhookURL
	return ui.RenderOutput(res, func() {
		ui.PrintSuccess("Workspace Webhook Configuration (%s)", res.WorkspaceSlug)
		fmt.Fprintf(ui.OutWriter, "  Workspace ID:   %s\n", res.WorkspaceID)
		fmt.Fprintf(ui.OutWriter, "  Workspace Slug: %s\n", res.WorkspaceSlug)
		fmt.Fprintf(ui.OutWriter, "  Webhook Secret: %s\n", res.WebhookSecret)
		fmt.Fprintf(ui.OutWriter, "  Ingestion URL:  %s\n", fullURL)
	})
}

var webhooksSecretRotateCmd = &cobra.Command{
	Use:     "rotate",
	Aliases: []string{"regenerate"},
	Short:   "Regenerate the workspace's webhook secret token",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		ws := c.Workspace
		if ws == "" {
			ws = "default"
		}
		if !webhooksForce && !confirmAction(fmt.Sprintf("Rotate webhook secret for workspace '%s'? Existing integrations using the old secret will fail.", ws)) {
			ui.PrintInfo("Rotation cancelled.")
			return nil
		}

		res, err := c.RegenerateWebhookSecret(ws)
		if err != nil {
			return err
		}

		fullURL := strings.TrimRight(c.Endpoint, "/") + res.WebhookURL
		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Rotated webhook secret for workspace '%s'", res.WorkspaceSlug)
			fmt.Fprintf(ui.OutWriter, "  New Secret:    %s\n", res.WebhookSecret)
			fmt.Fprintf(ui.OutWriter, "  Ingestion URL: %s\n", fullURL)
		})
	},
}

var webhooksSendCmd = &cobra.Command{
	Use:     "send",
	Aliases: []string{"ingest"},
	Short:   "Send an event payload to workspace webhook ingestion (MinIO Parquet buffer + optional pipeline trigger)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var rawPayload []byte
		if webhooksPayloadFile != "" {
			b, err := os.ReadFile(webhooksPayloadFile)
			if err != nil {
				return fmt.Errorf("failed to read payload file %s: %w", webhooksPayloadFile, err)
			}
			rawPayload = b
		} else if webhooksPayload != "" {
			rawPayload = []byte(webhooksPayload)
		} else if len(args) > 0 {
			rawPayload = []byte(args[0])
		} else {
			return ui.NewUsageError(fmt.Errorf("please provide --payload / -d or --file / -f"))
		}

		c := getClient()
		res, err := c.IngestWebhookEvent(c.Workspace, rawPayload, webhooksTriggerPipeline, webhooksSecretToken, webhooksUseHMAC)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Webhook event buffered to MinIO Parquet (Event ID: %s)", res.EventID)
			fmt.Fprintf(ui.OutWriter, "  Status:      %s\n", res.Status)
			fmt.Fprintf(ui.OutWriter, "  Event ID:    %s\n", res.EventID)
			fmt.Fprintf(ui.OutWriter, "  Workspace:   %s\n", res.WorkspaceSlug)
			fmt.Fprintf(ui.OutWriter, "  Received At: %s\n", res.ReceivedAt)
			if res.TriggeredPipeline != nil && *res.TriggeredPipeline != "" {
				fmt.Fprintf(ui.OutWriter, "  Triggered:   %s\n", *res.TriggeredPipeline)
			}
			if res.DagRunID != nil && *res.DagRunID != "" {
				fmt.Fprintf(ui.OutWriter, "  DAG Run ID:  %s\n", *res.DagRunID)
			}
		})
	},
}

func init() {
	webhooksSecretGetCmd.Flags().BoolVar(&webhooksSecretRaw, "raw", false, "Print raw webhook secret string to stdout")
	webhooksSecretRotateCmd.Flags().BoolVarP(&webhooksForce, "force", "f", false, "Rotate secret without confirmation prompt")

	webhooksSendCmd.Flags().StringVarP(&webhooksPayload, "payload", "d", "", "JSON event payload string")
	webhooksSendCmd.Flags().StringVarP(&webhooksPayloadFile, "file", "f", "", "Path to JSON event payload file")
	webhooksSendCmd.Flags().StringVar(&webhooksTriggerPipeline, "trigger-pipeline", "", "Downstream pipeline DAG ID to trigger immediately after buffering")
	webhooksSendCmd.Flags().StringVar(&webhooksSecretToken, "secret", "", "Explicit workspace webhook secret (auto-resolved if omitted)")
	webhooksSendCmd.Flags().BoolVar(&webhooksUseHMAC, "hmac", false, "Sign payload with HMAC-SHA256 (X-Hub-Signature-256)")

	webhooksSecretCmd.AddCommand(webhooksSecretGetCmd)
	webhooksSecretCmd.AddCommand(webhooksSecretRotateCmd)

	webhooksCmd.AddCommand(webhooksSecretCmd)
	webhooksCmd.AddCommand(webhooksSendCmd)

	rootCmd.AddCommand(webhooksCmd)
}
