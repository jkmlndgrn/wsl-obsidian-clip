package cmd

import (
	"fmt"

	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running wsl-obsidian-clip daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.Stop(); err != nil {
			return fmt.Errorf("stop: %w", err)
		}
		fmt.Println("wsl-obsidian-clip daemon stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
