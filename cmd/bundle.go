package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
	bundleNoDriftFlag  bool
	bundleManifestFlag bool
	bundleConfFlag     string
	bundleWaitFlag     bool
	bundleTimeoutFlag  time.Duration
	bundlePollInterval time.Duration
	bundleOutFlag      string
	bundleDeployFlag   bool
	bundlePromptFlag   string
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"lab", "bundles"},
	Short:   "Develop, validate, deploy, and manage Lake Asset Bundles (LABs)",
	Long: `📦 Lake Asset Bundles (LABs) enable Infrastructure-as-Code for your lakehouse.
Manage catalogs, schemas, DuckDB & Python virtualenv pipelines, webhook triggers,
vector indexes, and dashboards declaratively using AI-first TOON (lake.toon) or YAML manifests.

Examples:
  angler bundle init my-lakehouse --template medallion
  angler bundle init my-webhook-etl --template webhook-etl
  angler bundle validate --local
  angler bundle deploy -T prod --dry-run
  angler bundle deploy -T dev
  angler bundle get my-lakehouse -T dev
  angler bundle run orders_etl -T dev --conf '{"batch_date":"2026-11-10"}' --wait
  angler bundle generate "Build a biomedical PubMed medallion lakehouse with RAG embeddings" --out lake.toon
  angler bundle destroy -T dev --purge-storage -y
  angler bundle ls -T dev`,
}

// -----------------------------------------------------------------------------
// 1. angler bundle init
// -----------------------------------------------------------------------------

var bundleInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a new Lake Asset Bundle project",
	Long: `Initialize a new Lake Asset Bundle with starter catalogs, schemas, pipelines, and manifests.

Available templates:
  • python-etl (default): Starter Python virtualenv + DuckDB pipeline and managed tables
  • webhook-etl: Inbound @webhook-triggered Python + DuckDB medallion pipeline
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
	sqlDir := "sql"
	var starterSQLFilename string
	var starterSQLContent string
	var starterPyFilename string
	var starterPyContent string

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

	case "webhook-etl", "webhook":
		starterPyFilename = "process_webhook.py"
		starterPyContent = `"""Inbound webhook payload processor for Lake Asset Bundle."""
import json
import os

def main():
    payload = os.environ.get("LAKE_WEBHOOK_PAYLOAD", "{}")
    data = json.loads(payload) if payload else {}
    print(f"Processed inbound webhook event payload keys: {list(data.keys())}")

if __name__ == "__main__":
    main()
`
		starterSQLFilename = "transform_webhook_events.sql"
		starterSQLContent = `-- Transform raw webhook events from Bronze into Silver
INSERT INTO main_lake.silver.webhook_events
SELECT
    event_id,
    event_type,
    payload_json,
    current_timestamp AS processed_at
FROM main_lake.bronze.raw_webhooks;
`
		if format == "yaml" || format == "yml" {
			manifestFilename = "lake.yml"
			manifestContent = fmt.Sprintf(`bundle:
  name: %s
  version: 0.1.0
  description: Event-driven @webhook Medallion Pipeline with Python virtualenv + DuckDB

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
      description: Event-driven analytics lakehouse
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
    - name: raw_webhooks
      catalog: main_lake
      schema: bronze
      format: parquet
    - name: webhook_events
      catalog: main_lake
      schema: silver
      format: parquet
  pipelines:
    webhook_ingest_etl:
      schedule: "@webhook"
      webhook_enabled: true
      tasks:
        - name: parse_webhook_payload
          type: python_script
          entrypoint: scripts/%s
          requirements:
            - "pandas>=2.0"
            - "httpx"
        - name: transform_silver_events
          type: duckdb_sql
          entrypoint: sql/%s
          upstream_tasks:
            - parse_webhook_payload
`, name, starterPyFilename, starterSQLFilename)
		} else {
			manifestFilename = "lake.toon"
			manifestContent = fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: Event-driven @webhook Medallion Pipeline with Python virtualenv + DuckDB

targets:
  dev:
    default: true
    workspace: dev-workspace
  prod:
    default: false
    workspace: prod-workspace

resources:
  catalogs[1]{name,description}:
    main_lake,Event-driven analytics lakehouse

  schemas[3]{catalog,name,tier}:
    main_lake,bronze,bronze
    main_lake,silver,silver
    main_lake,gold,gold

  tables[2]{name,catalog,schema,format}:
    raw_webhooks,main_lake,bronze,parquet
    webhook_events,main_lake,silver,parquet

  pipelines:
    webhook_ingest_etl:
      schedule: "@webhook"
      webhook_enabled: true
      tasks[2]{name,type,entrypoint,requirements,upstream_tasks}:
        parse_webhook_payload,python_script,scripts/%s,"[pandas>=2.0,httpx]",[]
        transform_silver_events,duckdb_sql,sql/%s,"",[parse_webhook_payload]
`, name, starterPyFilename, starterSQLFilename)
		}

	default: // python-etl
		starterPyFilename = "ingest_orders.py"
		starterPyContent = `"""Python ETL task executed in an isolated cached virtualenv."""
import os

def main():
    print("Ingesting bronze orders batch into Lake of Tears...")

if __name__ == "__main__":
    main()
`
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
  description: DuckDB & Python Lakehouse ETL Pipeline

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
        - name: ingest_orders
          type: python_script
          entrypoint: scripts/%s
          requirements:
            - "pandas>=2.0"
            - "httpx"
        - name: clean_orders
          type: duckdb_sql
          entrypoint: sql/%s
          upstream_tasks:
            - ingest_orders
