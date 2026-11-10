package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	adminSessionTimeout int
	adminMaxHours       int
	adminGeminiModel    string
	adminGeminiKey      string
	adminForce          bool

	// Vault admin flags
	adminVaultConfigJSON   string
	adminVaultConfigFile   string
	adminVaultURL          string
	adminVaultRegion       string
	adminVaultProjectID    string
	adminVaultAuthMethod   string
	adminVaultMountPoint   string
	adminVaultNamespace    string
	adminVaultToken        string
	adminVaultRoleID       string
	adminVaultSecretID     string
	adminVaultSAFile       string
	adminVaultClientID     string
	adminVaultClientSecret string
	adminVaultTenantID     string

	// AI provider admin flags
	adminAIConfigJSON     string
	adminAIConfigFile     string
	adminAIActivate       bool
	adminAIAPIKey         string
	adminAIChatModel      string
	adminAIEmbeddingModel string
	adminAIBaseURL        string
	adminAIProjectID      string
	adminAILocation       string
	adminAISAFile         string
	adminAIRegion         string
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Lake of Tears System Administration, Settings, and Session Auditing",
	Long: `Administrative command suite for cluster superadmins.
Inspect and update cluster-wide runtime settings, review active sessions,
and revoke compromised tokens or connections.

Requires cluster superadmin privileges.

Examples:
  angler admin settings get
  angler admin settings update --session-timeout 120 --gemini-model gemini-2.0-flash
  angler admin sessions list
  angler admin sessions revoke sess-123 --force`,
}

// ── Settings Subcommands ────────────────────────────────────────────────────

var adminSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Inspect and update system-wide configuration settings",
}

var adminSettingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve active cluster-wide configuration parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		settings, err := c.GetSystemSettings()
		if err != nil {
			return err
		}

		return ui.RenderOutput(settings, func() {
			geminiConfigured := "No"
			if settings.GeminiAPIKeyConfigured {
				geminiConfigured = "Yes"
			}
			model := "None"
			if settings.GeminiDefaultModel != nil && *settings.GeminiDefaultModel != "" {
				model = *settings.GeminiDefaultModel
			}

			ui.PrintSuccess("Cluster System Settings")
			fmt.Fprintf(ui.OutWriter, "Catalog Soft-Delete Days:       %d\n", settings.CatalogSoftDeleteDays)
			fmt.Fprintf(ui.OutWriter, "Workspace Inactive Grace Days:  %d\n", settings.WorkspaceInactiveGracePeriodDays)
			fmt.Fprintf(ui.OutWriter, "Session Inactivity Timeout:     %d minutes\n", settings.SessionInactivityTimeoutMinutes)
			fmt.Fprintf(ui.OutWriter, "Session Maximum Lifetime:       %d hours\n", settings.SessionMaxHours)
			fmt.Fprintf(ui.OutWriter, "Gemini API Key Configured:      %s\n", geminiConfigured)
			if settings.GeminiAPIKeyMasked != nil {
				fmt.Fprintf(ui.OutWriter, "Gemini Key Masked:              %s\n", *settings.GeminiAPIKeyMasked)
			}
			fmt.Fprintf(ui.OutWriter, "Default AI Model:               %s\n", model)
		})
	},
}

var adminSettingsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update cluster-wide configuration settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()

		var req client.UpdateSystemSettingsRequest
		if cmd.Flags().Changed("session-timeout") {
			req.SessionInactivityTimeoutMinutes = &adminSessionTimeout
		}
		if cmd.Flags().Changed("max-hours") {
			req.SessionMaxHours = &adminMaxHours
		}
		if cmd.Flags().Changed("gemini-model") {
			req.GeminiDefaultModel = &adminGeminiModel
		}
		if cmd.Flags().Changed("gemini-key") {
			req.GeminiAPIKey = &adminGeminiKey
		}

		settings, err := c.UpdateSystemSettings(req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(settings, func() {
			ui.PrintSuccess("System settings updated successfully.")
			fmt.Fprintf(ui.OutWriter, "Session Inactivity Timeout: %d minutes\n", settings.SessionInactivityTimeoutMinutes)
			fmt.Fprintf(ui.OutWriter, "Session Maximum Lifetime:   %d hours\n", settings.SessionMaxHours)
			if settings.GeminiDefaultModel != nil {
				fmt.Fprintf(ui.OutWriter, "Default AI Model:           %s\n", *settings.GeminiDefaultModel)
			}
		})
	},
}

