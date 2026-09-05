package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// One-shot foreground runtime.
//
// The legacy positional form and cobra `run` each started their own in-process
// protocol stack, bypassing the session registry, admission budgets and
// preflight -- so a config that the daemon would refuse could still be run, and
// two runtimes had to be kept honest with each other. `--once` is the same
// single-shot, foreground, exits-when-done shape, served by the daemon that
// already owns those checks.

// onceExit codes. Distinguishing a config the daemon refused from a run that
// failed is the point: a script that cannot tell them apart retries the
// unretryable.
const (
	onceExitOK      = 0
	onceExitRuntime = 1
	onceExitConfig  = 2
)

// OnceSummary is printed to stdout as JSON when a one-shot run finishes, so a
// caller reads a result rather than scraping the log.
type OnceSummary struct {
	SessionID       string  `json:"sessionId"`
	StatsAvailable  bool    `json:"statsAvailable"`
	Interface       string  `json:"interface"`
	ConfigPath      string  `json:"configPath"`
	DeviceCount     int     `json:"deviceCount"`
	DurationSeconds float64 `json:"durationSeconds"`
	PacketsSent     uint64  `json:"packetsSent"`
	PacketsReceived uint64  `json:"packetsReceived"`
	StoppedBy       string  `json:"stoppedBy"`
}

// runDaemonOnce starts one session in the foreground, runs it for the
// requested duration or until interrupted, then stops and reports.
func runDaemonOnce(options *daemonOptions, info versionInfo, args []string) error {
	// The daemon's own progress lines are diagnostics and belong on stderr, so
	// stdout can carry the JSON summary. It is not yet clean enough to pipe:
	// internal/protocols writes 162 diagnostics straight to os.Stdout, bypassing
	// this (niac#1805).
	logging.SetOutput(os.Stderr)
	defer logging.SetOutput(nil)

	iface, configPath, err := onceArgs(options, args)
	if err != nil {
		return withExitCode(onceExitConfig, err)
	}

	// A scenario or template name resolves like a path, the convenience the
	// deleted `run` command carried.
	source, err := resolveConfigSource(configPath)
	if err != nil {
		return withExitCode(onceExitConfig, err)
	}

	policies, err := parseAttachmentPolicies(options.attachmentPolicies)
	if err != nil {
		return withExitCode(onceExitConfig, err)
	}

	d, err := daemon.NewDaemon(daemon.Config{
		// No listener: a one-shot run is a foreground process, not a service,
		// and binding a port would make two concurrent runs collide on it.
		StoragePath:        options.storagePath,
		StorageKeep:        options.storageKeep,
		Version:            info.version,
		Commit:             info.commit,
		BuildTime:          info.date,
		ReleaseTrain:       info.releaseTrain,
		UIBuildHash:        info.uiBuildHash,
		AttachmentPolicies: policies,
		DebugLevel:         options.debugLevel,
	})
	if err != nil {
		return withExitCode(onceExitRuntime, fmt.Errorf("creating the runtime: %w", err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), onceShutdownGrace)
		defer cancel()
		_ = d.Shutdown(ctx)
	}()

	request := api.SimulationRequest{
		SessionID:      options.onceSessionID,
		Interface:      iface,
		ConfigData:     string(source.data),
		ConfigPath:     source.path,
		Attachment:     options.onceAttachment,
		AttachmentMode: fabric.AttachmentMode(options.onceAttachmentMode),
		AccessVLAN:     uint16(options.onceAccessVLAN), //nolint:gosec // range-checked below
	}
	if options.onceAccessVLAN < 0 || options.onceAccessVLAN > maxVLANID {
		return withExitCode(onceExitConfig,
			fmt.Errorf("access VLAN %d is outside 1..%d", options.onceAccessVLAN, maxVLANID))
	}

	started := time.Now()
	// Preflight and admission live inside StartSimulation, so a config the
	// daemon would refuse is refused here too -- which the legacy path could
	// not say.
	if startErr := d.StartSimulation(request); startErr != nil {
		return withExitCode(onceExitConfig, startErr)
	}

	stoppedBy := waitForOnce(options.onceDuration)

	status := d.GetStatus()
	// Read the counters before stopping: the stack is torn down on stop, and a
	// summary reporting zero packets for a run that carried traffic would be
	// worse than reporting none at all.
	stats, haveStats := d.SessionStats(options.onceSessionID)
	if stopErr := d.StopSimulation(options.onceSessionID); stopErr != nil {
		return withExitCode(onceExitRuntime, fmt.Errorf("stopping the session: %w", stopErr))
	}

	summary := OnceSummary{
		SessionID:       status.SessionID,
		StatsAvailable:  haveStats,
		Interface:       iface,
		ConfigPath:      source.label,
		DeviceCount:     status.DeviceCount,
		DurationSeconds: time.Since(started).Seconds(),
		PacketsSent:     stats.PacketsSent,
		PacketsReceived: stats.PacketsReceived,
		StoppedBy:       stoppedBy,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(summary); encodeErr != nil {
		return withExitCode(onceExitRuntime, encodeErr)
	}

	return nil
}

const (
	// maxVLANID is the highest 802.1Q VLAN identifier.
	maxVLANID = 4094
	// onceShutdownGrace is how long the runtime is given to close cleanly
	// before the process exits anyway.
	onceShutdownGrace = 10 * time.Second
	// onceArgsInterfaceAndConfig is the positional form: <interface> <config>.
	onceArgsInterfaceAndConfig = 2
)

// waitForOnce blocks for the run's duration or until interrupted, reporting
// which ended it. A zero duration means "until interrupted", which is what a
// soak or a manual session wants.
func waitForOnce(duration time.Duration) string {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if duration <= 0 {
		<-sigChan

		return "signal"
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return "duration"
	case <-sigChan:
		return "signal"
	}
}

// onceArgs resolves the interface and config path from the positional form
// (`daemon --once <iface> <config>`), which is the shape the legacy runtime
// used and the one the README documents.
func onceArgs(options *daemonOptions, args []string) (string, string, error) {
	iface, configPath := options.onceInterface, options.onceConfig
	switch len(args) {
	case 0:
	case 1:
		configPath = args[0]
	case onceArgsInterfaceAndConfig:
		iface, configPath = args[0], args[1]
	default:
		return "", "", fmt.Errorf(
			"expected at most <interface> <config>, got %d arguments", len(args))
	}

	if iface == "" {
		return "", "", errors.New("an interface is required: niac daemon --once <interface> <config>")
	}
	if configPath == "" {
		return "", "", errors.New("a config file is required: niac daemon --once <interface> <config>")
	}

	return iface, configPath, nil
}
