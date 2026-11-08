package cmd

import (
	"testing"
)

func TestComputeCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundCompute bool
	for _, c := range commands {
		if c.Name() == "compute" {
			foundCompute = true
			subCmds := c.Commands()
			subMap := make(map[string]bool)
			for _, sub := range subCmds {
				subMap[sub.Name()] = true
			}

			if !subMap["status"] {
				t.Errorf("expected 'status' subcommand on compute")
			}
			if !subMap["list"] {
				t.Errorf("expected 'list' subcommand on compute")
			}
			if !subMap["topology"] {
				t.Errorf("expected 'topology' subcommand on compute")
			}
		}
	}

	if !foundCompute {
		t.Errorf("expected 'compute' command to be registered on rootCmd")
	}
}