// ── Sessions Subcommands ────────────────────────────────────────────────────

var adminSessionsCmd = &cobra.Command{
	Use:     "sessions",
	Aliases: []string{"session"},
	Short:   "Audit and manage active user and API sessions",
}

var adminSessionsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List active user sessions across the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		sessions, err := c.ListSessions()
		if err != nil {
			return err
		}

		return ui.RenderOutput(sessions, func() {
			if len(sessions) == 0 {
				ui.PrintInfo("No active sessions found.")
				return
			}

			headers := []string{"SESSION ID", "USER", "IP ADDRESS", "CURRENT", "EXPIRES AT"}
			rows := make([][]string, len(sessions))
			for i, s := range sessions {
				userStr := s.UserID
				if s.Email != nil && *s.Email != "" {
					userStr = *s.Email
				}
				ipStr := "N/A"
				if s.IPAddress != nil && *s.IPAddress != "" {
					ipStr = *s.IPAddress
				}
				currentStr := "No"
				if s.IsCurrent {
					currentStr = "Yes"
				}

				rows[i] = []string{
					s.ID,
					userStr,
					ipStr,
					currentStr,
					s.ExpiresAt.Format("2006-01-02 15:04:05"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total active sessions: %d", len(sessions))
		})
	},
}

var adminSessionsRevokeCmd = &cobra.Command{
	Use:   "revoke <session_id>",
	Short: "Revoke an active user or service session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		c := getClient()

		if !adminForce && !confirmAction(fmt.Sprintf("Revoke session '%s'?", sessionID)) {
			ui.PrintInfo("Operation cancelled.")
			return nil
		}

		if err := c.RevokeSession(sessionID); err != nil {
			return err
		}

		ui.PrintSuccess("Session '%s' revoked successfully.", sessionID)
		return nil
	},
}

// ── Vault Providers Subcommands (`angler admin vaults`) ─────────────────────

var adminVaultsCmd = &cobra.Command{
	Use:     "vaults",
	Aliases: []string{"vault", "vault-providers"},
	Short:   "Configure, test, and audit Enterprise Secret Vault providers (Internal, Azure KV, AWS SM, HashiCorp Vault, GCP SM)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdminVaultsList(cmd, args)
	},
}

var adminVaultsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all supported vault providers and their configuration status",
	RunE:    runAdminVaultsList,
}

func runAdminVaultsList(cmd *cobra.Command, args []string) error {
	c := getClient()
	providers, err := c.ListVaultProviders()
	if err != nil {
		return err
	}

	return ui.RenderOutput(providers, func() {
		headers := []string{"PROVIDER", "DISPLAY NAME", "CONFIGURED"}
		rows := make([][]string, len(providers))
		for i, p := range providers {
			cfgStr := "No"
			if p.IsConfigured {
				cfgStr = "Yes"
			}
			rows[i] = []string{p.Provider, p.DisplayName, cfgStr}
		}
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total vault providers: %d", len(providers))
	})
}

var adminVaultsGetCmd = &cobra.Command{
	Use:   "get <provider>",
	Short: "Inspect a specific vault provider's configuration (secrets masked)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := args[0]
		c := getClient()
		prov, err := c.GetVaultProvider(provName)
		if err != nil {
			return err
		}

		return ui.RenderOutput(prov, func() {
			ui.PrintSuccess("Vault Provider: %s (%s)", prov.DisplayName, prov.Provider)
			fmt.Fprintf(ui.OutWriter, "  Configured: %v\n", prov.IsConfigured)
			if len(prov.Config) > 0 {
				fmt.Fprintln(ui.OutWriter, "  Settings:")
				for k, v := range prov.Config {
					fmt.Fprintf(ui.OutWriter, "    %s: %v\n", k, v)
				}
			}
		})
	},
}