`, name, starterPyFilename, starterSQLFilename)
		} else {
			manifestFilename = "lake.toon"
			manifestContent = fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: DuckDB & Python Lakehouse ETL Pipeline

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
      tasks[2]{name,type,entrypoint,requirements,upstream_tasks}:
        ingest_orders,python_script,scripts/%s,"[pandas>=2.0,httpx]",[]
        clean_orders,duckdb_sql,sql/%s,"",[ingest_orders]
`, name, starterPyFilename, starterSQLFilename)
		}
	}

	// Write manifest file
	if _, err := os.Stat(manifestFilename); err == nil {
		return fmt.Errorf("file '%s' already exists in current directory", manifestFilename)
	}

	if err := os.WriteFile(manifestFilename, []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", manifestFilename, err)
	}

	// Create directories and starter scripts
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", sqlDir, err)
	}
	_ = os.MkdirAll("notebooks", 0755)
	_ = os.MkdirAll("scripts", 0755)

	sqlFilePath := filepath.Join(sqlDir, starterSQLFilename)
	if err := os.WriteFile(sqlFilePath, []byte(starterSQLContent), 0644); err != nil {
		return fmt.Errorf("failed to write starter script %s: %w", sqlFilePath, err)
	}

	var pyFilePath string
	if starterPyFilename != "" {
		pyFilePath = filepath.Join("scripts", starterPyFilename)
		if err := os.WriteFile(pyFilePath, []byte(starterPyContent), 0644); err != nil {
			return fmt.Errorf("failed to write starter python script %s: %w", pyFilePath, err)
		}
	}

	ui.PrintSuccess("Initialized Lake Asset Bundle '%s' using template '%s'", name, template)
	fmt.Fprintf(ui.OutWriter, "  Created manifest:  %s (%s)\n", manifestFilename, strings.ToUpper(format))
	fmt.Fprintf(ui.OutWriter, "  Created script:    %s\n", sqlFilePath)
	if pyFilePath != "" {
		fmt.Fprintf(ui.OutWriter, "  Created script:    %s\n", pyFilePath)
	}
	fmt.Fprintf(ui.OutWriter, "  Created directory: notebooks/, scripts/\n\n")
	fmt.Fprintf(ui.OutWriter, "Next steps:\n")
	fmt.Fprintf(ui.OutWriter, "  1. Edit %s to customize your lakehouse resources\n", manifestFilename)
	fmt.Fprintf(ui.OutWriter, "  2. Validate your bundle:  angler bundle validate --local\n")
	fmt.Fprintf(ui.OutWriter, "  3. Deploy to dev target:  angler bundle deploy -T dev\n")

	return nil
}

// -----------------------------------------------------------------------------
// 2. angler bundle validate
// -----------------------------------------------------------------------------

var bundleValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate bundle manifest syntax, variable bindings, and DAG cycles",
	Long: `Validate a Lake Asset Bundle manifest against syntax requirements, variable interpolation,
pipeline DAG acyclicity, task virtualenv requirements, @webhook schedules, and remote workspace constraints.

Examples:
  angler bundle validate --local
  angler bundle validate -T dev
  angler bundle validate -f custom.toon -T prod`,
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
	parsed, err := unmarshalManifestBytes(format, raw)
	if err != nil {
		ui.PrintError("Local syntax validation failed for '%s': %v", filePath, err)
		return err
	}

	// Verify required sections
	bundleSec, hasBundle := parsed["bundle"].(map[string]interface{})
	if !hasBundle || bundleSec["name"] == nil || strings.TrimSpace(fmt.Sprintf("%v", bundleSec["name"])) == "" {
		err := fmt.Errorf("manifest missing required 'bundle.name' field")
		ui.PrintError("Structural error: %v", err)
		return err
	}

	bundleName := fmt.Sprintf("%v", bundleSec["name"])
	version := "0.1.0"
	if bundleSec["version"] != nil {
		version = fmt.Sprintf("%v", bundleSec["version"])
	}

	// Validate pipeline schedules (@webhook / cron), task types, requirements, and DAG cycles
	webhookPipelines := 0
	pythonVenvTasks := 0
	if resSec, ok := parsed["resources"].(map[string]interface{}); ok {
		if rawPipelines, hasPipes := resSec["pipelines"]; hasPipes && rawPipelines != nil {
			wCount, pyCount, pipeErr := validateLocalPipelines(rawPipelines)
			if pipeErr != nil {
				ui.PrintError("Pipeline validation failed: %v", pipeErr)
				return pipeErr
			}
			webhookPipelines = wCount
			pythonVenvTasks = pyCount
		}
	}

	ui.PrintSuccess("Local manifest '%s' is syntactically valid!", filePath)
	fmt.Fprintf(ui.OutWriter, "  Bundle:  %s (v%s)\n", bundleName, version)
	fmt.Fprintf(ui.OutWriter, "  Format:  %s\n", strings.ToUpper(format))
	if webhookPipelines > 0 {
		fmt.Fprintf(ui.OutWriter, "  Webhook Pipelines: %d (@webhook trigger enabled)\n", webhookPipelines)
	}
	if pythonVenvTasks > 0 {
		fmt.Fprintf(ui.OutWriter, "  Python Virtualenv Tasks: %d\n", pythonVenvTasks)
	}
	return nil
}

