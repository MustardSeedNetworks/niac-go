package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/daemon"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

type daemonOptions struct {
	listen      string
	token       string
	storagePath string
}

func addDaemonCommand(root *cobra.Command, info versionInfo) {
	options := new(daemonOptions)

	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run NIAC in daemon mode with web UI control",
		Long: `Start NIAC as a daemon process that serves the web UI and allows
starting/stopping simulations dynamically without restarting the daemon.

The daemon runs the API server and web UI independently from the simulation
engine, allowing you to:
  - Start/stop simulations from the web UI
  - Change network interfaces without restarting
  - Switch between different configuration files
  - Manage multiple simulation sessions

Example:
  niac daemon --listen :8080 --token mysecrettoken

The web UI will be available at http://localhost:8080`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDaemon(options, info)
		},
	}

	daemonCmd.Flags().StringVar(&options.listen, "listen", ":8080", "Address to listen on for API and web UI")
	daemonCmd.Flags().StringVar(&options.token, "token", "", "Bearer token for API authentication (RECOMMENDED for network-exposed instances)")
	daemonCmd.Flags().
		StringVar(&options.storagePath, "storage", "~/.niac/niac.db", "Path to run history database (use 'disabled' to disable)")

	root.AddCommand(daemonCmd)
}

func runDaemon(options *daemonOptions, info versionInfo) error {
	logging.InitColors(true)

	logging.Infof("Starting NIAC Daemon v%s", info.version)
	logging.Infof("Web UI will be available at http://localhost%s", options.listen)
	if options.token != "" {
		logging.Infof("API authentication enabled")
	} else {
		logging.Warningf("SECURITY: No API token set. Anyone with network access can control simulations.")
		logging.Warningf("         Use --token for production or network-exposed deployments.")
	}

	// Create daemon instance
	d, err := daemon.NewDaemon(daemon.Config{
		ListenAddr:  options.listen,
		Token:       options.token,
		StoragePath: options.storagePath,
		Version:     info.version,
		Commit:      info.commit,
		BuildTime:   info.date,
		UIBuildHash: info.uiBuildHash,
	})
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Start the daemon
	if startErr := d.Start(); startErr != nil {
		return fmt.Errorf("failed to start daemon: %w", startErr)
	}

	logging.Successf("✓ Daemon started successfully")
	logging.Infof("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logging.Infof("\nShutting down daemon...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), statsTickerInterval*time.Second)
	defer cancel()

	if shutdownErr := d.Shutdown(ctx); shutdownErr != nil {
		logging.Errorf("Error during shutdown: %v", shutdownErr)
		return fmt.Errorf("failed to shutdown daemon: %w", shutdownErr)
	}

	logging.Successf("✓ Daemon stopped gracefully")
	return nil
}
