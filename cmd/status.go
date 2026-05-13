package cmd

import (
	"fmt"

	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/daemon"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the wsl-obsidian-clip daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := daemon.Status()
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}
		fmt.Println(info)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