func validateLocalPipelines(rawPipelines interface{}) (webhookCount int, pyVenvCount int, err error) {
	pipeMaps := make(map[string]map[string]interface{})

	switch pTyped := rawPipelines.(type) {
	case map[string]interface{}:
		for k, v := range pTyped {
			if m, ok := v.(map[string]interface{}); ok {
				pipeMaps[k] = m
			}
		}
	case []interface{}:
		for i, item := range pTyped {
			if m, ok := item.(map[string]interface{}); ok {
				name := fmt.Sprintf("pipeline_%d", i)
				if m["name"] != nil {
					name = fmt.Sprintf("%v", m["name"])
				}
				pipeMaps[name] = m
			}
		}
	}

	validTaskTypes := map[string]bool{
		"duckdb_sql":    true,
		"python_script": true,
		"notebook":      true,
		"bash":          true,
	}

	for pipeName, pipeDef := range pipeMaps {
		sched := ""
		if pipeDef["schedule"] != nil {
			sched = strings.TrimSpace(fmt.Sprintf("%v", pipeDef["schedule"]))
		}
		webhookEnabled, _ := pipeDef["webhook_enabled"].(bool)
		if strings.EqualFold(sched, "@webhook") || webhookEnabled {
			webhookCount++
		}

		var taskList []map[string]interface{}
		if rawTasks, ok := pipeDef["tasks"].([]interface{}); ok {
			for _, t := range rawTasks {
				if tm, ok := t.(map[string]interface{}); ok {
					taskList = append(taskList, tm)
				}
			}
		} else if tmList, ok := pipeDef["tasks"].([]map[string]interface{}); ok {
			taskList = tmList
		}

		adj := make(map[string][]string)
		knownTasks := make(map[string]bool)

		for _, t := range taskList {
			tName := strings.TrimSpace(fmt.Sprintf("%v", t["name"]))
			if tName == "" || t["name"] == nil {
				return webhookCount, pyVenvCount, fmt.Errorf("pipeline '%s' has a task missing required 'name'", pipeName)
			}
			knownTasks[tName] = true

			tType := "duckdb_sql"
			if t["type"] != nil && strings.TrimSpace(fmt.Sprintf("%v", t["type"])) != "" {
				tType = strings.TrimSpace(fmt.Sprintf("%v", t["type"]))
			}
			if !validTaskTypes[tType] {
				return webhookCount, pyVenvCount, fmt.Errorf("pipeline '%s' task '%s' has unsupported type '%s'", pipeName, tName, tType)
			}

			if rawReq, hasReq := t["requirements"]; hasReq && rawReq != nil {
				switch rVal := rawReq.(type) {
				case string:
					if strings.TrimSpace(rVal) != "" && strings.TrimSpace(rVal) != "[]" {
						pyVenvCount++
					}
				case []interface{}:
					if len(rVal) > 0 {
						pyVenvCount++
					}
				case []string:
					if len(rVal) > 0 {
						pyVenvCount++
					}
				default:
					return webhookCount, pyVenvCount, fmt.Errorf("pipeline '%s' task '%s' has invalid requirements format", pipeName, tName)
				}
			}

			upstreams := parseStringListField(t["upstream_tasks"])
			adj[tName] = upstreams
		}

		// Verify all upstream references exist and check for DAG cycles using 3-state DFS
		for tName, ups := range adj {
			for _, u := range ups {
				if !knownTasks[u] {
					return webhookCount, pyVenvCount, fmt.Errorf("pipeline '%s' task '%s' references unknown upstream task '%s'", pipeName, tName, u)
				}
			}
		}

		state := make(map[string]int) // 0 = unvisited, 1 = visiting, 2 = visited
		var hasCycle func(node string) bool
		hasCycle = func(node string) bool {
			state[node] = 1
			for _, dep := range adj[node] {
				if state[dep] == 1 {
					return true
				}
				if state[dep] == 0 && hasCycle(dep) {
					return true
				}
			}
			state[node] = 2
			return false
		}

		for tName := range knownTasks {
			if state[tName] == 0 {
				if hasCycle(tName) {
					return webhookCount, pyVenvCount, fmt.Errorf("cyclic dependency detected in pipeline '%s'", pipeName)
				}
			}
		}
	}

	return webhookCount, pyVenvCount, nil
}

