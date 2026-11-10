package cmd

import (
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	catDescription    string
	catName           string
	catS3Path         string
	catWorkspace      string
	catPurge          bool
	catForce          bool
	catCascade        bool
	catAnomalySource  string
	catAnomalyTable   string
	catAnomalyContam  float64
	catRestoreStrat   string
	catRestoreNewName string
	catAccessMode     string
	catAccessMessage  string
	catToonRaw        bool
)

var catalogCmd = &cobra.Command{
	Use:     "catalog",
	Aliases: []string{"net"},
	Short:   "Unity Catalog-style metadata discovery and management (alias: net)",
	Long: `Inspect datasets, catalogs, database schemas, tables, soft-delete trash, TOON projections,
cross-workspace sharing, and data anomalies in Lake of Tears.
Modeled after Databricks Unity Catalog (catalogs, schemas, tables).

Aliases:
  angler catalog  - Unity Catalog standard command suite
  angler net      - Nautical Lake of Tears dataset inspect alias

Examples:
  angler catalog list
  angler catalog get main
  angler catalog update main --description "Production Lakehouse"
  angler catalog schemas list main
  angler catalog tables list main default
  angler catalog tables get main default users
  angler catalog toon main default users
  angler catalog trash list
  angler catalog undrop <asset_id>
  angler catalog directory
  angler catalog access list main
  angler catalog manifest
  angler catalog anomalies detect --table users`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogList()
	},
}

var catalogListCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all registered catalogs in the active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogList()
	},
}

var catalogGetCmd = &cobra.Command{
	Use:   "get <catalog_id>",
	Short: "Inspect catalog metadata, schemas, and workspace ownership",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		c := getClient()
		cat, err := c.GetCatalog(catalogID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(cat, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, fmt.Sprintf("Catalog: %s", cat.Name)))
			fmt.Fprintf(ui.OutWriter, "  ID:          %s\n", cat.ID)
			fmt.Fprintf(ui.OutWriter, "  Slug:        %s\n", cat.Slug)
			if cat.Description != "" {
				fmt.Fprintf(ui.OutWriter, "  Description: %s\n", cat.Description)
			}
			if cat.OwnerWorkspaceName != "" {
				fmt.Fprintf(ui.OutWriter, "  Workspace:   %s (%s)\n", cat.OwnerWorkspaceName, cat.OwnerWorkspaceID)
			}
			if cat.MyAccess != "" {
				fmt.Fprintf(ui.OutWriter, "  Access:      %s\n", cat.MyAccess)
			}
			if cat.IsDefault {
				fmt.Fprintln(ui.OutWriter, "  Default:     true")
			}

			if len(cat.Schemas) > 0 {
				fmt.Fprintln(ui.OutWriter)
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Schemas:"))
				for _, sch := range cat.Schemas {
					fmt.Fprintf(ui.OutWriter, "  - %-16s (tier: %s, tables: %d)\n", sch.Name, sch.Tier, len(sch.Tables))
				}
			}
		})
	},
}

var catalogCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Register a new catalog in the workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c := getClient()
		cat, err := c.CreateCatalog(client.CreateCatalogRequest{
			Name:        name,
			Description: catDescription,
			WorkspaceID: catWorkspace,
		})
		if err != nil {
			return err
		}

		ui.PrintSuccess("Catalog '%s' created (ID: %s).", cat.Name, cat.ID)
		return nil
	},
}

var catalogDeleteCmd = &cobra.Command{
	Use:     "delete <catalog_id>",
	Aliases: []string{"rm"},
	Short:   "Delete or purge a catalog",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			action := "delete"
			if catPurge {
				action = "PERMANENTLY PURGE"
			}
			if !confirmAction(fmt.Sprintf("Are you sure you want to %s catalog '%s'?", action, catalogID)) {
				ui.PrintWarning("Catalog deletion aborted.")
				return nil
			}
		}

		c := getClient()
		if err := c.DeleteCatalog(catalogID, catPurge); err != nil {
			return err
		}

		ui.PrintSuccess("Catalog '%s' deleted.", catalogID)
		return nil
	},
}

