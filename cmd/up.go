package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	composePath string
	detachMode  bool
)

var upCmd = &cobra.Command{
	Use:     "up",
	Aliases: []string{"launch"},
	Short:   "Launch local Lake of Tears stack (alias: launch)",
	Long: `Start the Lake of Tears cluster containers using Docker Compose.

Examples:
  angler up
  angler launch -f ./deploy/docker-compose.yml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerCmd, err := exec.LookPath("docker")
		if err != nil {
			return fmt.Errorf("docker binary not found in PATH: please install Docker")
		}

		composeFile := composePath
		if composeFile == "" {
			candidates := []string{
				"docker-compose.yml",
				"docker-compose.yaml",
				"deploy/docker-compose.yml",
				"../lake-of-tears/deploy/docker-compose.yml",
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					composeFile = c
					break
				}
			}
		}

		argsCompose := []string{"compose"}
		if composeFile != "" {
			ui.PrintInfo("Using compose file: %s", composeFile)
			argsCompose = append(argsCompose, "-f", composeFile)
		}
		argsCompose = append(argsCompose, "up")
		if detachMode {
			argsCompose = append(argsCompose, "-d")
		}

		ui.PrintInfo("Launching Lake of Tears stack...")
		execCmd := exec.Command(dockerCmd, argsCompose...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("docker compose up failed: %w", err)
		}

		if detachMode {
			ui.PrintSuccess("Lake of Tears launched in background!")
			ui.PrintInfo("Waiting 3s for services to stabilize...")
			time.Sleep(3 * time.Second)
			_ = runStatusPing()
		}

		return nil
	},
}

func runStatusPing() error {
	c := getClient()
	health, err := c.Health()
	if err == nil {
		ui.PrintSuccess("Lake of Tears is healthy! (Status: %s)", health.Status)
		ui.PrintInfo("Web Shell: http://localhost:80 (or your configured port)")
	}
	return nil
}

func init() {
	upCmd.Flags().StringVarP(&composePath, "file", "f", "", "path to docker-compose.yml")
	upCmd.Flags().BoolVarP(&detachMode, "detach", "d", true, "run containers in the background")
	rootCmd.AddCommand(upCmd)
}