func parseStringListField(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		var out []string
		for _, s := range v {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []interface{}:
		var out []string
		for _, item := range v {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", item)); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(v)
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		var out []string
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	return nil
}

// -----------------------------------------------------------------------------
// 3. angler bundle deploy
// -----------------------------------------------------------------------------

var bundleDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Lake Asset Bundle resources and pipelines to the active lakehouse",
	Long: `Deploys catalogs, schemas, DuckDB managed tables, vector indices, Python virtualenv Airflow DAGs,
webhook triggers, and Lakeset semantic datasets to the target environment.

Examples:
  angler bundle deploy -T dev
  angler bundle deploy -T prod --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, format, err := findManifestFile(bundleFileFlag)
		if err != nil {
			return err
		}

		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read manifest file '%s': %w", filePath, err)
		}

		// Collect local artifact files (sql/, notebooks/, scripts/, python/, src/)
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
			effectiveID := res.ID
			if effectiveID == "" {
				effectiveID = res.BundleID
			}

			if bundleDryRunFlag || res.DryRun || effectiveID == "" {
				ui.PrintWarning("DRY RUN: Deployment simulation completed")
				fmt.Fprintf(ui.OutWriter, "  Bundle:  %s\n", res.BundleName)
				fmt.Fprintf(ui.OutWriter, "  Target:  %s\n", res.Target)
				fmt.Fprintf(ui.OutWriter, "  Status:  %s\n", res.Status)
				if res.Message != "" {
					fmt.Fprintf(ui.OutWriter, "  Message: %s\n", res.Message)
				}
				if res.Diff != nil {
					renderDiffTable(res.Diff)
				}
				return
			}

			ui.PrintSuccess("Bundle '%s' deployed successfully to target '%s' in %.1fms", res.BundleName, res.Target, res.ExecutionDurationMs)
			fmt.Fprintf(ui.OutWriter, "  Bundle ID: %s\n", effectiveID)
			fmt.Fprintf(ui.OutWriter, "  Status:    %s\n", res.Status)
			if res.Version != "" {
				fmt.Fprintf(ui.OutWriter, "  Version:   v%s\n", res.Version)
			}
			if len(res.ResourcesMap) > 0 {
				fmt.Fprintf(ui.OutWriter, "  Provisioned Resources:\n")
				keys := make([]string, 0, len(res.ResourcesMap))
				for k := range res.ResourcesMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(ui.OutWriter, "    • %s: %d\n", k, len(res.ResourcesMap[k]))
				}
			} else if len(res.ProvisionedCounts) > 0 {
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
// 4. angler bundle get (inspect / status / drift)
// -----------------------------------------------------------------------------

var bundleGetCmd = &cobra.Command{
	Use:     "get [bundle_id_or_name]",
	Aliases: []string{"inspect", "status", "drift"},
	Short:   "Inspect a deployed Lake Asset Bundle and run live state drift detection",
	Long: `Retrieve detailed deployment metadata, resource inventory counts, provisioned UUID map,
and a live state drift report comparing the bundle manifest against active workspace resources.

Examples:
  angler bundle get
  angler bundle get customer-analytics -T dev
  angler bundle inspect 96fd3d6c-2c6e-4b81-8198-453ea28a085a --manifest
  angler bundle drift my-lakehouse --no-drift`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := getClient()

		idOrName := bundleIDFlag
		if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
			idOrName = strings.TrimSpace(args[0])
		}

		bundleID, err := resolveBundleIDOrName(c, idOrName, bundleTargetFlag)
		if err != nil {
			return err
		}

		res, err := c.GetBundleWithOptions(bundleID, !bundleNoDriftFlag)
		if err != nil {
			return err
		}

		return ui.RenderOutput(res, func() {
			ui.PrintSuccess("Lake Asset Bundle: %s (Target: %s)", res.BundleName, res.Target)
			fmt.Fprintf(ui.OutWriter, "  Bundle ID:      %s\n", res.ID)
			fmt.Fprintf(ui.OutWriter, "  Status:         %s\n", res.Status)
			if res.Version != "" {
				fmt.Fprintf(ui.OutWriter, "  Version:        v%s\n", strings.TrimPrefix(res.Version, "v"))
			}
			commitSHA := res.GitCommitSHA
			if commitSHA == "" {
				commitSHA = res.CommitHash
			}
			if commitSHA != "" {
				fmt.Fprintf(ui.OutWriter, "  Git Commit SHA: %s\n", commitSHA)
			}
			deployedAt := res.LastDeployedAt
			if deployedAt == "" {
				deployedAt = res.CreatedAt
			}
			if deployedAt != "" {
				fmt.Fprintf(ui.OutWriter, "  Last Deployed:  %s\n", deployedAt)
			}

			fmt.Fprintf(ui.OutWriter, "\n  Resource Inventory Counts:\n")
			fmt.Fprintf(ui.OutWriter, "    • Catalogs:   %d\n", res.CatalogsCount)
			fmt.Fprintf(ui.OutWriter, "    • Tables:     %d\n", res.TablesCount)
			fmt.Fprintf(ui.OutWriter, "    • Pipelines:  %d\n", res.PipelinesCount)
			fmt.Fprintf(ui.OutWriter, "    • Dashboards: %d\n", res.DashboardsCount)

			if len(res.ResourcesMap) > 0 {
				fmt.Fprintf(ui.OutWriter, "\n  Provisioned Resource Map:\n")
				keys := make([]string, 0, len(res.ResourcesMap))
				for k := range res.ResourcesMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(ui.OutWriter, "    • %s: %v\n", k, res.ResourcesMap[k])
				}
			}

			if !bundleNoDriftFlag && res.Drift != nil {
				renderDriftReport(res.Drift)
			}

			if bundleManifestFlag && strings.TrimSpace(res.ManifestRaw) != "" {
				fmt.Fprintf(ui.OutWriter, "\nStored Manifest Snapshot:\n%s\n", res.ManifestRaw)
			}
		})
	},
}

func renderDriftReport(drift map[string]interface{}) {
	fmt.Fprintf(ui.OutWriter, "\n  Live Drift Report:\n")
	inSync, _ := drift["in_sync"].(bool)
	totalChecked := 0
	if tc, ok := drift["total_checked"].(float64); ok {
		totalChecked = int(tc)
	} else if tc, ok := drift["total_checked"].(int); ok {
		totalChecked = tc
	}
	missingCount := 0
	if mc, ok := drift["missing_count"].(float64); ok {
		missingCount = int(mc)
	} else if mc, ok := drift["missing_count"].(int); ok {
		missingCount = mc
	}

	if inSync {
		fmt.Fprintf(ui.OutWriter, "    Status: ✓ In Sync (All %d tracked resources verified in workspace)\n", totalChecked)
		return
	}

	fmt.Fprintf(ui.OutWriter, "    Status: ⚠ Drifted (%d of %d tracked resources missing or modified)\n", missingCount, totalChecked)
	if missingMap, ok := drift["missing_resources"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(missingMap))
		for k := range missingMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items := parseStringListField(missingMap[k])
			if len(items) > 0 {
				fmt.Fprintf(ui.OutWriter, "    Missing %s: %s\n", k, strings.Join(items, ", "))
			}
		}
	}
}

// -----------------------------------------------------------------------------
// 5. angler bundle run
// -----------------------------------------------------------------------------

var bundleRunCmd = &cobra.Command{
	Use:   "run [pipeline]",
	Short: "Trigger ad-hoc execution of a bundle pipeline or DAG",
	Long: `Execute an Airflow pipeline defined within the deployed bundle.
Pass --conf to inject runtime JSON parameters into the DAG run, and --wait to poll until completion.

Examples:
  angler bundle run orders_etl -T dev
  angler bundle run orders_etl -T dev --conf '{"batch_date":"2026-11-10"}' --wait
  angler bundle run --bundle-id 96fd3d6c-2c6e-4b81-8198-453ea28a085a`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineName := ""
		if len(args) > 0 {
			pipelineName = args[0]
		}

		var confMap map[string]interface{}
		if strings.TrimSpace(bundleConfFlag) != "" {
			if err := json.Unmarshal([]byte(bundleConfFlag), &confMap); err != nil {
				return ui.NewUsageError(fmt.Errorf("invalid JSON for --conf: %w", err))
			}
		}

		c := getClient()
		bundleID, err := resolveBundleID(c, bundleIDFlag, bundleTargetFlag)
		if err != nil {
			return err
		}

		res, err := c.RunBundle(bundleID, pipelineName, confMap)
		if err != nil {
			return err
		}

		if !bundleWaitFlag {
			return ui.RenderOutput(res, func() {
				ui.PrintSuccess("Pipeline triggered successfully!")
				if res.BundleName != "" {
					fmt.Fprintf(ui.OutWriter, "  Bundle:    %s\n", res.BundleName)
				}
				fmt.Fprintf(ui.OutWriter, "  Pipeline:  %s\n", res.PipelineName)
				fmt.Fprintf(ui.OutWriter, "  DAG ID:    %s\n", res.DAGID)
				fmt.Fprintf(ui.OutWriter, "  Run ID:    %s\n", res.DAGRunID)
				fmt.Fprintf(ui.OutWriter, "  Status:    %s\n", res.Status)
			})
		}

		// Poll DAG run until completion
		timeout := bundleTimeoutFlag
		if timeout <= 0 {
			timeout = 10 * time.Minute
		}
		pollInterval := bundlePollInterval
		if pollInterval <= 0 {
			pollInterval = 3 * time.Second
		}

		ui.PrintInfo("Triggered bundle pipeline '%s' (Run ID: %s). Waiting for completion (timeout: %s)...", res.PipelineName, res.DAGRunID, timeout)

		deadline := time.Now().Add(timeout)
		var finalRun *client.PipelineRun
		for time.Now().Before(deadline) {
			run, pollErr := c.GetPipelineRun(res.DAGID, res.DAGRunID)
			if pollErr == nil && run != nil {
				finalRun = run
				state := strings.ToLower(run.State)
				if state == "success" || state == "failed" || state == "cancelled" || state == "skipped" {
					break
				}
			}
			time.Sleep(pollInterval)
		}

		if finalRun == nil {
			return fmt.Errorf("timeout waiting for bundle pipeline run '%s' on DAG '%s'", res.DAGRunID, res.DAGID)
		}

		tasksResp, _ := c.ListPipelineTaskInstances(res.DAGID, res.DAGRunID)
		resultPayload := map[string]interface{}{
			"bundle_id":     res.BundleID,
			"bundle_name":   res.BundleName,
			"pipeline_name": res.PipelineName,
			"dag_id":        res.DAGID,
			"dag_run_id":    finalRun.DagRunID,
			"status":        finalRun.State,
			"duration":      finalRun.DurationSeconds,
		}
		if tasksResp != nil {
			resultPayload["tasks"] = tasksResp.Tasks
		}

		return ui.RenderOutput(resultPayload, func() {
			if strings.ToLower(finalRun.State) == "success" {
				ui.PrintSuccess("Bundle pipeline '%s' completed successfully! (Run ID: %s)", res.PipelineName, finalRun.DagRunID)
			} else {
				ui.PrintError("Bundle pipeline '%s' finished with state: %s", res.PipelineName, strings.ToUpper(finalRun.State))
			}
		})
	},
}

// -----------------------------------------------------------------------------
// 6. angler bundle generate (AI Prompt-to-Bundle Generator)
// -----------------------------------------------------------------------------

var bundleGenerateCmd = &cobra.Command{
	Use:     "generate [prompt]",
	Aliases: []string{"gen", "ai"},
	Short:   "Generate a complete Lake Asset Bundle manifest from a natural language prompt",
	Long: `Use the Lake of Tears AI Architect (with offline heuristic fallback) to synthesize
a complete, valid lake.toon or lake.yml Lake Asset Bundle manifest from a natural language description.

Examples:
  angler bundle generate "Build a customer 360 medallion pipeline with webhook ingestion"
  angler bundle generate "Biomedical PubMed RAG lakehouse with vector search" --format yaml --out lake.yml
  angler bundle generate "IoT sensor telemetry pipeline" --out lake.toon --deploy --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.TrimSpace(bundlePromptFlag)
		if len(args) > 0 {
			prompt = strings.TrimSpace(strings.Join(args, " "))
		}
		if prompt == "" {
			return ui.NewUsageError(fmt.Errorf("prompt is required (pass as argument or via --prompt)"))
		}

		format := strings.ToLower(strings.TrimSpace(bundleFormatFlag))
		if format == "" {
			format = "toon"
		}
		if format == "yml" {
			format = "yaml"
		}
		if format != "toon" && format != "yaml" {
			return ui.NewUsageError(fmt.Errorf("unsupported bundle format '%s' (supported: toon, yaml)", format))
		}

		c := getClient()
		genRes := generateBundleFromPrompt(c, prompt, format)

		if bundleOutFlag != "" {
			if err := os.WriteFile(bundleOutFlag, []byte(genRes.Manifest), 0644); err != nil {
				return fmt.Errorf("failed to write generated manifest to '%s': %w", bundleOutFlag, err)
			}
		}

		var deployRes *client.BundleDeployResponse
		if bundleDeployFlag {
			dep, err := c.DeployBundle(&client.BundleDeployRequest{
				ManifestContent: genRes.Manifest,
				ManifestFormat:  format,
				Target:          bundleTargetFlag,
				DryRun:          bundleDryRunFlag,
			})
			if err != nil {
				return err
			}
			deployRes = dep
		}

		outputPayload := map[string]interface{}{
			"prompt":   prompt,
			"format":   genRes.Format,
			"model":    genRes.Model,
			"valid":    genRes.Valid,
			"manifest": genRes.Manifest,
		}
		if bundleOutFlag != "" {
			outputPayload["output_file"] = bundleOutFlag
		}
		if deployRes != nil {
			outputPayload["deployment"] = deployRes
		}

		return ui.RenderOutput(outputPayload, func() {
			ui.PrintSuccess("Generated Lake Asset Bundle (%s) via %s", strings.ToUpper(genRes.Format), genRes.Model)
			if bundleOutFlag != "" {
				fmt.Fprintf(ui.OutWriter, "  Saved manifest to: %s\n", bundleOutFlag)
			}
			fmt.Fprintf(ui.OutWriter, "\n%s\n", strings.TrimSpace(genRes.Manifest))

			if deployRes != nil {
				if bundleDryRunFlag || deployRes.DryRun {
					ui.PrintWarning("DRY RUN deployment simulation completed for '%s' (Target: %s)", deployRes.BundleName, deployRes.Target)
				} else {
					ui.PrintSuccess("Deployed generated bundle '%s' to target '%s' (ID: %s)", deployRes.BundleName, deployRes.Target, deployRes.BundleID)
				}
			}
		})
	},
}