func buildVaultConfigFromFlags() (map[string]any, error) {
	cfgMap := make(map[string]any)
	if adminVaultConfigFile != "" {
		b, err := os.ReadFile(adminVaultConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read --config-file %s: %w", adminVaultConfigFile, err)
		}
		if err := json.Unmarshal(b, &cfgMap); err != nil {
			return nil, ui.NewUsageError(fmt.Errorf("invalid JSON in --config-file: %w", err))
		}
	}
	if strings.TrimSpace(adminVaultConfigJSON) != "" {
		if err := json.Unmarshal([]byte(adminVaultConfigJSON), &cfgMap); err != nil {
			return nil, ui.NewUsageError(fmt.Errorf("invalid JSON in --config: %w", err))
		}
	}
	if adminVaultURL != "" {
		cfgMap["vault_url"] = adminVaultURL
	}
	if adminVaultRegion != "" {
		cfgMap["region_name"] = adminVaultRegion
	}
	if adminVaultProjectID != "" {
		cfgMap["project_id"] = adminVaultProjectID
	}
	if adminVaultAuthMethod != "" {
		cfgMap["auth_method"] = adminVaultAuthMethod
	}
	if adminVaultMountPoint != "" {
		cfgMap["mount_point"] = adminVaultMountPoint
	}
	if adminVaultNamespace != "" {
		cfgMap["namespace"] = adminVaultNamespace
	}
	if adminVaultToken != "" {
		cfgMap["token"] = adminVaultToken
	}
	if adminVaultRoleID != "" {
		cfgMap["role_id"] = adminVaultRoleID
	}
	if adminVaultSecretID != "" {
		cfgMap["secret_id"] = adminVaultSecretID
	}
	if adminVaultClientID != "" {
		cfgMap["client_id"] = adminVaultClientID
	}
	if adminVaultClientSecret != "" {
		cfgMap["client_secret"] = adminVaultClientSecret
	}
	if adminVaultTenantID != "" {
		cfgMap["tenant_id"] = adminVaultTenantID
	}
	if adminVaultSAFile != "" {
		b, err := os.ReadFile(adminVaultSAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read service account file %s: %w", adminVaultSAFile, err)
		}
		cfgMap["credentials_json"] = string(b)
	}
	return cfgMap, nil
}

var adminVaultsConfigureCmd = &cobra.Command{
	Use:     "configure <provider>",
	Aliases: []string{"update", "set"},
	Short:   "Update a vault provider's configuration",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := args[0]
		cfgMap, err := buildVaultConfigFromFlags()
		if err != nil {
			return err
		}

		c := getClient()
		res, err := c.UpdateVaultProvider(provName, cfgMap)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Updated configuration for vault provider '%s' (Configured: %v)", res.Provider, res.IsConfigured)
		})
	},
}

var adminVaultsTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Execute live reachability, authentication, and seal status verification against a vault provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := args[0]
		cfgMap, err := buildVaultConfigFromFlags()
		if err != nil {
			return err
		}

		c := getClient()
		res, err := c.TestVaultProvider(provName, cfgMap)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			if res.Success {
				ui.PrintSuccess("Vault connection test passed for '%s': %s", provName, res.Message)
			} else {
				ui.PrintError("Vault connection test failed for '%s': %s", provName, res.Message)
			}
		})
	},
}

var adminVaultsScopesCmd = &cobra.Command{
	Use:   "scopes",
	Short: "List all secret scopes across all workspaces for global security auditing",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		scopes, err := c.ListAllVaultScopes()
		if err != nil {
			return err
		}

		return ui.RenderOutput(scopes, func() {
			if len(scopes) == 0 {
				ui.PrintInfo("No secret scopes found across workspaces.")
				return
			}
			headers := []string{"SCOPE NAME", "WORKSPACE", "BACKEND TYPE", "KEYS"}
			rows := make([][]string, len(scopes))
			for i, s := range scopes {
				rows[i] = []string{
					s.Name,
					s.WorkspaceName,
					s.BackendType,
					fmt.Sprintf("%d", s.KeyCount),
				}
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total global secret scopes: %d", len(scopes))
		})
	},
}

// ── Multi-Provider AI Administration (`angler admin ai-providers`) ──────────

