package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/daemon"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
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
	RunE: runDaemon,
}

var daemonOpts struct {
	listen      string
	token       string
	storagePath string
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	daemonCmd.Flags().StringVar(&daemonOpts.listen, "listen", ":8080", "Address to listen on for API and web UI")
	daemonCmd.Flags().StringVar(&daemonOpts.token, "token", "", "Bearer token for API authentication (optional)")
	daemonCmd.Flags().StringVar(&daemonOpts.storagePath, "storage", "~/.niac/niac.db", "Path to run history database (use 'disabled' to disable)")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	logging.InitColors(true)

	logging.Info("Starting NIAC Daemon v%s", version)
	logging.Info("Web UI will be available at http://localhost%s", daemonOpts.listen)
	if daemonOpts.token != "" {
		logging.Info("API authentication enabled")
	} else {
		logging.Info("WARNING: No API token set - consider using --token for security")
	}

	// Create daemon instance
	d, err := daemon.NewDaemon(daemon.Config{
		ListenAddr:  daemonOpts.listen,
		Token:       daemonOpts.token,
		StoragePath: daemonOpts.storagePath,
		Version:     version,
	})
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Start the daemon
	if err := d.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	logging.Success("✓ Daemon started successfully")
	logging.Info("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logging.Info("\nShutting down daemon...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := d.Shutdown(ctx); err != nil {
		logging.Error("Error during shutdown: %v", err)
		return err
	}

	logging.Success("✓ Daemon stopped gracefully")
	return nil
}