func generateBundleFromPrompt(c *client.Client, prompt, format string) *client.BundleGenerateResponse {
	// 1. Try server-side AI Bundle Generator endpoint (/api/bundles/generate-ai)
	if c != nil && c.Endpoint != "" {
		if apiRes, err := c.GenerateBundleAI(prompt, format); err == nil && apiRes != nil && strings.TrimSpace(apiRes.Manifest) != "" {
			// If user requested YAML and server returned TOON without YAML structure, only use directly if format matches
			if format == "toon" || strings.EqualFold(apiRes.Format, "yaml") || !strings.Contains(apiRes.Manifest, "catalogs[") {
				apiRes.Format = format
				return apiRes
			}
		}
	}

	// 2. Fallback to deterministic intelligent heuristic architect (supports both TOON and YAML)
	ws := "analytics"
	if c != nil && strings.TrimSpace(c.Workspace) != "" {
		ws = strings.TrimSpace(c.Workspace)
	}
	manifest := buildHeuristicBundleManifest(prompt, format, ws)
	return &client.BundleGenerateResponse{
		Manifest: manifest,
		Format:   format,
		Model:    "heuristic-starter-architect",
		Valid:    true,
	}
}

func buildHeuristicBundleManifest(prompt, format, workspace string) string {
	wordRe := regexp.MustCompile(`[a-zA-Z0-9]+`)
	words := wordRe.FindAllString(strings.ToLower(prompt), -1)
	stopWords := map[string]bool{
		"build": true, "a": true, "an": true, "the": true, "with": true,
		"and": true, "in": true, "for": true, "to": true, "create": true,
		"bundle": true, "lakehouse": true, "pipeline": true, "from": true,
	}
	var nameParts []string
	for _, w := range words {
		if !stopWords[w] {
			nameParts = append(nameParts, w)
			if len(nameParts) == 3 {
				break
			}
		}
	}
	bundleName := "data-lake-bundle"
	if len(nameParts) > 0 {
		bundleName = strings.Join(nameParts, "-")
	}
	pipelineName := strings.ReplaceAll(bundleName, "-", "_")
	lowerPrompt := strings.ToLower(prompt)
	useWebhook := strings.Contains(lowerPrompt, "webhook") || strings.Contains(lowerPrompt, "event")
	useRAG := strings.Contains(lowerPrompt, "rag") || strings.Contains(lowerPrompt, "vector") || strings.Contains(lowerPrompt, "embed") || strings.Contains(lowerPrompt, "search")

	schedule := "0 4 * * *"
	webhookEnabledLineYAML := ""
	webhookEnabledLineTOON := ""
	if useWebhook {
		schedule = "@webhook"
		webhookEnabledLineYAML = "\n      webhook_enabled: true"
		webhookEnabledLineTOON = "\n      webhook_enabled: true"
	}

	desc := strings.TrimSpace(prompt)
	if len(desc) > 80 {
		desc = desc[:80] + "..."
	}

	if format == "yaml" || format == "yml" {
		vectorSection := ""
		if useRAG {
			vectorSection = `
  vector_indexes:
    - name: semantic_embeddings_idx
      table: main_lake.gold.aggregated_metrics
      column: content
      metric: cosine
      dimensions: 768`
		}
		return fmt.Sprintf(`bundle:
  name: %s
  version: 0.1.0
  description: %q

targets:
  dev:
    default: true
    workspace: %s
    default_catalog: main_lake
  prod:
    default: false
    workspace: %s
    default_catalog: main_lake

resources:
  catalogs:
    - name: main_lake
      description: Core Lakehouse Catalog
  schemas:
    - catalog: main_lake
      name: bronze
      tier: bronze
      description: Raw ingested events
    - catalog: main_lake
      name: silver
      tier: silver
      description: Cleaned dimensional models
    - catalog: main_lake
      name: gold
      tier: gold
      description: Aggregated analytics marts
  tables:
    - name: raw_events
      catalog: main_lake
      schema: bronze
      format: parquet
      s3_path_pattern: "s3://datalake/bronze/events/*.parquet"
    - name: cleaned_events
      catalog: main_lake
      schema: silver
      format: parquet
      s3_path_pattern: "s3://datalake/silver/events/*.parquet"
    - name: aggregated_metrics
      catalog: main_lake
      schema: gold
      format: parquet
      s3_path_pattern: "s3://datalake/gold/metrics/*.parquet"%s
  pipelines:
    %s_pipeline:
      schedule: %q%s
      tasks:
        - name: ingest_bronze
          type: python_script
          entrypoint: scripts/ingest.py
          requirements:
            - "pandas>=2.0"
            - "httpx"
        - name: aggregate_gold
          type: duckdb_sql
          entrypoint: sql/aggregate.sql
          upstream_tasks:
            - ingest_bronze
  dashboards:
    %s_overview:
      title: "%s Overview"
      widgets:
        kpi_count:
          type: metric
          query: "SELECT count(*) AS total_rows FROM main_lake.gold.aggregated_metrics"
`, bundleName, desc, workspace, workspace, vectorSection, pipelineName, schedule, webhookEnabledLineYAML, pipelineName, bundleName)
	}

	vectorSectionTOON := ""
	if useRAG {
		vectorSectionTOON = `
  vector_indexes[1]{name,table,column,metric,dimensions}:
    semantic_embeddings_idx,main_lake.gold.aggregated_metrics,content,cosine,768
`
	}

	return fmt.Sprintf(`# Lake Asset Bundle (TOON Specification)
bundle:
  name: %s
  version: 0.1.0
  description: %s

targets:
  dev:
    workspace: %s
    default_catalog: main_lake
  prod:
    workspace: %s
    default_catalog: main_lake

resources:
  catalogs[1]{name,description}:
    main_lake,Core Lakehouse Catalog

  schemas[3]{catalog,name,tier,description}:
    main_lake,bronze,bronze,Raw ingested events
    main_lake,silver,silver,Cleaned dimensional models
    main_lake,gold,gold,Aggregated analytics marts

  tables[3]{name,catalog,schema,format,s3_path_pattern}:
    raw_events,main_lake,bronze,parquet,"s3://datalake/bronze/events/*.parquet"
    cleaned_events,main_lake,silver,parquet,"s3://datalake/silver/events/*.parquet"
    aggregated_metrics,main_lake,gold,parquet,"s3://datalake/gold/metrics/*.parquet"
%s
  pipelines:
    %s_pipeline:
      schedule: %q%s
      tasks[2]{name,type,entrypoint,requirements,upstream_tasks}:
        ingest_bronze,python_script,scripts/ingest.py,"[pandas>=2.0,httpx]",[]
        aggregate_gold,duckdb_sql,sql/aggregate.sql,"",[ingest_bronze]

  dashboards:
    %s_overview:
      title: "%s Overview"
      widgets:
        kpi_count:
          type: metric
          query: "SELECT count(*) AS total_rows FROM main_lake.gold.aggregated_metrics"
`, bundleName, desc, workspace, workspace, vectorSectionTOON, pipelineName, schedule, webhookEnabledLineTOON, pipelineName, bundleName)
}

