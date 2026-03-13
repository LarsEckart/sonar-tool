package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/lars/sonar-tool/cmd"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := cmd.New(version, os.Stdout, os.Stderr)
	if err := root.Run(ctx, os.Args); err != nil {
		if errors.Is(err, context.Canceled) {
			err = cmd.Interrupted()
		}
		cmd.RenderError(os.Stderr, err)
		os.Exit(cmd.ExitCode(err))
	}
}
