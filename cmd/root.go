package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wsl-obsidian-clip",
	Short: "Bridge Windows clipboard screenshots into Obsidian vault attachments",
	Long: `wsl-obsidian-clip is a daemon that polls the Windows clipboard for screenshot images,
	offers an Obsidian embed (![[image.png]]) on the X11 clipboard, and saves the
	file in your vault while respecting your Obsidian settings.`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