// -----------------------------------------------------------------------------
// 7. angler bundle destroy
// -----------------------------------------------------------------------------

var bundleDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Tear down and unprovision bundle resources in the target workspace",
	Long: `Permanently deletes provisioned tables, pipelines, and dashboards managed by the bundle.
Core medallion schemas (bronze, silver, gold) remain protected.

Examples:
  angler bundle destroy -T dev -y
  angler bundle destroy -T dev --purge-storage -y`,
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
			effectiveID := res.BundleID
			if effectiveID == "" {
				effectiveID = res.ID
			}
			ui.PrintSuccess("Bundle '%s' destroyed successfully.", effectiveID)
			if bundlePurgeStorage {
				fmt.Fprintln(ui.OutWriter, "  Storage artifacts purged.")
			}
		})
	},
}

// -----------------------------------------------------------------------------
// 8. angler bundle ls
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

func unmarshalManifestBytes(format string, raw []byte) (map[string]interface{}, error) {
	var parsed map[string]interface{}
	var err error
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "toon":
		err = toon.Unmarshal(raw, &parsed)
	case "yaml", "yml":
		err = yaml.Unmarshal(raw, &parsed)
	default:
		err = json.Unmarshal(raw, &parsed)
	}
	return parsed, err
}

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

	subdirs := []string{"tasks", "sql", "notebooks", "scripts", "python", "src"}
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
			if ext != ".sql" && ext != ".py" && ext != ".ipynb" && ext != ".sh" && ext != ".json" && ext != ".txt" {
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

func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func resolveBundleIDOrName(c *client.Client, idOrName, target string) (string, error) {
	if idOrName != "" {
		if looksLikeUUID(idOrName) {
			return idOrName, nil
		}
		if res, err := c.ListBundles(target, ""); err == nil && res != nil {
			for _, b := range res.Bundles {
				if (b.ID == idOrName || b.BundleName == idOrName) && (target == "" || b.Target == target) {
					return b.ID, nil
				}
			}
			for _, b := range res.Bundles {
				if b.ID == idOrName || b.BundleName == idOrName {
					return b.ID, nil
				}
			}
		}
		return idOrName, nil
	}
	return resolveBundleID(c, "", target)
}

func resolveBundleID(c *client.Client, explicitID, target string) (string, error) {
	if explicitID != "" {
		return explicitID, nil
	}

	// Try reading local manifest for bundle name
	filePath, format, err := findManifestFile("")
	bundleName := ""
	if err == nil {
		raw, readErr := os.ReadFile(filePath)
		if readErr == nil {
			parsed, parseErr := unmarshalManifestBytes(format, raw)
			if parseErr == nil {
				if bSec, ok := parsed["bundle"].(map[string]interface{}); ok && bSec["name"] != nil {
					bundleName = fmt.Sprintf("%v", bSec["name"])
				}
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

	formatResource := func(item interface{}) string {
		if s, ok := item.(string); ok {
			return s
		}
		if m, ok := item.(map[string]interface{}); ok {
			t, _ := m["type"].(string)
			n, _ := m["name"].(string)
			if n == "" {
				if title, ok := m["title"].(string); ok {
					n = title
				}
			}
			if t != "" && n != "" {
				return fmt.Sprintf("%s.%s", t, n)
			}
			if n != "" {
				return n
			}
		}
		return fmt.Sprintf("%v", item)
	}

	for _, r := range diff.ToCreate {
		rows = append(rows, []string{"+ CREATE", formatResource(r)})
	}
	for _, r := range diff.ToUpdate {
		rows = append(rows, []string{"~ UPDATE", formatResource(r)})
	}
	for _, r := range diff.ToDelete {
		rows = append(rows, []string{"- DELETE", formatResource(r)})
	}
	for _, r := range diff.Unchanged {
		rows = append(rows, []string{"= UNCHANGED", formatResource(r)})
	}

	if len(rows) > 0 {
		fmt.Fprintf(ui.OutWriter, "\nReconciliation Diff Plan:\n")
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, rows))
	}
}

func init() {
	// Persistent and local flags
	bundleCmd.PersistentFlags().StringVarP(&bundleTargetFlag, "target", "T", "", "Target environment (e.g. dev, staging, prod)")
	bundleCmd.PersistentFlags().StringVarP(&bundleFileFlag, "file", "f", "", "Path to bundle manifest file (lake.toon, lake.yml)")

	// init flags
	bundleInitCmd.Flags().StringVar(&bundleTemplateFlag, "template", "python-etl", "Template (python-etl, webhook-etl, rag-search, medallion)")
	bundleInitCmd.Flags().StringVar(&bundleFormatFlag, "format", "toon", "Manifest format: toon or yaml")

	// validate flags
	bundleValidateCmd.Flags().BoolVar(&bundleLocalFlag, "local", false, "Validate manifest syntax offline without contacting server")

	// deploy flags
	bundleDeployCmd.Flags().BoolVar(&bundleDryRunFlag, "dry-run", false, "Simulate deployment and output diff without persisting changes")

	// get / inspect / drift flags
	bundleGetCmd.Flags().StringVar(&bundleIDFlag, "bundle-id", "", "UUID of the deployed bundle")
	bundleGetCmd.Flags().BoolVar(&bundleNoDriftFlag, "no-drift", false, "Skip live drift detection check")
	bundleGetCmd.Flags().BoolVar(&bundleManifestFlag, "manifest", false, "Print stored raw manifest snapshot")

	// run flags
	bundleRunCmd.Flags().StringVar(&bundleIDFlag, "bundle-id", "", "UUID of the deployed bundle")
	bundleRunCmd.Flags().StringVar(&bundleConfFlag, "conf", "", "Runtime configuration JSON payload passed to the pipeline DAG run")
	bundleRunCmd.Flags().BoolVar(&bundleWaitFlag, "wait", false, "Wait for triggered pipeline run to complete before exiting")
	bundleRunCmd.Flags().DurationVar(&bundleTimeoutFlag, "timeout", 10*time.Minute, "Timeout when waiting for pipeline completion")
	bundleRunCmd.Flags().DurationVar(&bundlePollInterval, "poll-interval", 3*time.Second, "Polling interval when waiting for pipeline completion")

	// generate flags
	bundleGenerateCmd.Flags().StringVar(&bundleFormatFlag, "format", "toon", "Generated manifest format: toon or yaml")
	bundleGenerateCmd.Flags().StringVar(&bundleOutFlag, "out", "", "Write generated manifest directly to file path")
	bundleGenerateCmd.Flags().BoolVar(&bundleDeployFlag, "deploy", false, "Immediately deploy the generated bundle to the active workspace")
	bundleGenerateCmd.Flags().BoolVar(&bundleDryRunFlag, "dry-run", false, "When used with --deploy, simulate deployment without persisting")
	bundleGenerateCmd.Flags().StringVar(&bundlePromptFlag, "prompt", "", "Natural language description of the lakehouse bundle to generate")

	// destroy flags
	bundleDestroyCmd.Flags().StringVar(&bundleIDFlag, "bundle-id", "", "UUID of the deployed bundle")
	bundleDestroyCmd.Flags().BoolVar(&bundlePurgeStorage, "purge-storage", false, "Also permanently purge raw Parquet files and vector embeddings in object storage")

	// Add subcommands
	bundleCmd.AddCommand(bundleInitCmd)
	bundleCmd.AddCommand(bundleValidateCmd)
	bundleCmd.AddCommand(bundleDeployCmd)
	bundleCmd.AddCommand(bundleGetCmd)
	bundleCmd.AddCommand(bundleRunCmd)
	bundleCmd.AddCommand(bundleGenerateCmd)
	bundleCmd.AddCommand(bundleDestroyCmd)
	bundleCmd.AddCommand(bundleListCmd)

	rootCmd.AddCommand(bundleCmd)
}