var adminAIProvidersCmd = &cobra.Command{
	Use:     "ai-providers",
	Aliases: []string{"ai-provider"},
	Short:   "Configure, test, and discover models across Enterprise AI Providers (Vertex AI, Bedrock, OpenAI, Anthropic, Azure, Gemini)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdminAIProvidersList(cmd, args)
	},
}

var adminAIProvidersListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "get"},
	Short:   "Inspect active AI provider and all provider configurations (secrets masked)",
	RunE:    runAdminAIProvidersList,
}

func runAdminAIProvidersList(cmd *cobra.Command, args []string) error {
	c := getClient()
	resp, err := c.GetAIProviderSettings()
	if err != nil {
		return err
	}

	return ui.RenderOutput(resp, func() {
		ui.PrintSuccess("Active Enterprise AI Provider: %s", resp.ActiveProvider)
		if len(resp.Providers) > 0 {
			headers := []string{"PROVIDER", "LABEL", "ACTIVE", "CONFIGURED"}
			rows := make([][]string, len(resp.Providers))
			for i, p := range resp.Providers {
				act := "No"
				if p.Active || p.Name == resp.ActiveProvider {
					act = "Yes"
				}
				cfg := "No"
				if p.Configured {
					cfg = "Yes"
				}
				rows[i] = []string{p.Name, p.Label, act, cfg}
			}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		}
		if len(resp.DeprecationWarning) > 0 {
			ui.PrintWarning("Deprecation Fallback Alert: %v", resp.DeprecationWarning)
		}
	})
}

func buildAIProviderConfigFromFlags() (map[string]any, error) {
	cfgMap := make(map[string]any)
	if adminAIConfigFile != "" {
		b, err := os.ReadFile(adminAIConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read --config-file %s: %w", adminAIConfigFile, err)
		}
		if err := json.Unmarshal(b, &cfgMap); err != nil {
			return nil, ui.NewUsageError(fmt.Errorf("invalid JSON in --config-file: %w", err))
		}
	}
	if strings.TrimSpace(adminAIConfigJSON) != "" {
		if err := json.Unmarshal([]byte(adminAIConfigJSON), &cfgMap); err != nil {
			return nil, ui.NewUsageError(fmt.Errorf("invalid JSON in --config: %w", err))
		}
	}
	if adminAIAPIKey != "" {
		cfgMap["api_key"] = adminAIAPIKey
	}
	if adminAIChatModel != "" {
		cfgMap["chat_model"] = adminAIChatModel
	}
	if adminAIEmbeddingModel != "" {
		cfgMap["embedding_model"] = adminAIEmbeddingModel
	}
	if adminAIBaseURL != "" {
		cfgMap["base_url"] = adminAIBaseURL
	}
	if adminAIProjectID != "" {
		cfgMap["project_id"] = adminAIProjectID
	}
	if adminAILocation != "" {
		cfgMap["location"] = adminAILocation
	}
	if adminAIRegion != "" {
		cfgMap["region"] = adminAIRegion
	}
	if adminAISAFile != "" {
		b, err := os.ReadFile(adminAISAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read --service-account-file %s: %w", adminAISAFile, err)
		}
		cfgMap["service_account_json"] = string(b)
	}
	return cfgMap, nil
}

var adminAIProvidersConfigureCmd = &cobra.Command{
	Use:     "configure <provider>",
	Aliases: []string{"update", "set"},
	Short:   "Save and optionally activate an AI provider configuration (gemini, vertex, openai, anthropic, azure_openai, bedrock)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := args[0]
		cfgMap, err := buildAIProviderConfigFromFlags()
		if err != nil {
			return err
		}

		c := getClient()
		res, err := c.SaveAIProviderSettings(client.SaveAIProviderRequest{
			Provider: provName,
			Config:   cfgMap,
			Activate: adminAIActivate,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Configured AI provider '%s' (Active Provider: %s)", provName, res.ActiveProvider)
		})
	},
}

