package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/toon"
	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	bundleFileFlag     string
	bundleTargetFlag   string
	bundleFormatFlag   string
	bundleTemplateFlag string
	bundleLocalFlag    bool
	bundleDryRunFlag   bool
	bundlePurgeStorage bool
	bundleIDFlag       string
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"lab", "bundles"},
	Short:   "Develop, validate, deploy, and manage Lake Asset Bundles (LABs)",
	Long: `📦 Lake Asset Bundles (LABs) enable Infrastructure-as-Code for your lakehouse.
Manage catalogs, schemas, DuckDB pipelines, vector indexes, and dashboards
declaratively using AI-first TOON (lake.toon) or YAML manifests.

Examples:
  angler bundle init my-lakehouse --template medallion
  angler bundle validate --local
  angler bundle deploy -t prod --dry-run
  angler bundle deploy -t dev
  angler bundle run orders_etl -t dev
  angler bundle destroy -t dev --purge-storage -y
  angler bundle ls -t dev`,
}

// -----------------------------------------------------------------------------
// 1. angler bundle init
// -----------------------------------------------------------------------------

var bundleInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a new Lake Asset Bundle project",
	Long: `Initialize a new Lake Asset Bundle with starter catalogs, schemas, pipelines, and manifests.

Available templates:
  • python-etl (default): Starter DuckDB pipeline and managed tables
  • rag-search: Lakehouse with vector embeddings and semantic search
  • medallion: Full Bronze, Silver, Gold tiered architecture with transformation stages`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "my-lakehouse"
		if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
			name = strings.TrimSpace(args[0])
		}

		template := "python-etl"
		if bundleTemplateFlag != "" {
			template = strings.ToLower(strings.TrimSpace(bundleTemplateFlag))
		}

		format := "toon"
		if bundleFormatFlag != "" {
			format = strings.ToLower(strings.TrimSpace(bundleFormatFlag))
		}

		return runBundleInit(name, template, format)
	},
}

