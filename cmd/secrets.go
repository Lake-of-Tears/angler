package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	secretScopeBackend string
	secretScopeConfig  string
	secretStringValue  string
	secretValueStdin   bool
	secretGetRaw       bool
	secretGetUnmask    bool
	secretGetShowValue bool
	secretForce        bool
	secretACLPrincipal string
	secretACLType      string
	secretACLPerm      string
)

var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Aliases: []string{"secret"},
	Short:   "Enterprise Secret Scopes, Encrypted Keys, and Scope ACLs Management",
	Long: `Manage workspace secret scopes, encrypted key-value pairs, and fine-grained
access control rules (READ, WRITE, MANAGE) backed by Internal DB, Azure Key Vault,
AWS Secrets Manager, HashiCorp Vault, or Google Cloud Secret Manager.

Modeled after 'databricks secrets' for seamless CI/CD and developer workflows.

Examples:
  angler secrets scopes list
  angler secrets scopes create prod-vault --backend hashicorp_vault --config '{"mount_point":"secret"}'
  angler secrets put prod-vault db_password --string-value "s3cr3t!"
  echo "s3cr3t!" | angler secrets put prod-vault db_password --value-stdin
  angler secrets list prod-vault
  angler secrets get prod-vault db_password
  export DB_PASS=$(angler secrets get prod-vault db_password --raw)
  angler secrets acls list prod-vault
  angler secrets acls set prod-vault --principal read --principal-type role --permission READ`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSecretsScopesList(cmd, args)
	},
}

// ── 1. Workspace Secret Scopes (`angler secrets scopes`) ────────────────────

var secretsScopesCmd = &cobra.Command{
	Use:     "scopes",
	Aliases: []string{"scope"},
	Short:   "Create, inspect, list, and delete workspace secret scopes",
	RunE:    runSecretsScopesList,
}

var secretsScopesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all secret scopes in the active workspace",
	RunE:    runSecretsScopesList,
}

var secretsListScopesAliasCmd = &cobra.Command{
	Use:   "list-scopes",
	Short: "List all secret scopes in the active workspace (Databricks-compatible alias)",
	RunE:  runSecretsScopesList,
}

func runSecretsScopesList(cmd *cobra.Command, args []string) error {
	c := getClient()
	scopes, err := c.ListSecretScopes()
	if err != nil {
		return err
	}

	return ui.RenderOutput(scopes, func() {
		if len(scopes) == 0 {
			ui.PrintInfo("No secret scopes found in active workspace.")
			return
		}

		headers := []string{"SCOPE NAME", "BACKEND TYPE", "PERMISSION", "KEYS", "CREATED AT"}
		rows := make([][]string, len(scopes))
		for i, s := range scopes {
			rows[i] = []string{
				s.Name,
				s.BackendType,
				s.UserPermission,
				fmt.Sprintf("%d", s.KeyCount),
				s.CreatedAt,
			}
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total secret scopes: %d", len(scopes))
	})
}

var secretsScopesGetCmd = &cobra.Command{
	Use:   "get <scope>",
	Short: "Inspect a secret scope's metadata and provider configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		c := getClient()
		sc, err := c.GetSecretScope(scopeName)
		if err != nil {
			return err
		}

		return ui.RenderOutput(sc, func() {
			ui.PrintSuccess("Secret Scope: %s (%s)", sc.Name, sc.ID)
			fmt.Fprintf(ui.OutWriter, "  Backend Type: %s\n", sc.BackendType)
			fmt.Fprintf(ui.OutWriter, "  Permission:   %s\n", sc.UserPermission)
			fmt.Fprintf(ui.OutWriter, "  Key Count:    %d\n", sc.KeyCount)
			fmt.Fprintf(ui.OutWriter, "  Created At:   %s\n", sc.CreatedAt)
			if len(sc.ProviderConfig) > 0 {
				cfgBytes, _ := json.Marshal(sc.ProviderConfig)
				fmt.Fprintf(ui.OutWriter, "  Config:       %s\n", string(cfgBytes))
			}
		})
	},
}

var secretsScopesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace secret scope",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var providerCfg map[string]any
		if strings.TrimSpace(secretScopeConfig) != "" {
			if err := json.Unmarshal([]byte(secretScopeConfig), &providerCfg); err != nil {
				return ui.NewUsageError(fmt.Errorf("invalid JSON for --config: %w", err))
			}
		}

		c := getClient()
		sc, err := c.CreateSecretScope(client.SecretScopeCreateRequest{
			Name:           name,
			BackendType:    secretScopeBackend,
			ProviderConfig: providerCfg,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(sc, func() {
			ui.PrintSuccess("Created secret scope '%s' (Backend: %s)", sc.Name, sc.BackendType)
		})
	},
}

var secretsScopesDeleteCmd = &cobra.Command{
	Use:     "delete <scope>",
	Aliases: []string{"rm"},
	Short:   "Delete a secret scope and its keys",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		if !secretForce && !confirmAction(fmt.Sprintf("Permanently delete secret scope '%s'?", scopeName)) {
			ui.PrintInfo("Deletion cancelled.")
			return nil
		}

		c := getClient()
		if err := c.DeleteSecretScope(scopeName); err != nil {
			return err
		}

		payload := map[string]any{"status": "deleted", "scope": scopeName}
		return ui.RenderOutput(payload, func() {
			ui.PrintSuccess("Deleted secret scope '%s'", scopeName)
		})
	},
}

// ── 2. Secret Key Operations (`angler secrets list|put|get|delete`) ─────────

var secretsListCmd = &cobra.Command{
	Use:     "list <scope>",
	Aliases: []string{"ls"},
	Short:   "List secret key names, versions, and timestamps in a scope (without exposing values)",
	Args:    cobra.ExactArgs(1),
	RunE:    runSecretsKeysList,
}

var secretsKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage secret keys within a scope",
}

var secretsKeysListCmd = &cobra.Command{
	Use:     "list <scope>",
	Aliases: []string{"ls"},
	Short:   "List secret keys in a scope",
	Args:    cobra.ExactArgs(1),
	RunE:    runSecretsKeysList,
}

func runSecretsKeysList(cmd *cobra.Command, args []string) error {
	scopeName := args[0]
	c := getClient()
	keys, err := c.ListSecretKeys(scopeName)
	if err != nil {
		return err
	}

	return ui.RenderOutput(keys, func() {
		if len(keys) == 0 {
			ui.PrintInfo("No secret keys found in scope '%s'.", scopeName)
			return
		}

		headers := []string{"KEY", "VERSION", "UPDATED AT"}
		rows := make([][]string, len(keys))
		for i, k := range keys {
			upd := k.CreatedAt
			if k.UpdatedAt != nil && *k.UpdatedAt != "" {
				upd = *k.UpdatedAt
			}
			rows[i] = []string{
				k.Key,
				fmt.Sprintf("v%d", k.Version),
				upd,
			}
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total keys in '%s': %d", scopeName, len(keys))
	})
}

var secretsPutCmd = &cobra.Command{
	Use:   "put <scope> <key>",
	Short: "Create or rotate a secret value in a scope",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		keyName := args[1]

		var val string
		switch {
		case cmd.Flags().Changed("string-value"):
			val = secretStringValue
		case secretValueStdin:
			rawBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read secret value from stdin: %w", err)
			}
			val = strings.TrimRight(string(rawBytes), "\r\n")
		default:
			if ui.IsNonInteractive() {
				return ui.NewUsageError(fmt.Errorf("please provide --string-value or --value-stdin when running non-interactively"))
			}
			fmt.Fprintf(ui.ErrWriter, "Enter secret value for %s/%s: ", scopeName, keyName)
			if term.IsTerminal(int(os.Stdin.Fd())) {
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(ui.ErrWriter)
				if err != nil {
					return fmt.Errorf("failed to read secret value: %w", err)
				}
				val = string(b)
			} else {
				reader := bufio.NewReader(os.Stdin)
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return fmt.Errorf("failed to read secret value: %w", err)
				}
				val = strings.TrimRight(line, "\r\n")
			}
		}

		if val == "" {
			return ui.NewUsageError(fmt.Errorf("secret value cannot be empty"))
		}

		c := getClient()
		summary, err := c.PutSecretKey(scopeName, keyName, val)
		if err != nil {
			return err
		}

		return ui.RenderOutput(summary, func() {
			ui.PrintSuccess("Stored secret '%s' in scope '%s' (version v%d)", summary.Key, scopeName, summary.Version)
		})
	},
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <scope> <key>",
	Short: "Retrieve secret metadata or value (masked in table output unless --unmask or --raw is used)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		keyName := args[1]

		c := getClient()
		sec, err := c.GetSecretKey(scopeName, keyName)
		if err != nil {
			return err
		}

		if secretGetRaw {
			fmt.Fprintln(ui.OutWriter, sec.Value)
			return nil
		}

		showPlain := secretGetUnmask || secretGetShowValue
		outPayload := *sec
		if !showPlain && ui.GetOutputFormat() == ui.FormatTable {
			outPayload.Value = "••••••••"
		}

		return ui.RenderOutput(outPayload, func() {
			displayVal := "•••••••• (pass --unmask or --raw to view value)"
			if showPlain {
				displayVal = sec.Value
			}
			ui.PrintSuccess("Secret: %s/%s (v%d)", scopeName, sec.Key, sec.Version)
			fmt.Fprintf(ui.OutWriter, "  Key:        %s\n", sec.Key)
			fmt.Fprintf(ui.OutWriter, "  Value:      %s\n", displayVal)
			fmt.Fprintf(ui.OutWriter, "  Version:    v%d\n", sec.Version)
			if sec.UpdatedAt != nil && *sec.UpdatedAt != "" {
				fmt.Fprintf(ui.OutWriter, "  Updated At: %s\n", *sec.UpdatedAt)
			}
		})
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:     "delete <scope> <key>",
	Aliases: []string{"rm"},
	Short:   "Delete a secret key from a scope",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		keyName := args[1]

		if !secretForce && !confirmAction(fmt.Sprintf("Permanently delete secret key '%s/%s'?", scopeName, keyName)) {
			ui.PrintInfo("Deletion cancelled.")
			return nil
		}

		c := getClient()
		if err := c.DeleteSecretKey(scopeName, keyName); err != nil {
			return err
		}

		payload := map[string]any{"status": "deleted", "scope": scopeName, "key": keyName}
		return ui.RenderOutput(payload, func() {
			ui.PrintSuccess("Deleted secret key '%s' from scope '%s'", keyName, scopeName)
		})
	},
}

