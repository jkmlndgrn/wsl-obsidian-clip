package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/clipboard"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/config"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/daemon"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/embed"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/obsidian"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/platform"
	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/poller"
	"github.com/spf13/cobra"
)

var (
	flagDaemon   bool
	flagInterval int
	flagVerbose  bool
	flagQuiet    bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start polling the Windows clipboard for screenshots",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&flagDaemon, "daemon", "d", false, "Run as background daemon")
	startCmd.Flags().IntVarP(&flagInterval, "interval", "i", 250, "Polling interval in milliseconds (100-5000)")
	startCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Log all PowerShell I/O")
	startCmd.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress informational messages")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	if flagInterval < 100 || flagInterval > 5000 {
		return fmt.Errorf("interval must be between 100 and 5000 ms")
	}

	loadedConfig, err := config.Load()
	if err != nil {
		return err
	}

	if err := checkPlatform(flagQuiet); err != nil {
		return fmt.Errorf("platform check failed: %w", err)
	}

	powershellPath, err := platform.ResolvePowerShell(loadedConfig.Config.PowerShellPathOverride)
	if err != nil {
		return fmt.Errorf("powershell detection: %w", err)
	}

	vault, err := obsidian.DetectVault(loadedConfig.Config)
	if err != nil {
		return fmt.Errorf("vault detection: %w", err)
	}

	if flagDaemon {
		return daemon.Daemonize(flagQuiet)
	}

	cleanup, err := daemon.MarkChildRunning()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	logger := log.New(os.Stderr, "[wsl-obsidian-clip] ", log.LstdFlags)
	if flagQuiet {
		logger.SetOutput(io.Discard)
	}

	logger.Printf("Using config: %s", loadedConfig.Path)
	logger.Printf("Using PowerShell: %s", powershellPath)
	logger.Printf("Using vault: %s", vault.Path)
	logger.Printf("Attachment dir: %s", vault.AttachmentDir())

	if err := os.MkdirAll(vault.AttachmentDir(), 0755); err != nil {
		return fmt.Errorf("create attachment dir: %w", err)
	}

	embedWriter, err := embed.NewX11Writer(logger)
	if err != nil {
		return fmt.Errorf("start X11 clipboard owner: %w", err)
	}

	clientFactory := func() (poller.Clipboard, error) {
		return clipboard.NewClient(logger, flagVerbose, powershellPath)
	}

	return poller.Run(cmd.Context(), logger, flagInterval, vault.AttachmentDir(), clientFactory, embedWriter)
}

func checkPlatform(quiet bool) error {
	if !quiet {
		return platformCheck()
	}

	oldOutput := log.Default().Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(oldOutput)

	return platformCheck()
}

var platformCheck = platform.Check