func runBundleInit(name, template, format string) error {
	var manifestFilename string
	var manifestContent string
	var sqlDir string = "sql"
	var starterSQLFilename string
	var starterSQLContent string

	switch template {
	case "rag-search", "rag":
		starterSQLFilename = "embed_documents.sql"
		starterSQLContent = `-- Semantic search transformation
INSERT INTO main_lake.gold.doc_embeddings
SELECT 
    doc_id,
    content,
    title,
    current_timestamp AS embedded_at
FROM main_lake.bronze.raw_documents;
`
		if format == "yaml" || format == "yml" {
			manifestFilename = "lake.yml"
			manifestContent = fmt.Sprintf(`bundle:
  name: %s
  version: 0.1.0
  description: RAG & semantic search lakehouse with vector indices

targets:
  dev:
    default: true
    workspace: analytics-dev
  prod:
    default: false
    workspace: analytics-prod

resources:
  catalogs:
    - name: main_lake
      description: Core analytics lakehouse
  schemas:
    - catalog: main_lake
      name: bronze
      tier: bronze
    - catalog: main_lake
      name: gold
      tier: gold
  tables:
    - name: raw_documents
      catalog: main_lake
      schema: bronze
      format: parquet
    - name: doc_embeddings
      catalog: main_lake
      schema: gold
      format: parquet
      vector_index:
        embedding_column: embedding
        dimension: 768
        metric: cosine
  pipelines:
    rag_indexer:
      schedule: "0 */6 * * *"
      tasks:
        - name: generate_embeddings
          type: duckdb_sql
          entrypoint: sql/%s
`, name, starterSQLFilename)
		} else {
			manifestFilename = "lake.toon"
			manifestContent = fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: RAG & semantic search lakehouse with vector indices

targets:
  dev:
    default: true
    workspace: analytics-dev
  prod:
    default: false
    workspace: analytics-prod

resources:
  catalogs[1]{name,description}:
    main_lake,Core analytics lakehouse

  schemas[2]{catalog,name,tier}:
    main_lake,bronze,bronze
    main_lake,gold,gold

  tables[2]{name,catalog,schema,format}:
    raw_documents,main_lake,bronze,parquet
    doc_embeddings,main_lake,gold,parquet

  pipelines:
    rag_indexer:
      schedule: "0 */6 * * *"
      tasks[1]{name,type,entrypoint,upstream_tasks}:
        generate_embeddings,duckdb_sql,sql/%s,[]
`, name, starterSQLFilename)
		}

	case "medallion", "dbt-duckdb":
		starterSQLFilename = "orders_mart.sql"
		starterSQLContent = `-- Medallion Gold aggregation mart
INSERT INTO main_lake.gold.daily_revenue
SELECT 
    order_date,
    count(*) AS total_orders,
    sum(amount) AS total_revenue
FROM main_lake.silver.orders
GROUP BY order_date;
`
		if format == "yaml" || format == "yml" {
			manifestFilename = "lake.yml"
			manifestContent = fmt.Sprintf(`bundle:
  name: %s
  version: 0.1.0
  description: Enterprise Medallion Lakehouse (Bronze, Silver, Gold)

targets:
  dev:
    default: true
    workspace: enterprise-dev
  prod:
    default: false
    workspace: enterprise-prod

resources:
  catalogs:
    - name: main_lake
      description: Production lakehouse catalog
  schemas:
    - catalog: main_lake
      name: bronze
      tier: bronze
    - catalog: main_lake
      name: silver
      tier: silver
    - catalog: main_lake
      name: gold
      tier: gold
  tables:
    - name: raw_orders
      catalog: main_lake
      schema: bronze
      format: parquet
    - name: orders
      catalog: main_lake
      schema: silver
      format: parquet
    - name: daily_revenue
      catalog: main_lake
      schema: gold
      format: parquet
  pipelines:
    medallion_etl:
      schedule: "0 2 * * *"
      tasks:
        - name: build_gold_revenue
          type: duckdb_sql
          entrypoint: sql/%s
  dashboards:
    revenue_kpi:
      title: Executive Revenue KPI
      widgets:
        total_revenue:
          type: metric
          query: "SELECT sum(total_revenue) FROM main_lake.gold.daily_revenue"
`, name, starterSQLFilename)
		} else {
			manifestFilename = "lake.toon"
			manifestContent = fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: Enterprise Medallion Lakehouse (Bronze, Silver, Gold)

targets:
  dev:
    default: true
    workspace: enterprise-dev
  prod:
    default: false
    workspace: enterprise-prod

resources:
  catalogs[1]{name,description}:
    main_lake,Production lakehouse catalog

  schemas[3]{catalog,name,tier,description}:
    main_lake,bronze,bronze,Raw ingested events
    main_lake,silver,silver,Cleaned dimensional models
    main_lake,gold,gold,Aggregated revenue marts

  tables[3]{name,catalog,schema,format}:
    raw_orders,main_lake,bronze,parquet
    orders,main_lake,silver,parquet
    daily_revenue,main_lake,gold,parquet

  pipelines:
    medallion_etl:
      schedule: "0 2 * * *"
      tasks[1]{name,type,entrypoint,upstream_tasks}:
        build_gold_revenue,duckdb_sql,sql/%s,[]

  dashboards:
    revenue_kpi:
      title: Executive Revenue KPI
      widgets:
        total_revenue:
          type: metric
          query: "SELECT sum(total_revenue) FROM main_lake.gold.daily_revenue"
`, name, starterSQLFilename)
		}

	default: // python-etl
		starterSQLFilename = "clean_orders.sql"
		starterSQLContent = `-- Clean bronze orders into silver
INSERT INTO main_lake.silver.orders
SELECT 
    id,
    customer_id,
    amount,
    current_timestamp AS processed_at
FROM main_lake.bronze.raw_orders
WHERE amount > 0;
`
		if format == "yaml" || format == "yml" {
			manifestFilename = "lake.yml"
			manifestContent = fmt.Sprintf(`bundle:
  name: %s
  version: 0.1.0
  description: DuckDB Lakehouse ETL Pipeline

targets:
  dev:
    default: true
    workspace: dev-workspace
  prod:
    default: false
    workspace: prod-workspace

resources:
  catalogs:
    - name: main_lake
      description: Analytics lakehouse
  schemas:
    - catalog: main_lake
      name: bronze
      tier: bronze
    - catalog: main_lake
      name: silver
      tier: silver
  tables:
    - name: raw_orders
      catalog: main_lake
      schema: bronze
      format: parquet
    - name: orders
      catalog: main_lake
      schema: silver
      format: parquet
  pipelines:
    orders_pipeline:
      schedule: "0 4 * * *"
      tasks:
        - name: clean_orders
          type: duckdb_sql
          entrypoint: sql/%s
`, name, starterSQLFilename)
		} else {
			manifestFilename = "lake.toon"
			manifestContent = fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: DuckDB Lakehouse ETL Pipeline

targets:
  dev:
    default: true
    workspace: dev-workspace
  prod:
    default: false
    workspace: prod-workspace

resources:
  catalogs[1]{name,description}:
    main_lake,Analytics lakehouse

  schemas[2]{catalog,name,tier}:
    main_lake,bronze,bronze
    main_lake,silver,silver

  tables[2]{name,catalog,schema,format}:
    raw_orders,main_lake,bronze,parquet
    orders,main_lake,silver,parquet

  pipelines:
    orders_pipeline:
      schedule: "0 4 * * *"
      tasks[1]{name,type,entrypoint,upstream_tasks}:
        clean_orders,duckdb_sql,sql/%s,[]
`, name, starterSQLFilename)
		}
	}

	// Write manifest file
	if _, err := os.Stat(manifestFilename); err == nil {
		return fmt.Errorf("file '%s' already exists in current directory", manifestFilename)
	}

	if err := os.WriteFile(manifestFilename, []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", manifestFilename, err)
	}

	// Create directories and starter script
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", sqlDir, err)
	}
	_ = os.MkdirAll("notebooks", 0755)

	sqlFilePath := filepath.Join(sqlDir, starterSQLFilename)
	if err := os.WriteFile(sqlFilePath, []byte(starterSQLContent), 0644); err != nil {
		return fmt.Errorf("failed to write starter script %s: %w", sqlFilePath, err)
	}

	ui.PrintSuccess("Initialized Lake Asset Bundle '%s' using template '%s'", name, template)
	fmt.Fprintf(ui.OutWriter, "  Created manifest:  %s (%s)\n", manifestFilename, strings.ToUpper(format))
	fmt.Fprintf(ui.OutWriter, "  Created script:    %s\n", sqlFilePath)
	fmt.Fprintf(ui.OutWriter, "  Created directory: notebooks/\n\n")
	fmt.Fprintf(ui.OutWriter, "Next steps:\n")
	fmt.Fprintf(ui.OutWriter, "  1. Edit %s to customize your lakehouse resources\n", manifestFilename)
	fmt.Fprintf(ui.OutWriter, "  2. Validate your bundle:  angler bundle validate --local\n")
	fmt.Fprintf(ui.OutWriter, "  3. Deploy to dev target:  angler bundle deploy -t dev\n")

	return nil
}

// -----------------------------------------------------------------------------
// 2. angler bundle validate
// -----------------------------------------------------------------------------

var bundleValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate bundle manifest syntax, variable bindings, and DAG cycles",
	Long: `Validate a Lake Asset Bundle manifest against syntax requirements, variable interpolation,
pipeline DAG acyclicity, and remote workspace constraints.

Examples:
  angler bundle validate --local
  angler bundle validate -t dev
  angler bundle validate -f custom.toon -t prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, format, err := findManifestFile(bundleFileFlag)
		if err != nil {
			return err
		}

		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read manifest file '%s': %w", filePath, err)
		}

		if bundleLocalFlag {
			return runLocalValidation(filePath, format, rawBytes)
		}

		c := getClient()
		res, err := c.ValidateBundle(&client.BundleValidateRequest{
			ManifestContent: string(rawBytes),
			ManifestFormat:  format,
			Target:          bundleTargetFlag,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			if !res.Valid {
				ui.PrintError("Bundle validation failed for target '%s':", res.Target)
				for _, e := range res.Errors {
					fmt.Fprintf(ui.OutWriter, "  • %s\n", e)
				}
				os.Exit(ui.ExitGeneralError)
				return
			}

			ui.PrintSuccess("Bundle '%s' is valid for target '%s'", res.BundleName, res.Target)
			if res.DagValidated {
				fmt.Fprintln(ui.OutWriter, "  ✓ Pipeline DAG topology verified: No cyclic dependencies detected")
			}

			if res.Diff != nil {
				renderDiffTable(res.Diff)
			}
		})
	},
}