// ── Schema Commands ──────────────────────────────────────────────────────────

var catalogSchemasCmd = &cobra.Command{
	Use:     "schemas",
	Aliases: []string{"schema"},
	Short:   "Manage and inspect catalog database schemas",
	Long:    `List, create, inspect, and delete schemas within a catalog.`,
}

var catalogSchemasListCmd = &cobra.Command{
	Use:     "list <catalog_id>",
	Aliases: []string{"ls"},
	Short:   "List all schemas in a catalog",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		c := getClient()
		schemas, err := c.ListSchemas(catalogID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(schemas, func() {
			if len(schemas) == 0 {
				ui.PrintInfo("No schemas found in catalog '%s'.", catalogID)
				return
			}

			headers := []string{"SCHEMA NAME", "TIER", "DESCRIPTION", "CREATED"}
			rows := make([][]string, len(schemas))
			for i, s := range schemas {
				rows[i] = []string{
					s.Name,
					s.Tier,
					s.Description,
					s.CreatedAt.Format("2006-01-02 15:04"),
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Showing %d schemas.", len(schemas))
		})
	},
}

var catalogSchemasGetCmd = &cobra.Command{
	Use:   "get <catalog_id> <schema_id>",
	Short: "Inspect schema details and its tables",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]

		c := getClient()
		sch, err := c.GetSchema(catalogID, schemaID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(sch, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, fmt.Sprintf("Schema: %s", sch.Name)))
			fmt.Fprintf(ui.OutWriter, "  ID:          %s\n", sch.ID)
			fmt.Fprintf(ui.OutWriter, "  Catalog:     %s\n", sch.CatalogID)
			fmt.Fprintf(ui.OutWriter, "  Tier:        %s\n", sch.Tier)
			if sch.Description != "" {
				fmt.Fprintf(ui.OutWriter, "  Description: %s\n", sch.Description)
			}

			if len(sch.Tables) > 0 {
				fmt.Fprintln(ui.OutWriter)
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Tables:"))
				for _, t := range sch.Tables {
					fmt.Fprintf(ui.OutWriter, "  - %-20s (path: %s)\n", t.Name, t.S3PathPattern)
				}
			}
		})
	},
}

var catalogSchemasCreateCmd = &cobra.Command{
	Use:   "create <catalog_id> <name>",
	Short: "Create a database schema in a catalog",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		name := args[1]

		c := getClient()
		sch, err := c.CreateSchema(catalogID, client.CreateSchemaRequest{
			Name:        name,
			Description: catDescription,
		})
		if err != nil {
			return err
		}

		ui.PrintSuccess("Schema '%s' created in catalog '%s' (ID: %s).", sch.Name, catalogID, sch.ID)
		return nil
	},
}

var catalogSchemasDeleteCmd = &cobra.Command{
	Use:     "delete <catalog_id> <schema_id>",
	Aliases: []string{"rm"},
	Short:   "Drop a schema from a catalog",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]

		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			if !confirmAction(fmt.Sprintf("Are you sure you want to drop schema '%s' in catalog '%s'?", schemaID, catalogID)) {
				ui.PrintWarning("Schema deletion aborted.")
				return nil
			}
		}

		c := getClient()
		if err := c.DeleteSchema(catalogID, schemaID, catCascade); err != nil {
			return err
		}

		ui.PrintSuccess("Schema '%s' dropped.", schemaID)
		return nil
	},
}

// ── Table Commands ───────────────────────────────────────────────────────────

var catalogTablesCmd = &cobra.Command{
	Use:     "tables",
	Aliases: []string{"table"},
	Short:   "Manage and inspect catalog tables",
	Long:    `List, inspect, and drop tables within a catalog schema.`,
}