// ── 3. Secret Scope ACL Management (`angler secrets acls`) ──────────────────

var secretsACLsCmd = &cobra.Command{
	Use:     "acls",
	Aliases: []string{"acl"},
	Short:   "List, grant, and revoke access control rules on a secret scope",
}

var secretsACLsListCmd = &cobra.Command{
	Use:     "list <scope>",
	Aliases: []string{"ls"},
	Short:   "List all access control rules on a secret scope",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		c := getClient()
		acls, err := c.ListSecretACLs(scopeName)
		if err != nil {
			return err
		}

		return ui.RenderOutput(acls, func() {
			if len(acls) == 0 {
				ui.PrintInfo("No custom ACLs configured on scope '%s'.", scopeName)
				return
			}

			headers := []string{"ACL ID", "PRINCIPAL TYPE", "PRINCIPAL", "PERMISSION", "CREATED AT"}
			rows := make([][]string, len(acls))
			for i, a := range acls {
				rows[i] = []string{
					a.ID,
					a.PrincipalType,
					a.PrincipalID,
					a.Permission,
					a.CreatedAt,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total ACLs on '%s': %d", scopeName, len(acls))
		})
	},
}

var secretsACLsSetCmd = &cobra.Command{
	Use:     "set <scope>",
	Aliases: []string{"put", "grant"},
	Short:   "Grant or update an ACL rule on a secret scope (READ, WRITE, MANAGE)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		if strings.TrimSpace(secretACLPrincipal) == "" {
			return ui.NewUsageError(fmt.Errorf("--principal is required"))
		}
		perm := strings.ToUpper(strings.TrimSpace(secretACLPerm))
		if perm != "READ" && perm != "WRITE" && perm != "MANAGE" {
			return ui.NewUsageError(fmt.Errorf("--permission must be READ, WRITE, or MANAGE"))
		}

		c := getClient()
		acl, err := c.SetSecretACL(scopeName, client.SecretACLSetRequest{
			PrincipalType: strings.ToLower(strings.TrimSpace(secretACLType)),
			PrincipalID:   strings.TrimSpace(secretACLPrincipal),
			Permission:    perm,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(acl, func() {
			ui.PrintSuccess("Granted %s permission on scope '%s' to %s '%s'", acl.Permission, scopeName, acl.PrincipalType, acl.PrincipalID)
		})
	},
}

var secretsACLsDeleteCmd = &cobra.Command{
	Use:     "delete <scope> <acl_id>",
	Aliases: []string{"rm", "revoke"},
	Short:   "Revoke an ACL rule from a secret scope",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeName := args[0]
		aclID := args[1]

		if !secretForce && !confirmAction(fmt.Sprintf("Revoke ACL '%s' on scope '%s'?", aclID, scopeName)) {
			ui.PrintInfo("Revocation cancelled.")
			return nil
		}

		c := getClient()
		if err := c.DeleteSecretACL(scopeName, aclID); err != nil {
			return err
		}

		payload := map[string]any{"status": "deleted", "scope": scopeName, "acl_id": aclID}
		return ui.RenderOutput(payload, func() {
			ui.PrintSuccess("Revoked ACL '%s' on scope '%s'", aclID, scopeName)
		})
	},
}

