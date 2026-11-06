package cmd

import (
	"fmt"
	"sort"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var computeCmd = &cobra.Command{
	Use:     "compute",
	Aliases: []string{"clusters"},
	Short:   "Manage and inspect Lake of Tears cluster compute engines and topology",
	Long: `Inspect and control cluster compute services (DuckDB, MinIO, Airflow, Jupyter, Superset)
and deployment topology across Kubernetes, Helm, and local stacks.

Aliases:
  angler clusters`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runComputeStatus()
	},
}

var computeStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"health"},
	Short:   "Check compute cluster health, uptime, and service allocations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runComputeStatus()
	},
}

var computeListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List compute engines, pods, and resource limits",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runComputeList()
	},
}

var computeTopologyCmd = &cobra.Command{
	Use:     "topology",
	Aliases: []string{"topo"},
	Short:   "Inspect cluster ingress routes, URLs, and persistent storage volumes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runComputeTopology()
	},
}

func runComputeStatus() error {
	c := getClient()
	ctx := getActiveContext()

	ui.PrintInfo("Querying cluster status from %s ...", ui.StyleText(ui.TitleStyle, ctx.Endpoint))

	status, err := c.ClusterStatus()
	if err != nil {
		return err
	}

	return ui.RenderOutput(status, func() {
		statusStyle := ui.SuccessStyle
		if status.Status != "healthy" {
			statusStyle = ui.ErrorStyle
		}

		ui.PrintSuccess("Cluster Status: %s", ui.StyleText(statusStyle, status.Status))
		fmt.Fprintf(ui.OutWriter, "Mode:        %s\n", status.Mode)
		fmt.Fprintf(ui.OutWriter, "Namespace:   %s\n", status.Namespace)
		fmt.Fprintf(ui.OutWriter, "Connected:   %v\n", status.ClusterConnected)
		fmt.Fprintf(ui.OutWriter, "Services:    %d healthy / %d total\n", status.HealthyCount, status.TotalServices)
		fmt.Fprintf(ui.OutWriter, "Timestamp:   %s\n\n", status.Timestamp)

		tableRows := make([][]string, 0, len(status.Services))
		var keys []string
		for k := range status.Services {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			srv := status.Services[k]
			latencyStr := "—"
			if srv.LatencyMs != nil {
				latencyStr = fmt.Sprintf("%.1fms", *srv.LatencyMs)
			}
			restartsStr := fmt.Sprintf("%d", srv.Restarts)
			limitsStr := "—"
			if srv.CpuLimit != nil || srv.MemoryLimit != nil {
				limitsStr = ""
				if srv.CpuLimit != nil {
					limitsStr += "cpu:" + *srv.CpuLimit
				}
				if srv.MemoryLimit != nil {
					if limitsStr != "" {
						limitsStr += " "
					}
					limitsStr += "mem:" + *srv.MemoryLimit
				}
			}

			tableRows = append(tableRows, []string{
				srv.DisplayName,
				srv.Category,
				srv.Status,
				restartsStr,
				latencyStr,
				limitsStr,
			})
		}

		headers := []string{"Service", "Category", "Status", "Restarts", "Latency", "Allocations"}
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
	})
}

func runComputeList() error {
	c := getClient()
	status, err := c.ClusterStatus()
	if err != nil {
		return err
	}

	return ui.RenderOutput(status, func() {
		tableRows := make([][]string, 0, len(status.Services))
		var keys []string
		for k := range status.Services {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			srv := status.Services[k]
			pod := "—"
			if srv.PodName != nil && *srv.PodName != "" {
				pod = *srv.PodName
			}
			node := "—"
			if srv.NodeName != nil && *srv.NodeName != "" {
				node = *srv.NodeName
			}
			uptime := "—"
			if srv.Uptime != nil && *srv.Uptime != "" {
				uptime = *srv.Uptime
			}

			tableRows = append(tableRows, []string{
				srv.Name,
				srv.Category,
				srv.Status,
				pod,
				node,
				uptime,
			})
		}

		headers := []string{"Service ID", "Category", "Status", "Pod", "Node", "Uptime"}
		fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, tableRows))
		ui.PrintSuccess("Total compute services: %d", len(status.Services))
	})
}

func runComputeTopology() error {
	c := getClient()
	ctx := getActiveContext()

	ui.PrintInfo("Querying cluster topology from %s ...", ui.StyleText(ui.TitleStyle, ctx.Endpoint))

	topology, err := c.ClusterTopology()
	if err != nil {
		return err
	}

	return ui.RenderOutput(topology, func() {
		fmt.Fprintf(ui.OutWriter, "Namespace: %s\n", topology.Namespace)
		fmt.Fprintf(ui.OutWriter, "Mode:      %s\n\n", topology.Mode)

		fmt.Fprintln(ui.OutWriter, ui.StyleText(ui.TitleStyle, "Active Ingress Routes:"))
		if len(topology.IngressRoutes) > 0 {
			routeRows := make([][]string, len(topology.IngressRoutes))
			for i, r := range topology.IngressRoutes {
				tlsStr := "No"
				if r.TLSEnabled {
					tlsStr = "Yes"
				}
				routeRows[i] = []string{
					r.Service,
					r.Host,
					r.Path,
					tlsStr,
					r.URL,
				}
			}
			headers := []string{"Service", "Host", "Path", "TLS", "Full URL"}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, routeRows))
		} else {
			ui.PrintInfo("No ingress routes configured.")
		}

		fmt.Fprintln(ui.OutWriter, "\n"+ui.StyleText(ui.TitleStyle, "Storage Volumes & Mounts:"))
		if len(topology.StorageVolumes) > 0 {
			volRows := make([][]string, len(topology.StorageVolumes))
			for i, v := range topology.StorageVolumes {
				sizeStr := "—"
				if v.Size != nil && *v.Size != "" {
					sizeStr = *v.Size
				}
				scStr := "—"
				if v.StorageClass != nil && *v.StorageClass != "" {
					scStr = *v.StorageClass
				}
				statusStr := "—"
				if v.Status != nil && *v.Status != "" {
					statusStr = *v.Status
				}
				volRows[i] = []string{
					v.Name,
					v.MountPath,
					sizeStr,
					scStr,
					statusStr,
				}
			}
			headers := []string{"Volume Name", "Mount Path", "Size", "Storage Class", "Status"}
			fmt.Fprintln(ui.OutWriter, ui.RenderTable(headers, volRows))
		} else {
			ui.PrintInfo("No storage volumes found.")
		}
	})
}

func init() {
	computeCmd.AddCommand(computeStatusCmd)
	computeCmd.AddCommand(computeListCmd)
	computeCmd.AddCommand(computeTopologyCmd)
	rootCmd.AddCommand(computeCmd)
}
