//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"

	"github.com/MustardSeedNetworks/niac-go/internal/attachpolicy"
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

	// The service is a third way to start a runtime on this data directory,
	// and `niac daemon` on the same machine must not start underneath it.
	lock, lockErr := acquireInstanceLock()
	if lockErr != nil {
		logging.Errorf("Refusing to start: %v", lockErr)
		return
	}
	defer func() { _ = lock.Release() }()

	dirs, dirErr := prepareServiceDirs(securedir.Options{})
	if dirErr != nil {
		logging.Errorf("Refusing to start: %v", dirErr)
		return
	}

	policies, policyErr := mergeAttachmentPolicies(
		filepath.Join(dirs.data, attachpolicy.FileName), attachpolicy.OperatorTrust(), nil,
	)
	if policyErr != nil {
		logging.Errorf("Refusing to start: %v", policyErr)
		return
	}

	d, err := daemon.NewDaemon(daemon.Config{
		ListenAddr:   listen,
		Token:        resolveAPIToken(""),
		TokenFile:    os.Getenv("NIAC_API_TOKEN_FILE"),
		StoragePath:  storagePath,
		Version:      p.info.version,
		Commit:       p.info.commit,
		BuildTime:    p.info.date,
		ReleaseTrain: p.info.releaseTrain,
		UIBuildHash:  p.info.uiBuildHash,
		CertDir:      dirs.cert,
		DebugLevel:   daemon.DefaultDebugLevel,

		AttachmentPolicies: policies,
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

	publishBoundPort(lock, d.BoundAddr())

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return installWindowsService(cmd.OutOrStdout())
		},
	}

	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Windows service",
		Long:  `Remove the NiAC Windows service.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return uninstallWindowsService(cmd.OutOrStdout())
		},
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Windows service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return controlWindowsService(cmd.OutOrStdout(), "start")
		},
	}

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop Windows service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return controlWindowsService(cmd.OutOrStdout(), "stop")
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show Windows service status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return showWindowsServiceStatus(cmd.OutOrStdout())
		},
	}

	serviceCmd.AddCommand(runCmd, installCmd, uninstallCmd, startCmd, stopCmd, statusCmd)
	root.AddCommand(serviceCmd)
}

// resolvedExecutable is the running binary's real path, with symlinks and
// junctions followed, so what is judged and registered is the file that runs.
func resolvedExecutable() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks in %s: %w", execPath, err)
	}
	return resolved, nil
}

// requireInstalledLocation refuses a binary that is not under
// %ProgramFiles%\NiAC. The service runs as SYSTEM, so registering a binary a
// user can replace would hand them SYSTEM.
func requireInstalledLocation(execPath string) error {
	programFiles, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil {
		return fmt.Errorf("locate Program Files: %w", err)
	}
	installDir, err := filepath.EvalSymlinks(filepath.Join(programFiles, "NiAC"))
	if err != nil {
		return fmt.Errorf("locate the install directory: %w", err)
	}
	return requireWithin(installDir, execPath)
}

// requireWithin accepts path only when it is inside root. Rel compares
// case-insensitively on Windows and, unlike a prefix test, does not accept a
// sibling such as NiACevil.
func requireWithin(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) ||
		rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf(
			"refusing to register %s: it is not inside %s; run install.ps1 from the release archive",
			path,
			root,
		)
	}
	return nil
}

func getServiceConfig() (*service.Config, error) {
	execPath, err := resolvedExecutable()
	if err != nil {
		return nil, err
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

func installWindowsService(out io.Writer) error {
	execPath, err := resolvedExecutable()
	if err != nil {
		return err
	}
	if locationErr := requireInstalledLocation(execPath); locationErr != nil {
		return locationErr
	}

	dirs, err := resolveServiceDirs(securedir.Options{})
	if err != nil {
		return err
	}
	if ensureErr := dirs.ensure(securedir.Options{}); ensureErr != nil {
		return ensureErr
	}

	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	if installErr := svc.Install(); installErr != nil {
		return fmt.Errorf("failed to install service: %w", installErr)
	}

	_, err = fmt.Fprint(out, `Service installed successfully. It starts automatically on boot.

To start it now:
  niac service start
  or: sc start NiACSimulator
`)
	return err
}

func uninstallWindowsService(out io.Writer) error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	// Stop service first if running
	_ = svc.Stop()

	if uninstallErr := svc.Uninstall(); uninstallErr != nil {
		return fmt.Errorf("failed to uninstall service: %w", uninstallErr)
	}

	_, err = fmt.Fprintln(out, "Service uninstalled successfully.")
	return err
}

func controlWindowsService(out io.Writer, action string) error {
	svc, err := newServiceFor(&niacProgram{})
	if err != nil {
		return err
	}

	switch action {
	case "start":
		if startErr := svc.Start(); startErr != nil {
			return fmt.Errorf("failed to start service: %w", startErr)
		}
		_, err = fmt.Fprintln(out, "Service started.")
	case "stop":
		if stopErr := svc.Stop(); stopErr != nil {
			return fmt.Errorf("failed to stop service: %w", stopErr)
		}
		_, err = fmt.Fprintln(out, "Service stopped.")
	}
	return err
}

func showWindowsServiceStatus(out io.Writer) error {
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

	_, err = fmt.Fprintf(out, "Service: %s\nStatus:  %s\n", windowsDisplayName, statusStr)
	return err
}
