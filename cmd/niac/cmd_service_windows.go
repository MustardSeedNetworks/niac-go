//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	// windowsServiceName is the internal service name used by Windows SCM.
	windowsServiceName = "NiACSimulator"

	// windowsDisplayName is the human-readable service name shown in services.msc.
	windowsDisplayName = "NiAC Network Simulator Service"

	// windowsDescription is the detailed description shown in service properties.
	windowsDescription = "Network device simulation service by Mustard Seed Networks"

	// serviceStopTimeout is the maximum time to wait for graceful shutdown.
	serviceStopTimeout = 30 * time.Second
)

// niacProgram implements service.Interface for Windows service management.
type niacProgram struct {
	info   versionInfo
	daemon *daemon.Daemon
	stopCh chan struct{}
	doneCh chan struct{}
}

// Start is called when the Windows Service Manager starts the service.
func (p *niacProgram) Start(_ service.Service) error {
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	go p.run()
	return nil
}

// Stop is called when the Windows Service Manager requests service stop.
func (p *niacProgram) Stop(_ service.Service) error {
	close(p.stopCh)

	select {
	case <-p.doneCh:
		return nil
	case <-time.After(serviceStopTimeout):
		return errors.New("service stop timeout")
	}
}

// run executes the daemon logic as a Windows service.
func (p *niacProgram) run() {
	defer close(p.doneCh)

	logging.InitColors(false)

	listen := os.Getenv("NIAC_LISTEN_ADDR")
	if listen == "" {
		listen = "127.0.0.1:8445"
	}

	storagePath := os.Getenv("NIAC_STORAGE_PATH")
	if storagePath == "" {
		storagePath = "~/.niac/niac.db"
	}
	if expanded, err := expandUserHome(storagePath); err == nil {
		storagePath = expanded
	}

	d, err := daemon.NewDaemon(daemon.Config{
		ListenAddr:   listen,
		Token:        resolveAPIToken(""),
		StoragePath:  storagePath,
		Version:      p.info.version,
		Commit:       p.info.commit,
		BuildTime:    p.info.date,
		ReleaseTrain: p.info.releaseTrain,
		UIBuildHash:  p.info.uiBuildHash,
		CertDir:      defaultCertDir(),
	})
	if err != nil {
		logging.Errorf("Failed to create daemon: %v", err)
		return
	}

	p.daemon = d

	if startErr := d.Start(); startErr != nil {
		logging.Errorf("Failed to start daemon: %v", startErr)
		return
	}

	logging.Infof("NiAC daemon started as Windows service")

	// Wait for stop signal
	<-p.stopCh

	logging.Infof("Shutting down NiAC daemon...")

	ctx, cancel := context.WithTimeout(context.Background(), serviceStopTimeout)
	defer cancel()

	if shutdownErr := d.Shutdown(ctx); shutdownErr != nil {
		logging.Errorf("Error during shutdown: %v", shutdownErr)
	}

	logging.Infof("NiAC daemon stopped")
}

func addServiceCommand(root *cobra.Command, _ *serviceOptions) {
	info := readVersionInfo()

	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Windows service management commands",
		Long:  `Manage the NiAC Windows service (install, uninstall, start, stop, run).`,
	}

	// Run command - runs as Windows service (internal use)
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run as Windows service (internal use)",
		Long:  `This command is called by the Windows Service Manager. Do not run directly.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWindowsService(info)
		},
	}

	// Install command
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install as Windows service",
		Long:  `Install NiAC as a Windows service that starts automatically on boot.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return installWindowsService()
		},
	}

	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Windows service",
		Long:  `Remove the NiAC Windows service.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return uninstallWindowsService()
		},
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Windows service",
		RunE: func(_ *cobra.Command, _ []string) error {
			return controlWindowsService("start")
		},
	}

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop Windows service",
		RunE: func(_ *cobra.Command, _ []string) error {
			return controlWindowsService("stop")
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show Windows service status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return showWindowsServiceStatus()
		},
	}

	serviceCmd.AddCommand(runCmd, installCmd, uninstallCmd, startCmd, stopCmd, statusCmd)
	root.AddCommand(serviceCmd)
}

func getServiceConfig() (*service.Config, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	return &service.Config{
		Name:        windowsServiceName,
		DisplayName: windowsDisplayName,
		Description: windowsDescription,
		Arguments:   []string{"service", "run"},
		Executable:  execPath,
	}, nil
}

// expandUserHome replaces a leading "~" in path with the current user's home
// directory. Used for the Windows service storage path config which historically
// hardcoded "~/.niac/niac.db" and never expanded the tilde.
func expandUserHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path, err
	}
	return home + path[1:], nil
}

// newServiceFor creates a kardianos service.Service for the given program,
// surfacing the os.Executable lookup failure rather than silently swallowing it.
func newServiceFor(prg *niacProgram) (service.Service, error) {
	cfg, err := getServiceConfig()
	if err != nil {
		return nil, err
	}
	svc, err := service.New(prg, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	return svc, nil
}

func runWindowsService(info versionInfo) error {
	svc, err := newServiceFor(&niacProgram{info: info})
	if err != nil {
		return err
	}

	return svc.Run()
}

func installWindowsService() error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	if err := svc.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("Service installed successfully.")
	fmt.Println("")
	fmt.Println("To start the service:")
	fmt.Println("  niac service start")
	fmt.Println("  or: sc start NiACSimulator")
	fmt.Println("")
	fmt.Println("To configure automatic startup:")
	fmt.Println("  sc config NiACSimulator start= auto")
	return nil
}

func uninstallWindowsService() error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	// Stop service first if running
	_ = svc.Stop()

	if err := svc.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}

func controlWindowsService(action string) error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	switch action {
	case "start":
		if err := svc.Start(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		fmt.Println("Service started.")
	case "stop":
		if err := svc.Stop(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		fmt.Println("Service stopped.")
	}
	return nil
}

func showWindowsServiceStatus() error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	status, err := svc.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	statusStr := "Unknown"
	switch status {
	case service.StatusRunning:
		statusStr = "Running"
	case service.StatusStopped:
		statusStr = "Stopped"
	case service.StatusUnknown:
		statusStr = "Unknown (service may not be installed)"
	}

	fmt.Printf("Service: %s\n", windowsDisplayName)
	fmt.Printf("Status:  %s\n", statusStr)
	return nil
}