func init() {
	// scopes create & delete flags
	secretsScopesCreateCmd.Flags().StringVar(&secretScopeBackend, "backend", "internal", "Vault backend: internal, azure_keyvault, aws_secrets_manager, hashicorp_vault, gcp_secret_manager")
	secretsScopesCreateCmd.Flags().StringVar(&secretScopeConfig, "config", "", "Provider-specific scope configuration JSON")
	secretsScopesDeleteCmd.Flags().BoolVarP(&secretForce, "force", "f", false, "Delete scope without confirmation")

	// put flags
	secretsPutCmd.Flags().StringVar(&secretStringValue, "string-value", "", "Secret string value to store")
	secretsPutCmd.Flags().BoolVar(&secretValueStdin, "value-stdin", false, "Read secret value from standard input (safe for CI/CD)")

	// get flags
	secretsGetCmd.Flags().BoolVar(&secretGetRaw, "raw", false, "Print raw unmasked secret value directly to stdout for piping")
	secretsGetCmd.Flags().BoolVar(&secretGetUnmask, "unmask", false, "Unmask the secret value in table output")
	secretsGetCmd.Flags().BoolVar(&secretGetShowValue, "show-value", false, "Unmask the secret value in table output")

	// delete key flags
	secretsDeleteCmd.Flags().BoolVarP(&secretForce, "force", "f", false, "Delete key without confirmation")

	// acls flags
	secretsACLsSetCmd.Flags().StringVar(&secretACLPrincipal, "principal", "", "Target user ID/email or workspace role (admin, write, read)")
	secretsACLsSetCmd.Flags().StringVar(&secretACLType, "principal-type", "role", "Principal type: 'role' or 'user'")
	secretsACLsSetCmd.Flags().StringVar(&secretACLPerm, "permission", "READ", "Permission level: READ, WRITE, or MANAGE")
	secretsACLsDeleteCmd.Flags().BoolVarP(&secretForce, "force", "f", false, "Revoke ACL without confirmation")

	// scopes hierarchy
	secretsScopesCmd.AddCommand(secretsScopesListCmd)
	secretsScopesCmd.AddCommand(secretsScopesGetCmd)
	secretsScopesCmd.AddCommand(secretsScopesCreateCmd)
	secretsScopesCmd.AddCommand(secretsScopesDeleteCmd)

	// keys hierarchy
	secretsKeysCmd.AddCommand(secretsKeysListCmd)

	// acls hierarchy
	secretsACLsCmd.AddCommand(secretsACLsListCmd)
	secretsACLsCmd.AddCommand(secretsACLsSetCmd)
	secretsACLsCmd.AddCommand(secretsACLsDeleteCmd)

	// root secrets hierarchy
	secretsCmd.AddCommand(secretsScopesCmd)
	secretsCmd.AddCommand(secretsListScopesAliasCmd)
	secretsCmd.AddCommand(secretsKeysCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsPutCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
	secretsCmd.AddCommand(secretsACLsCmd)

	rootCmd.AddCommand(secretsCmd)
}