func runLocalValidation(filePath, format string, raw []byte) error {
	var parsed map[string]interface{}
	var err error

	if format == "toon" {
		err = toon.Unmarshal(raw, &parsed)
	} else if format == "yaml" || format == "yml" {
		err = yaml.Unmarshal(raw, &parsed)
	} else {
		err = json.Unmarshal(raw, &parsed)
	}

	if err != nil {
		ui.PrintError("Local syntax validation failed for '%s': %v", filePath, err)
		return err
	}

	// Verify required sections
	bundleSec, hasBundle := parsed["bundle"].(map[string]interface{})
	if !hasBundle || bundleSec["name"] == nil {
		err := fmt.Errorf("manifest missing required 'bundle.name' field")
		ui.PrintError("Structural error: %v", err)
		return err
	}

	bundleName := fmt.Sprintf("%v", bundleSec["name"])
	version := "0.1.0"
	if bundleSec["version"] != nil {
		version = fmt.Sprintf("%v", bundleSec["version"])
	}

	ui.PrintSuccess("Local manifest '%s' is syntactically valid!", filePath)
	fmt.Fprintf(ui.OutWriter, "  Bundle:  %s (v%s)\n", bundleName, version)
	fmt.Fprintf(ui.OutWriter, "  Format:  %s\n", strings.ToUpper(format))
	return nil
}