var adminAIProvidersTestCmd = &cobra.Command{
	Use:   "test [provider]",
	Short: "Probe chat latency, embedding latency, and vector dimensions for an AI provider",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := ""
		if len(args) > 0 {
			provName = args[0]
		}
		cfgMap, err := buildAIProviderConfigFromFlags()
		if err != nil {
			return err
		}

		c := getClient()
		res, err := c.TestAIProvider(client.TestAIProviderRequest{
			Provider: provName,
			Config:   cfgMap,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			okVal, _ := res["ok"].(bool)
			msg, _ := res["message"].(string)
			prov, _ := res["provider"].(string)
			if okVal {
				ui.PrintSuccess("AI Provider '%s' connection test passed: %s", prov, msg)
			} else {
				ui.PrintError("AI Provider '%s' connection test failed: %s", prov, msg)
			}
			if dim, ok := res["embedding_dimension"]; ok {
				fmt.Fprintf(ui.OutWriter, "  Embedding Dimension: %v\n", dim)
			}
		})
	},
}

var adminAIProvidersDiscoverCmd = &cobra.Command{
	Use:   "discover-models [provider]",
	Short: "Dynamically query the provider's live API / Model Garden for available chat and embedding models",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provName := ""
		if len(args) > 0 {
			provName = args[0]
		}
		cfgMap, err := buildAIProviderConfigFromFlags()
		if err != nil {
			return err
		}

		c := getClient()
		res, err := c.DiscoverAIModels(client.DiscoverAIModelsRequest{
			Provider: provName,
			Config:   cfgMap,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Discovered Models for '%s': %s", res.Provider, res.Message)
			if chats := res.Models["chat"]; len(chats) > 0 {
				fmt.Fprintf(ui.OutWriter, "  Chat Models (%d):      %s\n", len(chats), strings.Join(chats, ", "))
			}
			if embeds := res.Models["embedding"]; len(embeds) > 0 {
				fmt.Fprintf(ui.OutWriter, "  Embedding Models (%d): %s\n", len(embeds), strings.Join(embeds, ", "))
			}
		})
	},
}

var adminAIProvidersDeprecationCmd = &cobra.Command{
	Use:   "deprecation",
	Short: "View active self-healing model deprecation fallback alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		res, err := c.GetAIDeprecationWarning()
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			if len(res.Warning) == 0 {
				ui.PrintSuccess("No active AI model deprecation warnings.")
				return
			}
			ui.PrintWarning("Active AI Model Deprecation Fallback Alert:")
			for k, v := range res.Warning {
				fmt.Fprintf(ui.OutWriter, "  %s: %v\n", k, v)
			}
		})
	},
}

var adminAIProvidersDismissDeprecationCmd = &cobra.Command{
	Use:   "dismiss-deprecation",
	Short: "Clear active self-healing model deprecation fallback alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		res, err := c.DismissAIDeprecationWarning()
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Dismissed AI model deprecation warning.")
		})
	},
}

