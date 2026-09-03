package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	daemonTLSPort         = 8445
	daemonDefaultListenIP = "127.0.0.1"
)

type daemonOptions struct {
	listen              string
	token               string
	storagePath         string
	webhookAllowedHosts []string
	attachmentPolicies  []string
	certDir             string
	apiToken            string
	// Wave 2 (SIGHUP rotation + scoped tokens) options. tokenFile is the
	// path to a 0o600 JSON file containing one or more {value, scope}
	// pairs; preferred over apiToken / NIAC_API_TOKEN. SIGHUP re-reads
	// whichever source is active.
	tokenFile string
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
  - Replace the active simulation without restarting the daemon
  - Run several scenarios at once, one per physical VLAN, on a trunk attachment

The daemon serves HTTPS on 127.0.0.1:8445 by default. Binding to a
non-loopback address (e.g. --listen 0.0.0.0) requires an API token via
NIAC_API_TOKEN or --api-token.`,
		Example: `  # Default: HTTPS on 127.0.0.1:8445 (loopback only, no token needed)
  niac daemon

  # Listen on all interfaces — requires an API token
  export NIAC_API_TOKEN=$(openssl rand -base64 32)
  niac daemon --listen 0.0.0.0

  # Use a token file with scoped tokens (read-only / read-write)
  niac daemon --token-file /etc/niac/tokens.json

  # Permit routed labs on an operator-managed access port
  niac daemon --attachment-policy eth0=access:200

  # Permit concurrent scenario VLANs on an operator-managed trunk
  niac daemon --attachment-policy eth0=trunk:200,201,202,203,204,205,299

  # Permit a directly connected untagged tester
  niac daemon --attachment-policy eth1=direct

  # Disable run-history persistence
  niac daemon --storage disabled`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDaemon(options, info)
		},
	}

	daemonCmd.Flags().
		StringVar(&options.listen, "listen", "", "Address to listen on for the HTTPS API and web UI (default: 127.0.0.1:8445)")
	daemonCmd.Flags().
		StringVar(&options.token, "token", "", "Bearer token for API authentication (DEPRECATED: use NIAC_API_TOKEN env var)")
	if err := daemonCmd.Flags().MarkDeprecated("token",
		"the value lands in /proc/<pid>/cmdline and `ps`. Set NIAC_API_TOKEN in the environment instead."); err != nil {
		panic(fmt.Errorf("mark --token deprecated: %w", err))
	}
	daemonCmd.Flags().
		StringVar(&options.storagePath, "storage", "~/.niac/niac.db", "Path to run history database (use 'disabled' to disable)")
	daemonCmd.Flags().
		StringSliceVar(&options.webhookAllowedHosts, "webhook-allowed-host", nil,
			"Hostname allowed as alert webhook destination (repeatable; if any are set, all webhook URLs must match exactly). When unset, the existing private-IP/blocked-hostname filter is used.")
	daemonCmd.Flags().
		StringArrayVar(&options.attachmentPolicies, "attachment-policy", nil,
			"Operator-approved routed attachment (repeatable): INTERFACE=direct, INTERFACE=access:VLAN, or INTERFACE=trunk:VLAN,...")

	daemonCmd.Flags().
		StringVar(&options.certDir, "cert-dir", "", "Directory holding the self-signed cert and key (default: certs/ relative to CWD; override with NIAC_CERT_DIR)")
	daemonCmd.Flags().
		StringVar(&options.apiToken, "api-token", "", "Bearer token (preferred: NIAC_API_TOKEN). Required when --listen is non-loopback.")
	daemonCmd.Flags().
		StringVar(&options.tokenFile, "token-file", "",
			"Path to a 0600 JSON file with scoped tokens (overrides --api-token / NIAC_API_TOKEN). "+
				"Schema: {\"tokens\":[{\"value\":\"...\",\"scope\":\"read-only|read-write\"}]}. "+
				"Re-read on SIGHUP.")

	root.AddCommand(daemonCmd)
}

// resolveDaemonTokenFile layers --token-file / NIAC_API_TOKEN_FILE in
// the documented precedence order. Returns the resolved path (or "").
func resolveDaemonTokenFile(o *daemonOptions) string {
	if o.tokenFile != "" {
		return o.tokenFile
	}
	return os.Getenv("NIAC_API_TOKEN_FILE")
}

func resolveDaemonListen(o *daemonOptions) string {
	listen := o.listen
	if listen == "" {
		listen = os.Getenv("NIAC_LISTEN_ADDR")
	}
	switch {
	case listen == "":
		listen = net.JoinHostPort(daemonDefaultListenIP, strconv.Itoa(daemonTLSPort))
	case !strings.Contains(listen, ":"):
		listen = net.JoinHostPort(listen, strconv.Itoa(daemonTLSPort))
	}
	return listen
}

// resolveDaemonCertDir returns the cert-dir to use. Explicit --cert-dir
// wins; otherwise NIAC_CERT_DIR; otherwise the platform default.
func resolveDaemonCertDir(o *daemonOptions) string {
	if o.certDir != "" {
		return o.certDir
	}
	return defaultCertDir()
}

// resolveDaemonAPIToken layers --api-token / NIAC_API_TOKEN / --token in
// the documented precedence order. Returns the resolved token. The
// legacy --token path still works but emits a deprecation warning via
// resolveAPIToken.
func resolveDaemonAPIToken(o *daemonOptions) string {
	if envToken := os.Getenv("NIAC_API_TOKEN"); envToken != "" {
		return envToken
	}
	if o.apiToken != "" {
		return o.apiToken
	}
	return resolveAPIToken(o.token)
}

func parseAttachmentPolicies(values []string) ([]fabric.PhysicalAttachmentPolicy, error) {
	policies := make([]fabric.PhysicalAttachmentPolicy, 0, len(values))
	// One interface carries N tagged sessions plus at most one native session,
	// exactly as the session registry models a trunk port with a native VLAN
	// (#1426), so it needs one policy per mode rather than one policy outright
	// (#1463). Direct is the exception: it is unisolated ownership of the whole
	// interface, so it cannot share one.
	seenModes := make(map[string]map[fabric.AttachmentMode]struct{}, len(values))
	for _, value := range values {
		policy, err := parseAttachmentPolicy(value)
		if err != nil {
			return nil, err
		}
		modes := seenModes[policy.Interface]
		if modes == nil {
			modes = make(map[fabric.AttachmentMode]struct{}, attachmentModesPerInterface)
			seenModes[policy.Interface] = modes
		}
		if _, duplicate := modes[policy.Mode]; duplicate {
			return nil, fmt.Errorf(
				"duplicate %s policy for interface %q in attachment policy %q",
				policy.Mode, policy.Interface, value,
			)
		}
		if exclusiveErr := checkDirectExclusive(policy, modes, value); exclusiveErr != nil {
			return nil, exclusiveErr
		}
		modes[policy.Mode] = struct{}{}
		policies = append(policies, policy)
	}
	return policies, nil
}

// checkDirectExclusive rejects a direct policy sharing an interface with any
// other mode, in either order.
func checkDirectExclusive(
	policy fabric.PhysicalAttachmentPolicy,
	modes map[fabric.AttachmentMode]struct{},
	value string,
) error {
	_, hasDirect := modes[fabric.ModeDirect]
	if !hasDirect && policy.Mode != fabric.ModeDirect {
		return nil
	}
	if hasDirect || len(modes) > 0 {
		return fmt.Errorf(
			"interface %q cannot combine direct with another attachment mode (%q)",
			policy.Interface, value,
		)
	}
	return nil
}

// attachmentModesPerInterface is the most modes one interface can hold: a
// trunk policy for the tagged sessions plus an access policy for the native one.
const attachmentModesPerInterface = 2

const attachmentPolicySyntax = "expected INTERFACE=direct, INTERFACE=access:VLAN, or INTERFACE=trunk:VLAN,..."

func parseAttachmentPolicy(value string) (fabric.PhysicalAttachmentPolicy, error) {
	interfaceName, modeValue, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(interfaceName) != interfaceName || interfaceName == "" {
		return fabric.PhysicalAttachmentPolicy{}, fmt.Errorf(
			"invalid attachment policy %q: %s", value, attachmentPolicySyntax,
		)
	}
	policy := fabric.PhysicalAttachmentPolicy{Interface: interfaceName}
	switch {
	case modeValue == string(fabric.ModeDirect):
		policy.Mode = fabric.ModeDirect
	case strings.HasPrefix(modeValue, string(fabric.ModeAccess)+":"):
		vlan, err := parseAttachmentVLAN(
			value,
			strings.TrimPrefix(modeValue, "access:"),
			"access VLAN",
		)
		if err != nil {
			return fabric.PhysicalAttachmentPolicy{}, err
		}
		policy.Mode, policy.AccessVLAN = fabric.ModeAccess, vlan
	case strings.HasPrefix(modeValue, string(fabric.ModeTrunk)+":"):
		vlans, err := parseTrunkVLANs(value, strings.TrimPrefix(modeValue, "trunk:"))
		if err != nil {
			return fabric.PhysicalAttachmentPolicy{}, err
		}
		policy.Mode, policy.AllowedVLANs = fabric.ModeTrunk, vlans
	default:
		return fabric.PhysicalAttachmentPolicy{}, fmt.Errorf(
			"invalid attachment policy %q: %s", value, attachmentPolicySyntax,
		)
	}
	return policy, nil
}

func parseAttachmentVLAN(value, text, label string) (uint16, error) {
	vlan, err := strconv.ParseUint(text, 10, 16)
	if err != nil || vlan == 0 || vlan > 4094 {
		return 0, fmt.Errorf(
			"invalid attachment policy %q: %s must be between 1 and 4094",
			value,
			label,
		)
	}
	return uint16(vlan), nil
}

func parseTrunkVLANs(value, text string) ([]uint16, error) {
	if text == "" {
		return nil, fmt.Errorf(
			"invalid attachment policy %q: trunk requires at least one VLAN",
			value,
		)
	}
	vlans := make([]uint16, 0, strings.Count(text, ",")+1)
	var previous uint16
	for item := range strings.SplitSeq(text, ",") {
		vlan, err := parseAttachmentVLAN(value, item, "trunk VLANs")
		if err != nil {
			return nil, err
		}
		if vlan == previous {
			return nil, fmt.Errorf("invalid attachment policy %q: duplicate VLAN %d", value, vlan)
		}
		if vlan < previous {
			return nil, fmt.Errorf(
				"invalid attachment policy %q: trunk VLANs must be in ascending order",
				value,
			)
		}
		vlans = append(vlans, vlan)
		previous = vlan
	}
	return vlans, nil
}

func runDaemon(options *daemonOptions, info versionInfo) error {
	logging.InitColors(true)

	listenAddr := resolveDaemonListen(options)
	token := resolveDaemonAPIToken(options)
	tokenFile := resolveDaemonTokenFile(options)
	certDir := resolveDaemonCertDir(options)
	attachmentPolicies, err := parseAttachmentPolicies(options.attachmentPolicies)
	if err != nil {
		return err
	}

	logging.Infof("Starting NIAC Daemon v%s", info.version)
	logging.Infof("Web UI will be available at https://%s", listenAddr)
	authEnabled := token != "" || tokenFile != ""
	switch {
	case tokenFile != "":
		logging.Infof("API authentication enabled (token file: %s; SIGHUP rotates)", tokenFile)
	case token != "":
		logging.Infof("API authentication enabled (single token via env/flag; SIGHUP rotates)")
	default:
		logging.Warningf(
			"SECURITY: No API token set. Anyone with network access can control simulations.",
		)
		logging.Warningf(
			"         Use NIAC_API_TOKEN env var for production or network-exposed deployments.",
		)
	}

	cfg := daemon.Config{
		ListenAddr:          listenAddr,
		Token:               token,
		TokenFile:           tokenFile,
		StoragePath:         options.storagePath,
		RecoveryPath:        daemon.DefaultRecoveryPath(),
		Version:             info.version,
		Commit:              info.commit,
		BuildTime:           info.date,
		ReleaseTrain:        info.releaseTrain,
		UIBuildHash:         info.uiBuildHash,
		WebhookAllowedHosts: options.webhookAllowedHosts,
		CertDir:             certDir,
		AttachmentPolicies:  attachmentPolicies,
	}

	// Sanity-check the listen address up front so we fail with the helpful
	// non-loopback-requires-token message before the API server starts and
	// returns the same gate (the duplicate check is intentional defense in
	// depth — the server will refuse too, but printing the friendly hint
	// here surfaces it earlier in the log timeline).
	if nonLoopback, parseErr := isNonLoopbackListen(listenAddr); parseErr == nil && nonLoopback &&
		!authEnabled {
		return errors.New(
			"niac refuses to bind a non-loopback address without an API token.\n" +
				"Set NIAC_API_TOKEN=<value>, --api-token=<value>, or --token-file=<path>.\n" +
				"If you want loopback-only (no auth), set --listen=127.0.0.1")
	}

	d, err := daemon.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	if startErr := d.Start(); startErr != nil {
		return fmt.Errorf("failed to start daemon: %w", startErr)
	}

	logging.Successf("✓ Daemon started successfully")
	logging.Infof("Press Ctrl+C to stop (SIGHUP rotates tokens without restart)")

	// Wave 2 (#91): SIGHUP rotates the bearer-token set without
	// restart. Operators can edit the token file or update
	// NIAC_API_TOKEN and `kill -HUP <pid>` to apply the new value;
	// in-flight requests under the old token are unaffected.
	//
	// This is daemon-mode-specific: standalone mode's SIGHUP handler
	// (main.go's handleReload) reloads the YAML config instead, because
	// there is no API token to rotate there. See docs/DEPLOYMENT.md
	// "Signal Handling".
	hupChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	defer signal.Stop(hupChan)
	hupDone := make(chan struct{})
	go handleSIGHUP(hupChan, hupDone, d)
	defer close(hupDone)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	<-sigChan

	logging.Infof("\nShutting down daemon...")

	ctx, cancel := context.WithTimeout(context.Background(), statsTickerInterval*time.Second)
	defer cancel()

	if shutdownErr := d.Shutdown(ctx); shutdownErr != nil {
		logging.Errorf("Error during shutdown: %v", shutdownErr)
		return fmt.Errorf("failed to shutdown daemon: %w", shutdownErr)
	}

	logging.Successf("✓ Daemon stopped gracefully")
	return nil
}

// handleSIGHUP services the SIGHUP rotation channel. It runs as a goroutine
// for the daemon's lifetime and exits when `done` is closed (the main
// loop closes it on shutdown so this handler never outlives the daemon).
//
// On each SIGHUP we ask the daemon to re-read its configured token
// source. The daemon returns (count, error); we log either outcome
// with the scope breakdown but never log the token values themselves.
// On error the previous tokens stay active — a broken token file must
// not lock the operator out of their running daemon.
func handleSIGHUP(hupChan <-chan os.Signal, done <-chan struct{}, d *daemon.Daemon) {
	for {
		select {
		case <-done:
			return
		case <-hupChan:
			count, err := d.ReloadTokens()
			if err != nil {
				logging.Errorf(
					"SIGHUP: token reload failed (%v); previous tokens remain active",
					err,
				)
				continue
			}
			ro, rw, admin := d.TokenScopeCounts()
			logging.Infof(
				"SIGHUP: token set rotated (%d total: read-only=%d, read-write=%d, admin=%d)",
				count, ro, rw, admin,
			)
		}
	}
}

// isNonLoopbackListen mirrors api.addrIsNonLoopback so cmd/niac can pre-
// check the listen address without an import cycle. Kept tiny on purpose
// — the canonical implementation is in internal/api.
func isNonLoopbackListen(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("split host/port %q: %w", addr, err)
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return true, nil
	case "localhost":
		return false, nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback(), nil
	}
	// We don't do DNS resolution here; that path lives in
	// internal/api.addrIsNonLoopback. If we couldn't determine via the
	// fast checks above, assume non-loopback so the user sees the
	// token-required hint.
	return true, nil
}
