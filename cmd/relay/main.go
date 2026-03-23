package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/erisristemena/relay/internal/app"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := newRootCommand(logger).Execute(); err != nil {
		logger.Error("relay command failed", "error", err)
		os.Exit(1)
	}
}

func newRootCommand(logger *slog.Logger) *cobra.Command {
	root := &cobra.Command{
		Use:   "relay",
		Short: "Launch the Relay local workspace",
		Version: fmt.Sprintf("%s (%s, %s)", version, commit, date),
	}

	root.AddCommand(newServeCommand(logger))
	return root
}

func newServeCommand(logger *slog.Logger) *cobra.Command {
	var preferredPort int
	var devMode bool
	var noBrowser bool

	command := &cobra.Command{
		Use:   "serve",
		Short: "Start Relay and open the local workspace",
		RunE: func(command *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return app.Run(ctx, app.Options{
				PreferredPort: preferredPort,
				DevMode:       devMode,
				NoBrowser:     noBrowser,
				Version:       version,
				Logger:        logger,
			})
		},
	}

	command.Flags().IntVar(&preferredPort, "port", 4747, "Preferred Relay port for this run")
	command.Flags().BoolVar(&devMode, "dev", false, "Proxy browser routes to a local Next.js dev server")
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "Start Relay without opening the browser automatically")

	return command
}