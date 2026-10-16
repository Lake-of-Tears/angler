package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Lake-of-Tears/angler/internal/ui"
	"github.com/spf13/cobra"
)

var (
	downComposePath string
	removeVolumes   bool
)

var downCmd = &cobra.Command{
	Use:     "down",
	Aliases: []string{"dock"},
	Short:   "Stop and dock the Lake of Tears stack (alias: dock)",
	Long: `Stops and removes Lake of Tears containers and networks.

Examples:
  angler down
  angler dock -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerCmd, err := exec.LookPath("docker")
		if err != nil {
			return fmt.Errorf("docker binary not found in PATH")
		}

		composeFile := downComposePath
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
			argsCompose = append(argsCompose, "-f", composeFile)
		}
		argsCompose = append(argsCompose, "down")
		if removeVolumes {
			argsCompose = append(argsCompose, "-v")
		}

		ui.PrintInfo("Docking Lake of Tears stack...")
		execCmd := exec.Command(dockerCmd, argsCompose...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("docker compose down failed: %w", err)
		}

		ui.PrintSuccess("Lake of Tears stack docked safely.")
		return nil
	},
}

func init() {
	downCmd.Flags().StringVarP(&downComposePath, "file", "f", "", "path to docker-compose.yml")
	downCmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "remove named volumes declared in compose")
	rootCmd.AddCommand(downCmd)
}