func init() {
	// Settings update flags
	adminSettingsUpdateCmd.Flags().IntVar(&adminSessionTimeout, "session-timeout", 60, "Session inactivity timeout in minutes")
	adminSettingsUpdateCmd.Flags().IntVar(&adminMaxHours, "max-hours", 24, "Maximum session duration in hours")
	adminSettingsUpdateCmd.Flags().StringVar(&adminGeminiModel, "gemini-model", "", "Default Gemini AI model")
	adminSettingsUpdateCmd.Flags().StringVar(&adminGeminiKey, "gemini-key", "", "Gemini API key")

	// Session revoke flags
	adminSessionsRevokeCmd.Flags().BoolVarP(&adminForce, "force", "f", false, "Revoke session without confirmation prompt")

	// Vault configure & test flags
	for _, c := range []*cobra.Command{adminVaultsConfigureCmd, adminVaultsTestCmd} {
		c.Flags().StringVar(&adminVaultConfigJSON, "config", "", "JSON configuration payload for vault provider")
		c.Flags().StringVar(&adminVaultConfigFile, "config-file", "", "Path to JSON configuration file")
		c.Flags().StringVar(&adminVaultURL, "vault-url", "", "Vault URL (Azure Key Vault or HashiCorp Vault)")
		c.Flags().StringVar(&adminVaultRegion, "region", "", "AWS region name")
		c.Flags().StringVar(&adminVaultProjectID, "project-id", "", "GCP project ID")
		c.Flags().StringVar(&adminVaultAuthMethod, "auth-method", "", "Auth method (token, approle, kubernetes, adc, service_account)")
		c.Flags().StringVar(&adminVaultMountPoint, "mount-point", "", "HashiCorp Vault KV mount point")
		c.Flags().StringVar(&adminVaultNamespace, "namespace", "", "HashiCorp Vault Enterprise namespace")
		c.Flags().StringVar(&adminVaultToken, "vault-token", "", "HashiCorp Vault token")
		c.Flags().StringVar(&adminVaultRoleID, "role-id", "", "HashiCorp Vault AppRole role_id")
		c.Flags().StringVar(&adminVaultSecretID, "secret-id", "", "HashiCorp Vault AppRole secret_id")
		c.Flags().StringVar(&adminVaultSAFile, "service-account-file", "", "Path to GCP Service Account JSON file")
		c.Flags().StringVar(&adminVaultClientID, "client-id", "", "Azure client ID")
		c.Flags().StringVar(&adminVaultClientSecret, "client-secret", "", "Azure client secret")
		c.Flags().StringVar(&adminVaultTenantID, "tenant-id", "", "Azure tenant ID")
	}

	// AI Provider configure, test, discover flags
	adminAIProvidersConfigureCmd.Flags().BoolVar(&adminAIActivate, "activate", true, "Set this provider as the active cluster AI provider")
	for _, c := range []*cobra.Command{adminAIProvidersConfigureCmd, adminAIProvidersTestCmd, adminAIProvidersDiscoverCmd} {
		c.Flags().StringVar(&adminAIConfigJSON, "config", "", "JSON configuration dictionary for the AI provider")
		c.Flags().StringVar(&adminAIConfigFile, "config-file", "", "Path to JSON configuration file")
		c.Flags().StringVar(&adminAIAPIKey, "api-key", "", "Provider API key")
		c.Flags().StringVar(&adminAIChatModel, "chat-model", "", "Default chat/completion model ID")
		c.Flags().StringVar(&adminAIEmbeddingModel, "embedding-model", "", "Default text embedding model ID")
		c.Flags().StringVar(&adminAIBaseURL, "base-url", "", "Custom base URL or Azure endpoint")
		c.Flags().StringVar(&adminAIProjectID, "project-id", "", "Google Cloud Vertex AI project ID")
		c.Flags().StringVar(&adminAILocation, "location", "", "Google Cloud Vertex AI region/location (e.g. us-central1)")
		c.Flags().StringVar(&adminAISAFile, "service-account-file", "", "Path to Google Cloud Service Account JSON key file")
		c.Flags().StringVar(&adminAIRegion, "region", "", "AWS Bedrock region")
	}

	// Settings hierarchy
	adminSettingsCmd.AddCommand(adminSettingsGetCmd)
	adminSettingsCmd.AddCommand(adminSettingsUpdateCmd)

	// Sessions hierarchy
	adminSessionsCmd.AddCommand(adminSessionsListCmd)
	adminSessionsCmd.AddCommand(adminSessionsRevokeCmd)

	// Vaults hierarchy
	adminVaultsCmd.AddCommand(adminVaultsListCmd)
	adminVaultsCmd.AddCommand(adminVaultsGetCmd)
	adminVaultsCmd.AddCommand(adminVaultsConfigureCmd)
	adminVaultsCmd.AddCommand(adminVaultsTestCmd)
	adminVaultsCmd.AddCommand(adminVaultsScopesCmd)

	// AI Providers hierarchy
	adminAIProvidersCmd.AddCommand(adminAIProvidersListCmd)
	adminAIProvidersCmd.AddCommand(adminAIProvidersConfigureCmd)
	adminAIProvidersCmd.AddCommand(adminAIProvidersTestCmd)
	adminAIProvidersCmd.AddCommand(adminAIProvidersDiscoverCmd)
	adminAIProvidersCmd.AddCommand(adminAIProvidersDeprecationCmd)
	adminAIProvidersCmd.AddCommand(adminAIProvidersDismissDeprecationCmd)

	// Admin hierarchy
	adminCmd.AddCommand(adminSettingsCmd)
	adminCmd.AddCommand(adminSessionsCmd)
	adminCmd.AddCommand(adminVaultsCmd)
	adminCmd.AddCommand(adminAIProvidersCmd)

	rootCmd.AddCommand(adminCmd)
}
