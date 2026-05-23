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

	"github.com/krisarmstrong/niac-go/internal/daemon"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

// Daemon listen port defaults. The TLS port is the Wave 1 canonical for
// niac (mirrors seed:8443 and stem:8444). The legacy HTTP port stays at
// 8080 for `niac daemon --http` so existing deployment scripts keep
// working until they cut over.
const (
	daemonTLSPort          = 8445
	daemonHTTPPort         = 8080
	daemonHTTPRedirectPort = 8044
	daemonDefaultListenIP  = "127.0.0.1"
)

type daemonOptions struct {
	listen              string
	token               string
	storagePath         string
	webhookAllowedHosts []string
	// Wave 1 (TLS-by-default) options.
	certDir    string
	apiToken   string
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
  - Manage multiple simulation sessions

TLS is enabled by default and the daemon listens on 127.0.0.1:8445.
Pass --http to run the legacy HTTP listener on :8080 instead. Binding
to a non-loopback address (e.g. --listen 0.0.0.0) requires an API
token via NIAC_API_TOKEN or --api-token.

Example:
  niac daemon                            # https://127.0.0.1:8445
  niac daemon --listen 0.0.0.0 --api-token $(openssl rand -base64 32)
  niac daemon --http --listen 127.0.0.1  # http://127.0.0.1:8080`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDaemon(options, info)
		},
	}

	daemonCmd.Flags().
		StringVar(&options.listen, "listen", "", "Address to listen on for API and web UI (default depends on --tls)")
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

	// HTTPS is required, no opt-out. The HTTP listener exists only as a
	// 308 redirector. No --tls or --http flags.
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

// resolveDaemonAddrs returns the TLS listen address and the HTTP→HTTPS
// redirector address. HTTPS is always on; no opt-out is supported.
func resolveDaemonAddrs(o *daemonOptions) (string, string, bool) {
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

	redirect := net.JoinHostPort(daemonDefaultListenIP, strconv.Itoa(daemonHTTPRedirectPort))
	return listen, redirect, true
}

// resolveDaemonCertDir returns the cert-dir to use. Explicit --cert-dir
// wins; otherwise NIAC_CERT_DIR; otherwise empty (api package picks
// `certs/`).
func resolveDaemonCertDir(o *daemonOptions) string {
	if o.certDir != "" {
		return o.certDir
	}
	return os.Getenv("NIAC_CERT_DIR")
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

func runDaemon(options *daemonOptions, info versionInfo) error {
	logging.InitColors(true)

	listenAddr, redirectAddr, tlsOn := resolveDaemonAddrs(options)
	token := resolveDaemonAPIToken(options)
	tokenFile := resolveDaemonTokenFile(options)
	certDir := resolveDaemonCertDir(options)

	scheme := "http"
	if tlsOn {
		scheme = "https"
	}

	logging.Infof("Starting NIAC Daemon v%s", info.version)
	logging.Infof("Web UI will be available at %s://%s", scheme, listenAddr)
	if tlsOn && redirectAddr != "" {
		logging.Infof("HTTP→HTTPS redirector listening on %s", redirectAddr)
	}
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
		logging.Warningf("         Use NIAC_API_TOKEN env var for production or network-exposed deployments.")
	}

	cfg := daemon.Config{
		ListenAddr:          listenAddr,
		Token:               token,
		TokenFile:           tokenFile,
		StoragePath:         options.storagePath,
		Version:             info.version,
		Commit:              info.commit,
		BuildTime:           info.date,
		UIBuildHash:         info.uiBuildHash,
		WebhookAllowedHosts: options.webhookAllowedHosts,
		EnableTLS:           tlsOn,
		CertDir:             certDir,
		HTTPRedirectAddr:    redirectAddr,
		TLSPort:             daemonTLSPort,
	}

	// Sanity-check the listen address up front so we fail with the helpful
	// non-loopback-requires-token message before the API server starts and
	// returns the same gate (the duplicate check is intentional defense in
	// depth — the server will refuse too, but printing the friendly hint
	// here surfaces it earlier in the log timeline).
	if nonLoopback, parseErr := isNonLoopbackListen(listenAddr); parseErr == nil && nonLoopback && !authEnabled {
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
			ro, rw := d.TokenScopeCounts()
			logging.Infof(
				"SIGHUP: token set rotated (%d total: read-only=%d, read-write=%d)",
				count, ro, rw,
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