// -----------------------------------------------------------------------------
// 3. angler bundle deploy
// -----------------------------------------------------------------------------

var bundleDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Lake Asset Bundle resources and pipelines to the active lakehouse",
	Long: `Deploys catalogs, schemas, DuckDB managed tables, vector indices, Airflow DAGs,
and dashboards to the target environment.

Examples:
  angler bundle deploy -t dev
  angler bundle deploy -t prod --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, format, err := findManifestFile(bundleFileFlag)
		if err != nil {
			return err
		}

		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read manifest file '%s': %w", filePath, err)
		}

		// Collect local artifact files (sql/, notebooks/)
		filesMap := collectLocalFiles(filepath.Dir(filePath))

		c := getClient()
		res, err := c.DeployBundle(&client.BundleDeployRequest{
			ManifestContent: string(rawBytes),
			ManifestFormat:  format,
			Target:          bundleTargetFlag,
			DryRun:          bundleDryRunFlag,
			Files:           filesMap,
		})
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			if res.DryRun {
				ui.PrintWarning("DRY RUN: Deployment simulation completed in %.1fms", res.ExecutionDurationMs)
				fmt.Fprintf(ui.OutWriter, "Target: %s | Status: %s | Version: v%s\n\n", res.Target, res.Status, res.Version)
				if res.Diff != nil {
					renderDiffTable(res.Diff)
				}
				return
			}

			ui.PrintSuccess("Bundle '%s' deployed successfully to target '%s' in %.1fms", res.BundleName, res.Target, res.ExecutionDurationMs)
			fmt.Fprintf(ui.OutWriter, "  Bundle ID: %s\n", res.BundleID)
			fmt.Fprintf(ui.OutWriter, "  Status:    %s\n", res.Status)
			fmt.Fprintf(ui.OutWriter, "  Version:   v%s\n", res.Version)
			if len(res.ProvisionedCounts) > 0 {
				fmt.Fprintf(ui.OutWriter, "  Provisioned:\n")
				for k, count := range res.ProvisionedCounts {
					fmt.Fprintf(ui.OutWriter, "    • %s: %d\n", k, count)
				}
			}
			if res.Diff != nil {
				renderDiffTable(res.Diff)
			}
		})
	},
}

// -----------------------------------------------------------------------------
// 4. angler bundle run
// -----------------------------------------------------------------------------

var bundleRunCmd = &cobra.Command{
	Use:   "run [pipeline]",
	Short: "Trigger ad-hoc execution of a bundle pipeline or DAG",
	Long: `Execute an Airflow pipeline defined within the deployed bundle.

Examples:
  angler bundle run orders_etl -t dev
  angler bundle run --bundle-id 96fd3d6c-2c6e-4b81-8198-453ea28a085a`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineName := ""
		if len(args) > 0 {
			pipelineName = args[0]
		}

		c := getClient()
		bundleID, err := resolveBundleID(c, bundleIDFlag, bundleTargetFlag)
		if err != nil {
			return err
		}

		res, err := c.RunBundle(bundleID, pipelineName)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Pipeline triggered successfully!")
			fmt.Fprintf(ui.OutWriter, "  Bundle:    %s\n", res.BundleName)
			fmt.Fprintf(ui.OutWriter, "  Pipeline:  %s\n", res.PipelineName)
			fmt.Fprintf(ui.OutWriter, "  DAG ID:    %s\n", res.DAGID)
			fmt.Fprintf(ui.OutWriter, "  Run ID:    %s\n", res.DAGRunID)
			fmt.Fprintf(ui.OutWriter, "  Status:    %s\n", res.Status)
		})
	},
}

// -----------------------------------------------------------------------------
// 5. angler bundle destroy
// -----------------------------------------------------------------------------

var bundleDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Tear down and unprovision bundle resources in the target workspace",
	Long: `Permanently deletes provisioned tables, pipelines, and dashboards managed by the bundle.