var catalogTablesListCmd = &cobra.Command{
	Use:     "list <catalog_id> <schema_id>",
	Aliases: []string{"ls"},
	Short:   "List all tables in a catalog schema",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]

		c := getClient()
		tables, err := c.ListTables(catalogID, schemaID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(tables, func() {
			if len(tables) == 0 {
				ui.PrintInfo("No tables found in %s.%s", catalogID, schemaID)
				return
			}

			headers := []string{"TABLE NAME", "COLUMNS", "S3 STORAGE PATH", "DRIFT"}
			rows := make([][]string, len(tables))
			for i, t := range tables {
				colCount := fmt.Sprintf("%d", len(t.ColumnDefs))
				driftStr := "No"
				if t.SchemaDrift {
					driftStr = "DRIFT DETECTED"
				}
				rows[i] = []string{
					t.Name,
					colCount,
					t.S3PathPattern,
					driftStr,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Showing %d tables.", len(tables))
		})
	},
}

var catalogTablesGetCmd = &cobra.Command{
	Use:   "get <catalog_id> <schema_id> <table_id>",
	Short: "Inspect table schema, column types, and storage pattern",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]
		tableID := args[2]

		c := getClient()
		tbl, err := c.GetTable(catalogID, schemaID, tableID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(tbl, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, fmt.Sprintf("Table: %s", tbl.Name)))
			fmt.Fprintf(ui.OutWriter, "  ID:          %s\n", tbl.ID)
			fmt.Fprintf(ui.OutWriter, "  Schema ID:   %s\n", tbl.SchemaID)
			if tbl.Description != "" {
				fmt.Fprintf(ui.OutWriter, "  Description: %s\n", tbl.Description)
			}
			if tbl.S3PathPattern != "" {
				fmt.Fprintf(ui.OutWriter, "  Path:        %s\n", tbl.S3PathPattern)
			}
			if tbl.SchemaDrift {
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.WarningStyle, "  Status:      SCHEMA DRIFT DETECTED"))
			}

			if len(tbl.ColumnDefs) > 0 {
				fmt.Fprintln(ui.OutWriter)
				fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.SuccessStyle, "Columns:"))
				for _, col := range tbl.ColumnDefs {
					cName := fmt.Sprintf("%v", col["name"])
					cType := fmt.Sprintf("%v", col["type"])
					fmt.Fprintf(ui.OutWriter, "  - %-20s %s\n", cName, cType)
				}
			}
		})
	},
}

var catalogTablesDeleteCmd = &cobra.Command{
	Use:     "delete <catalog_id> <schema_id> <table_id>",
	Aliases: []string{"rm"},
	Short:   "Drop a table from a catalog schema",
	Args:    cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]
		tableID := args[2]

		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			if !confirmAction(fmt.Sprintf("Are you sure you want to drop table '%s'?", tableID)) {
				ui.PrintWarning("Table deletion aborted.")
				return nil
			}
		}

		c := getClient()
		if err := c.DeleteTable(catalogID, schemaID, tableID); err != nil {
			return err
		}

		ui.PrintSuccess("Table '%s' dropped.", tableID)
		return nil
	},
}

// ── Manifest Command ─────────────────────────────────────────────────────────

var catalogManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Export catalog manifest projection JSON",
	Long:  `Retrieve the full hierarchical projection of registered catalogs, schemas, and tables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		manifest, err := c.GetCatalogManifest()
		if err != nil {
			return err
		}

		return ui.RenderOutput(manifest, func() {
			out, _ := ui.FormatData(ui.FormatJSON, manifest)
			fmt.Fprintln(ui.OutWriter, out)
		})
	},
}

// ── Anomalies Command ────────────────────────────────────────────────────────

var catalogAnomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Data and table anomaly detection suite",
}

var catalogAnomaliesDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect anomalies across catalog tables",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		ui.PrintInfo("Running anomaly detection...")
		resp, err := c.DetectAnomalies(client.AnomalyDetectRequest{
			Source:        catAnomalySource,
			TableSlug:     catAnomalyTable,
			Contamination: catAnomalyContam,
		})
		if err != nil {
			return err
		}

		if !resp.OK && resp.Error != "" {
			return ui.NewCLIError(ui.ExitGeneralError, fmt.Errorf("anomaly detection failed: %s", resp.Error))
		}

		return ui.RenderOutput(resp, func() {
			fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Anomaly Detection Results"))
			fmt.Fprintf(ui.OutWriter, "  Workspace: %s\n", resp.Workspace)
			if resp.Summary != nil {
				data, _ := ui.FormatData(ui.FormatJSON, resp.Summary)
				fmt.Fprintln(ui.OutWriter, data)
			}
			ui.PrintSuccess("Anomaly detection completed.")
		})
	},
}

func runCatalogList() error {
	c := getClient()
	catalogs, err := c.ListCatalogs()
	if err == nil && len(catalogs) > 0 {
		return ui.RenderOutput(catalogs, func() {
			headers := []string{"CATALOG NAME", "SLUG", "SCHEMAS", "ACCESS", "DEFAULT"}
			rows := make([][]string, len(catalogs))
			for i, cat := range catalogs {
				defStr := "No"
				if cat.IsDefault {
					defStr = "Yes"
				}
				rows[i] = []string{
					cat.Name,
					cat.Slug,
					fmt.Sprintf("%d", len(cat.Schemas)),
					cat.MyAccess,
					defStr,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total catalogs: %d", len(catalogs))
		})
	}

	// Fallback to legacy /api/catalog if /api/catalogs is empty or unavailable
	items, legacyErr := c.ListCatalog()
	if legacyErr != nil {
		if err != nil {
			return err
		}
		return legacyErr
	}

	return ui.RenderOutput(items, func() {
		if len(items) == 0 {
			ui.PrintInfo("No datasets or catalogs found.")
			return
		}

		tableRows := make([][]string, len(items))
		for i, item := range items {
			tableRows[i] = []string{
				item.Name,
				item.Type,
				item.Format,
				fmt.Sprintf("%d", item.Rows),
				item.Size,
				item.LastUpdated,
			}
		}

		headers := []string{"Dataset Name", "Type", "Format", "Rows", "Size", "Updated"}
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
		ui.PrintSuccess("Total datasets netted: %d", len(items))
	})
}

// ── Catalog, Schema & Table Updates + Native TOON Export ─────────────────────

var catalogUpdateCmd = &cobra.Command{
	Use:   "update <catalog_id>",
	Short: "Update catalog description or name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		var req client.UpdateCatalogRequest
		if cmd.Flags().Changed("name") {
			req.Name = &catName
		}
		if cmd.Flags().Changed("description") {
			req.Description = &catDescription
		}
		if req.Name == nil && req.Description == nil {
			return ui.NewUsageError(fmt.Errorf("at least one of --description or --name must be specified"))
		}

		c := getClient()
		cat, err := c.UpdateCatalog(catalogID, req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(cat, func() {
			ui.PrintSuccess("Catalog '%s' updated.", cat.Name)
		})
	},
}

var catalogSchemasUpdateCmd = &cobra.Command{
	Use:   "update <catalog_id> <schema_id>",
	Short: "Update schema description",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]

		var req client.UpdateSchemaRequest
		if cmd.Flags().Changed("description") {
			req.Description = &catDescription
		} else {
			return ui.NewUsageError(fmt.Errorf("--description must be specified"))
		}

		c := getClient()
		sch, err := c.UpdateSchema(catalogID, schemaID, req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(sch, func() {
			ui.PrintSuccess("Schema '%s' updated in catalog '%s'.", sch.Name, catalogID)
		})
	},
}

var catalogTablesUpdateCmd = &cobra.Command{
	Use:   "update <catalog_id> <schema_id> <table_id>",
	Short: "Update table description or S3 path pattern",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]
		tableID := args[2]

		var req client.UpdateTableRequest
		if cmd.Flags().Changed("description") {
			req.Description = &catDescription
		}
		if cmd.Flags().Changed("s3-path") {
			req.S3PathPattern = &catS3Path
		}
		if req.Description == nil && req.S3PathPattern == nil {
			return ui.NewUsageError(fmt.Errorf("at least one of --description or --s3-path must be specified"))
		}

		c := getClient()
		tbl, err := c.UpdateTable(catalogID, schemaID, tableID, req)
		if err != nil {
			return err
		}

		return ui.RenderOutput(tbl, func() {
			ui.PrintSuccess("Table '%s' updated.", tbl.Name)
		})
	},
}

var catalogToonCmd = &cobra.Command{
	Use:   "toon <catalog_id> <schema_id> [table_id]",
	Short: "Export token-efficient TOON (Token-Oriented Object Notation) schema or table projection",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		schemaID := args[1]
		c := getClient()

		if len(args) == 3 {
			tableID := args[2]
			res, err := c.GetTableTOON(catalogID, schemaID, tableID)
			if err != nil {
				return err
			}
			if catToonRaw {
				fmt.Fprintln(ui.OutWriter, strings.TrimSpace(res.TOON))
				return nil
			}
			return ui.RenderOutput(res, func() {
				fmt.Fprintln(ui.OutWriter, strings.TrimSpace(res.TOON))
			})
		}

		res, err := c.GetSchemaTOON(catalogID, schemaID)
		if err != nil {
			return err
		}
		if catToonRaw {
			fmt.Fprintln(ui.OutWriter, strings.TrimSpace(res.TOON))
			return nil
		}
		return ui.RenderOutput(res, func() {
			fmt.Fprintln(ui.OutWriter, strings.TrimSpace(res.TOON))
		})
	},
}

// ── Catalog Soft-Delete Trash & Undrop Commands ──────────────────────────────

var catalogTrashCmd = &cobra.Command{
	Use:   "trash",
	Short: "Manage soft-deleted catalogs, schemas, and tables in the workspace trash",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogTrashList()
	},
}

var catalogTrashListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List soft-deleted catalogs, schemas, and tables in the workspace trash",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogTrashList()
	},
}

func runCatalogTrashList() error {
	c := getClient()
	assets, err := c.ListCatalogTrash()
	if err != nil {
		return err
	}

	return ui.RenderOutput(assets, func() {
		if len(assets) == 0 {
			ui.PrintInfo("Workspace catalog trash is empty.")
			return
		}

		headers := []string{"ASSET ID", "TYPE", "NAME", "CATALOG.SCHEMA", "DROPPED AT", "TTL REMAINING"}
		rows := make([][]string, len(assets))
		for i, a := range assets {
			loc := a.CatalogName
			if a.SchemaName != "" {
				loc = fmt.Sprintf("%s.%s", a.CatalogName, a.SchemaName)
			}
			droppedStr := "—"
			if !a.DroppedAt.IsZero() {
				droppedStr = a.DroppedAt.Format("2006-01-02 15:04")
			}
			ttlStr := fmt.Sprintf("%.1f days", a.DaysRemaining)
			rows[i] = []string{
				a.ID,
				a.AssetType,
				a.Name,
				loc,
				droppedStr,
				ttlStr,
			}
		}

		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
		ui.PrintSuccess("Total soft-deleted assets in trash: %d", len(assets))
	})
}

func runCatalogTrashRestore(assetID string) error {
	c := getClient()
	var req *client.RestoreTrashAssetRequest
	if catRestoreStrat != "" || catRestoreNewName != "" {
		req = &client.RestoreTrashAssetRequest{
			Strategy: catRestoreStrat,
			NewName:  catRestoreNewName,
		}
	}

	res, err := c.RestoreCatalogTrashAsset(assetID, req)
	if err != nil {
		return err
	}

	return ui.RenderOutput(res, func() {
		ui.PrintSuccess("Soft-deleted asset '%s' restored from trash.", assetID)
	})
}

var catalogTrashRestoreCmd = &cobra.Command{
	Use:     "restore <asset_id>",
	Aliases: []string{"undrop"},
	Short:   "Restore a soft-deleted catalog, schema, or table from trash",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogTrashRestore(args[0])
	},
}

var catalogUndropCmd = &cobra.Command{
	Use:   "undrop <asset_id>",
	Short: "Restore a soft-deleted catalog, schema, or table from trash (alias for 'trash restore')",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCatalogTrashRestore(args[0])
	},
}

var catalogTrashPurgeCmd = &cobra.Command{
	Use:   "purge <asset_id>",
	Short: "Permanently delete a specific trashed asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]
		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			if !confirmAction(fmt.Sprintf("Permanently purge trashed asset '%s'? This cannot be undone.", assetID)) {
				ui.PrintWarning("Purge aborted.")
				return nil
			}
		}

		c := getClient()
		res, err := c.PurgeCatalogTrashAsset(assetID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Trashed asset '%s' permanently purged.", assetID)
		})
	},
}

var catalogTrashPurgeExpiredCmd = &cobra.Command{
	Use:   "purge-expired",
	Short: "Permanently purge all trashed assets past their retention TTL",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			if !confirmAction("Permanently purge all expired trashed assets?") {
				ui.PrintWarning("Purge expired aborted.")
				return nil
			}
		}

		c := getClient()
		res, err := c.PurgeExpiredCatalogTrash()
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Expired catalog trash purged.")
		})
	},
}

// ── User-Scoped Catalog Directory & Access Sharing Commands ──────────────────

var catalogDirectoryCmd = &cobra.Command{
	Use:   "directory",
	Short: "List all discoverable catalogs across the organization with caller access status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		items, err := c.ListCatalogDirectory(catWorkspace)
		if err != nil {
			return err
		}

		return ui.RenderOutput(items, func() {
			if len(items) == 0 {
				ui.PrintInfo("No external catalogs found in organization directory.")
				return
			}

			headers := []string{"CATALOG ID", "NAME", "OWNER WORKSPACE", "SCHEMAS", "ACCESS STATUS"}
			rows := make([][]string, len(items))
			for i, it := range items {
				owner := it.OwnerWorkspaceName
				if owner == "" {
					owner = it.OwnerWorkspaceID
				}
				rows[i] = []string{
					it.ID,
					it.Name,
					owner,
					fmt.Sprintf("%d", it.SchemaCount),
					it.AccessStatus,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Total directory catalogs: %d", len(items))
		})
	},
}

var catalogAccessCmd = &cobra.Command{
	Use:   "access",
	Short: "Manage user-scoped and cross-workspace catalog access requests and shares",
}

var catalogAccessListCmd = &cobra.Command{
	Use:     "list <catalog_id>",
	Aliases: []string{"ls"},
	Short:   "List inbound access requests and active user-scoped shares for a catalog",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		c := getClient()
		items, err := c.ListCatalogAccess(catalogID)
		if err != nil {
			return err
		}

		return ui.RenderOutput(items, func() {
			if len(items) == 0 {
				ui.PrintInfo("No access requests or shares found for catalog '%s'.", catalogID)
				return
			}

			headers := []string{"ACCESS ID", "WORKSPACE", "REQUESTED BY", "MODE", "STATUS", "REQUESTED AT"}
			rows := make([][]string, len(items))
			for i, it := range items {
				ws := it.WorkspaceName
				if ws == "" {
					ws = it.WorkspaceID
				}
				reqAt := "—"
				if !it.RequestedAt.IsZero() {
					reqAt = it.RequestedAt.Format("2006-01-02 15:04")
				}
				rows[i] = []string{
					it.ID,
					ws,
					it.RequestedByEmail,
					it.Mode,
					it.Status,
					reqAt,
				}
			}

			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
			ui.PrintSuccess("Showing %d access entries.", len(items))
		})
	},
}

var catalogAccessRequestCmd = &cobra.Command{
	Use:   "request <catalog_id>",
	Short: "Request access to an external catalog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		c := getClient()
		res, err := c.RequestCatalogAccess(catalogID, client.RequestCatalogAccessRequest{
			Mode:        catAccessMode,
			Message:     catAccessMessage,
			WorkspaceID: catWorkspace,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Access request submitted for catalog '%s'.", catalogID)
		})
	},
}

var catalogAccessApproveCmd = &cobra.Command{
	Use:   "approve <catalog_id> <access_id>",
	Short: "Approve an inbound catalog access request",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		accessID := args[1]
		c := getClient()
		res, err := c.UpdateCatalogAccess(catalogID, accessID, "approved")
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Access request '%s' approved for catalog '%s'.", accessID, catalogID)
		})
	},
}

var catalogAccessRejectCmd = &cobra.Command{
	Use:   "reject <catalog_id> <access_id>",
	Short: "Reject an inbound catalog access request",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		accessID := args[1]
		c := getClient()
		res, err := c.UpdateCatalogAccess(catalogID, accessID, "rejected")
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Access request '%s' rejected for catalog '%s'.", accessID, catalogID)
		})
	},
}

var catalogAccessRevokeCmd = &cobra.Command{
	Use:   "revoke <catalog_id> <access_id>",
	Short: "Revoke a user-scoped catalog share",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogID := args[0]
		accessID := args[1]
		if !catForce && !assumeYes && !nonInteractive && !ui.IsNonInteractive() {
			if !confirmAction(fmt.Sprintf("Revoke access share '%s' on catalog '%s'?", accessID, catalogID)) {
				ui.PrintWarning("Revoke aborted.")
				return nil
			}
		}

		c := getClient()
		if err := c.RevokeCatalogAccess(catalogID, accessID); err != nil {
			return err
		}

		res := map[string]interface{}{"ok": true, "revoked_id": accessID}
		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Access share '%s' revoked from catalog '%s'.", accessID, catalogID)
		})
	},
}

func init() {
	// Catalog flags
	catalogCreateCmd.Flags().StringVarP(&catDescription, "description", "d", "", "Catalog description")
	catalogCreateCmd.Flags().StringVarP(&catWorkspace, "workspace", "w", "", "Owning workspace ID or slug")
	catalogUpdateCmd.Flags().StringVarP(&catDescription, "description", "d", "", "Updated catalog description")
	catalogUpdateCmd.Flags().StringVar(&catName, "name", "", "Updated catalog name")
	catalogDeleteCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Force delete without confirmation")
	catalogDeleteCmd.Flags().BoolVar(&catPurge, "purge", false, "Permanently purge catalog and bypass soft delete")

	// Schemas flags
	catalogSchemasCreateCmd.Flags().StringVarP(&catDescription, "description", "d", "", "Schema description")
	catalogSchemasUpdateCmd.Flags().StringVarP(&catDescription, "description", "d", "", "Updated schema description")
	catalogSchemasDeleteCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Force delete without confirmation")
	catalogSchemasDeleteCmd.Flags().BoolVar(&catCascade, "cascade", false, "Cascade delete all tables in schema")

	// Tables flags
	catalogTablesUpdateCmd.Flags().StringVarP(&catDescription, "description", "d", "", "Updated table description")
	catalogTablesUpdateCmd.Flags().StringVar(&catS3Path, "s3-path", "", "Updated S3 path pattern")
	catalogTablesDeleteCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Force delete without confirmation")

	// TOON flags
	catalogToonCmd.Flags().BoolVar(&catToonRaw, "raw", false, "Print raw TOON text without JSON wrapper")

	// Trash & Undrop flags
	for _, cmd := range []*cobra.Command{catalogTrashRestoreCmd, catalogUndropCmd} {
		cmd.Flags().StringVar(&catRestoreStrat, "strategy", "", "Conflict resolution strategy (rename or replace)")
		cmd.Flags().StringVar(&catRestoreNewName, "new-name", "", "New asset name when using --strategy rename")
	}
	catalogTrashPurgeCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Force permanent purge without confirmation")
	catalogTrashPurgeExpiredCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Force permanent purge of expired assets without confirmation")

	// Directory & Access flags
	catalogDirectoryCmd.Flags().StringVarP(&catWorkspace, "workspace", "w", "", "Caller workspace ID for directory lookup")
	catalogAccessRequestCmd.Flags().StringVarP(&catAccessMessage, "message", "m", "Requested via Angler CLI", "Reason message for requesting catalog access")
	catalogAccessRequestCmd.Flags().StringVar(&catAccessMode, "mode", "read", "Requested access mode (read or write)")
	catalogAccessRequestCmd.Flags().StringVarP(&catWorkspace, "workspace", "w", "", "Requesting workspace ID")
	catalogAccessRevokeCmd.Flags().BoolVarP(&catForce, "force", "f", false, "Revoke without confirmation")

	// Anomalies flags
	catalogAnomaliesDetectCmd.Flags().StringVar(&catAnomalySource, "source", "", "Data source filter")
	catalogAnomaliesDetectCmd.Flags().StringVar(&catAnomalyTable, "table", "", "Table slug filter")
	catalogAnomaliesDetectCmd.Flags().Float64Var(&catAnomalyContam, "contamination", 0.05, "Anomaly contamination threshold")

	// Register subcommands under catalogSchemasCmd
	catalogSchemasCmd.AddCommand(catalogSchemasListCmd)
	catalogSchemasCmd.AddCommand(catalogSchemasGetCmd)
	catalogSchemasCmd.AddCommand(catalogSchemasCreateCmd)
	catalogSchemasCmd.AddCommand(catalogSchemasUpdateCmd)
	catalogSchemasCmd.AddCommand(catalogSchemasDeleteCmd)

	// Register subcommands under catalogTablesCmd
	catalogTablesCmd.AddCommand(catalogTablesListCmd)
	catalogTablesCmd.AddCommand(catalogTablesGetCmd)
	catalogTablesCmd.AddCommand(catalogTablesUpdateCmd)
	catalogTablesCmd.AddCommand(catalogTablesDeleteCmd)

	// Register subcommands under catalogTrashCmd
	catalogTrashCmd.AddCommand(catalogTrashListCmd)
	catalogTrashCmd.AddCommand(catalogTrashRestoreCmd)
	catalogTrashCmd.AddCommand(catalogTrashPurgeCmd)
	catalogTrashCmd.AddCommand(catalogTrashPurgeExpiredCmd)

	// Register subcommands under catalogAccessCmd
	catalogAccessCmd.AddCommand(catalogAccessListCmd)
	catalogAccessCmd.AddCommand(catalogAccessRequestCmd)
	catalogAccessCmd.AddCommand(catalogAccessApproveCmd)
	catalogAccessCmd.AddCommand(catalogAccessRejectCmd)
	catalogAccessCmd.AddCommand(catalogAccessRevokeCmd)

	// Register subcommands under catalogAnomaliesCmd
	catalogAnomaliesCmd.AddCommand(catalogAnomaliesDetectCmd)

	// Register subcommands under catalogCmd
	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogGetCmd)
	catalogCmd.AddCommand(catalogCreateCmd)
	catalogCmd.AddCommand(catalogUpdateCmd)
	catalogCmd.AddCommand(catalogDeleteCmd)
	catalogCmd.AddCommand(catalogSchemasCmd)
	catalogCmd.AddCommand(catalogTablesCmd)
	catalogCmd.AddCommand(catalogToonCmd)
	catalogCmd.AddCommand(catalogTrashCmd)
	catalogCmd.AddCommand(catalogUndropCmd)
	catalogCmd.AddCommand(catalogDirectoryCmd)
	catalogCmd.AddCommand(catalogAccessCmd)
	catalogCmd.AddCommand(catalogManifestCmd)
	catalogCmd.AddCommand(catalogAnomaliesCmd)

	rootCmd.AddCommand(catalogCmd)
}
