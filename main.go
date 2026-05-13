package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jkmlndgrn/wsl-obsidian-clip/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