Core medallion schemas (bronze, silver, gold) remain protected.

Examples:
  angler bundle destroy -t dev -y
  angler bundle destroy -t dev --purge-storage -y`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		bundleID, err := resolveBundleID(c, bundleIDFlag, bundleTargetFlag)
		if err != nil {
			return err
		}

		if !assumeYes && !ui.IsNonInteractive() {
			fmt.Fprintf(ui.OutWriter, "⚠️  Are you sure you want to destroy bundle %s in target '%s'?\n", bundleID, bundleTargetFlag)
			if bundlePurgeStorage {
				fmt.Fprintln(ui.OutWriter, "   Storage purge is ENABLED: Raw Parquet data and vector embeddings will be permanently removed.")
			}
			fmt.Fprint(ui.OutWriter, "Type 'yes' to confirm: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(input)) != "yes" {
				ui.PrintInfo("Destruction cancelled.")
				return nil
			}
		}

		res, err := c.DestroyBundle(bundleID, true, bundlePurgeStorage)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Bundle '%s' destroyed successfully.", res.BundleID)
			if bundlePurgeStorage {
				fmt.Fprintln(ui.OutWriter, "  Storage artifacts purged.")
			}
		})
	},
}

// -----------------------------------------------------------------------------
// 6. angler bundle ls
// -----------------------------------------------------------------------------

var bundleListCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List deployed Lake Asset Bundles in the active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()
		res, err := c.ListBundles(bundleTargetFlag, "")
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			if len(res.Bundles) == 0 {
				ui.PrintInfo("No Lake Asset Bundles deployed in the active workspace.")
				return
			}

			tableRows := make([][]string, len(res.Bundles))
			for i, b := range res.Bundles {
				deployedAt := b.LastDeployedAt
				if deployedAt == "" {
					deployedAt = b.CreatedAt
				}
				if len(deployedAt) > 19 {
					deployedAt = deployedAt[:19]
				}

				tableRows[i] = []string{
					b.BundleName,
					b.Target,
					b.Status,
					"v" + b.Version,
					fmt.Sprintf("%d", b.CatalogsCount),
					fmt.Sprintf("%d", b.TablesCount),
					fmt.Sprintf("%d", b.PipelinesCount),
					fmt.Sprintf("%d", b.DashboardsCount),
					deployedAt,
				}
			}

			headers := []string{"Bundle Name", "Target", "Status", "Version", "Catalogs", "Tables", "Pipelines", "Dashboards", "Last Deployed"}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
			ui.PrintSuccess("Total deployed bundles: %d", res.Total)
		})
	},
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func findManifestFile(explicitPath string) (path string, format string, err error) {
	if explicitPath != "" {
		if _, statErr := os.Stat(explicitPath); statErr != nil {
			return "", "", fmt.Errorf("specified bundle manifest '%s' not found: %w", explicitPath, statErr)
		}
		ext := strings.ToLower(filepath.Ext(explicitPath))
		switch ext {
		case ".toon":
			return explicitPath, "toon", nil
		case ".yml", ".yaml":
			return explicitPath, "yaml", nil
		case ".json":
			return explicitPath, "json", nil
		default:
			return explicitPath, "toon", nil
		}
	}

	candidates := []struct {
		filename string
		format   string
	}{
		{"lake.toon", "toon"},
		{"lake.yml", "yaml"},
		{"lake.yaml", "yaml"},
		{"lake.json", "json"},
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.filename); err == nil {
			return c.filename, c.format, nil
		}
	}

	return "", "", fmt.Errorf("no bundle manifest found in current directory (looked for lake.toon, lake.yml, lake.yaml, lake.json).\nRun 'angler bundle init' to scaffold a new bundle")
}

func collectLocalFiles(baseDir string) map[string]string {
	files := make(map[string]string)
	if baseDir == "" {
		baseDir = "."
	}

	subdirs := []string{"sql", "notebooks"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(baseDir, subdir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			// Skip hidden or non-script files
			name := d.Name()
			if strings.HasPrefix(name, ".") {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".sql" && ext != ".py" && ext != ".ipynb" && ext != ".sh" && ext != ".json" {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr == nil && len(content) < 1024*1024 { // Max 1MB per artifact
				rel, _ := filepath.Rel(baseDir, path)
				files[filepath.ToSlash(rel)] = string(content)
			}
			return nil
		})
	}
	return files
}

func resolveBundleID(c *client.Client, explicitID, target string) (string, error) {
	if explicitID != "" {
		return explicitID, nil
	}

	// Try reading local manifest for bundle name
	filePath, _, err := findManifestFile("")
	bundleName := ""
	if err == nil {
		raw, readErr := os.ReadFile(filePath)
		if readErr == nil {
			var parsed map[string]interface{}
			_ = toon.Unmarshal(raw, &parsed)
			if bSec, ok := parsed["bundle"].(map[string]interface{}); ok && bSec["name"] != nil {
				bundleName = fmt.Sprintf("%v", bSec["name"])
			}
		}
	}

	// Query server for bundles matching target & name
	res, listErr := c.ListBundles(target, "")
	if listErr != nil {
		return "", fmt.Errorf("failed to resolve bundle ID: %w", listErr)
	}

	for _, b := range res.Bundles {
		if (bundleName == "" || b.BundleName == bundleName) && (target == "" || b.Target == target) {
			return b.ID, nil
		}
	}

	if len(res.Bundles) == 1 {
		return res.Bundles[0].ID, nil
	}

	if len(res.Bundles) > 1 {
		return "", fmt.Errorf("multiple bundles deployed in workspace; specify --bundle-id <uuid>")
	}

	return "", fmt.Errorf("no matching bundle found in target '%s'. Specify --bundle-id <uuid>", target)
}

func renderDiffTable(diff *client.BundleValidationDiff) {
	if diff == nil {
		return
	}

	headers := []string{"Action", "Resource Identifier"}
	var rows [][]string

	for _, r := range diff.ToCreate {
		rows = append(rows, []string{"+ CREATE", r})
	}
	for _, r := range diff.ToUpdate {
		rows = append(rows, []string{"~ UPDATE", r})
	}
	for _, r := range diff.ToDelete {
		rows = append(rows, []string{"- DELETE", r})
	}
	for _, r := range diff.Unchanged {
		rows = append(rows, []string{"= UNCHANGED", r})
	}

	if len(rows) > 0 {
		fmt.Fprintf(ui.OutWriter, "\nReconciliation Diff Plan:\n")
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
	}
}

func init() {
	// Persistent and local flags
	bundleCmd.PersistentFlags().StringVar(&bundleTargetFlag, "target", "", "Target environment (e.g. dev, staging, prod)")
	bundleCmd.PersistentFlags().StringVarP(&bundleFileFlag, "file", "f", "", "Path to bundle manifest file (lake.toon, lake.yml)")

	// init flags
	bundleInitCmd.Flags().StringVar(&bundleTemplateFlag, "template", "python-etl", "Template (python-etl, rag-search, medallion)")
	bundleInitCmd.Flags().StringVar(&bundleFormatFlag, "format", "toon", "Manifest format: toon or yaml")

	// validate flags
	bundleValidateCmd.Flags().BoolVar(&bundleLocalFlag, "local", false, "Validate manifest syntax offline without contacting server")

	// deploy flags
	bundleDeployCmd.Flags().BoolVar(&bundleDryRunFlag, "dry-run", false, "Simulate deployment and output diff without persisting changes")

	// run flags
	bundleRunCmd.Flags().StringVar(&bundleIDFlag, "bundle-id", "", "UUID of the deployed bundle")

	// destroy flags
	bundleDestroyCmd.Flags().StringVar(&bundleIDFlag, "bundle-id", "", "UUID of the deployed bundle")
	bundleDestroyCmd.Flags().BoolVar(&bundlePurgeStorage, "purge-storage", false, "Also permanently purge raw Parquet files and vector embeddings in object storage")

	// Add subcommands
	bundleCmd.AddCommand(bundleInitCmd)
	bundleCmd.AddCommand(bundleValidateCmd)
	bundleCmd.AddCommand(bundleDeployCmd)
	bundleCmd.AddCommand(bundleRunCmd)
	bundleCmd.AddCommand(bundleDestroyCmd)
	bundleCmd.AddCommand(bundleListCmd)

	rootCmd.AddCommand(bundleCmd)
}
