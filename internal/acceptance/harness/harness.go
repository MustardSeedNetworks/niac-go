// Package harness drives a shipped NIAC binary over its authenticated API.
//
// Every other test in this repository links the daemon into the test binary,
// so none of them can see what the release actually ships: an embedded UI, the
// version ldflags, the TLS listener, the bearer and CSRF middleware, and the
// exit behaviour all only exist in the built artifact. This package starts
// that artifact as a subprocess and talks to it the way an operator's script
// would.
package harness

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// BinaryEnv names the release-built binary the harness drives. There is no
// fallback to a `go build` of the working tree: the point of this package is
// to exercise what a release ships, and a silently substituted development
// build would report a pass the release had not earned.
const BinaryEnv = "NIAC_ACCEPTANCE_BINARY"

const (
	// startTimeout bounds the wait for the daemon's first answer. A release
	// build has an embedded UI to serve and a certificate to generate on
	// first run, so this is generous compared with the sub-second steady
	// state.
	startTimeout = 30 * time.Second
	// readyPoll is short enough that a fast start is not padded and long
	// enough that a slow one does not spin.
	readyPoll = 100 * time.Millisecond
	// tokenBytes is a 256-bit bearer, the size the daemon's own help
	// suggests minting.
	tokenBytes = 32
)

// Daemon is a running niac process and an authenticated client for it.
type Daemon struct {
	Client  *cliclient.Client
	BaseURL string
	Token   string

	command *exec.Cmd
	logPath string
}

// Options configures one daemon. The zero value reads the binary path from
// BinaryEnv and puts every piece of state in a temporary directory.
type Options struct {
	// BinaryPath overrides BinaryEnv.
	BinaryPath string
	// Root holds the daemon's certificates, configs and library. The caller
	// owns its lifetime; t.TempDir() is the usual answer.
	Root string
	// AttachmentPolicies are --attachment-policy values, in the daemon's own
	// spelling: INTERFACE=direct, INTERFACE=access:VLAN or
	// INTERFACE=trunk:VLAN,... A routed attachment is refused without one, so
	// a run that starts a scenario on a real interface has to name it.
	AttachmentPolicies []string
}

// Start launches the daemon and waits until it answers. The caller must call
// Stop, which is also what collects the daemon's log on failure.
func Start(ctx context.Context, options Options) (*Daemon, error) {
	binary, err := resolveBinary(options.BinaryPath)
	if err != nil {
		return nil, err
	}
	root, err := resolveRoot(options.Root)
	if err != nil {
		return nil, err
	}
	port, err := freeLoopbackPort(ctx)
	if err != nil {
		return nil, err
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("127.0.0.1:%d", port)
	logPath := filepath.Join(root, "daemon.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create daemon log: %w", err)
	}
	defer logFile.Close()

	arguments := []string{"daemon",
		"--listen", address,
		"--storage", "disabled",
		"--cert-dir", filepath.Join(root, "certs"),
	}
	for _, policy := range options.AttachmentPolicies {
		arguments = append(arguments, "--attachment-policy", policy)
	}
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Env = append(os.Environ(),
		"NIAC_API_TOKEN="+token,
		"NIAC_CONFIGS_DIR="+filepath.Join(root, "configs"),
		"NIAC_LIBRARY_ROOT="+filepath.Join(root, "library"),
	)
	command.Stdout = logFile
	command.Stderr = logFile
	if err = command.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", binary, err)
	}

	daemon := &Daemon{
		BaseURL: "https://" + address,
		Token:   token,
		command: command,
		logPath: logPath,
	}
	// The daemon generates its own certificate on first run, so there is
	// nothing to trust before it starts; the listener is on loopback and the
	// token is minted here, so skipping verification exposes nothing.
	daemon.Client, err = cliclient.New(cliclient.Config{
		BaseURL: daemon.BaseURL, Token: token, Insecure: true,
	})
	if err != nil {
		_ = daemon.Stop()
		return nil, err
	}
	if err = daemon.waitReady(ctx); err != nil {
		_ = daemon.Stop()
		return nil, err
	}

	return daemon, nil
}

// Stop ends the daemon and waits for it. A daemon that has already exited is
// not an error: the test may have stopped it deliberately.
func (d *Daemon) Stop() error {
	if d.command == nil || d.command.Process == nil {
		return nil
	}
	_ = d.command.Process.Kill()
	err := d.command.Wait()
	d.command = nil
	// Kill makes a non-zero exit the expected outcome, not a failure.
	if _, killed := errors.AsType[*exec.ExitError](err); killed {
		return nil
	}

	return err
}

// Log returns everything the daemon wrote, which is the only account of why a
// failed start failed.
func (d *Daemon) Log() string {
	contents, err := os.ReadFile(d.logPath)
	if err != nil {
		return "(no daemon log: " + err.Error() + ")"
	}

	return string(contents)
}

// waitReady polls the unauthenticated version route, which is the same signal
// the deployment check uses.
func (d *Daemon) waitReady(ctx context.Context) error {
	deadline := time.Now().Add(startTimeout)
	var last error
	for time.Now().Before(deadline) {
		if d.command.ProcessState != nil {
			return fmt.Errorf("daemon exited before answering:\n%s", d.Log())
		}
		version, err := d.Client.Version(ctx)
		if err == nil && version.Version != "" {
			return nil
		}
		last = err
		time.Sleep(readyPoll)
	}

	return fmt.Errorf("daemon did not answer within %s: %w\n%s", startTimeout, last, d.Log())
}

func resolveBinary(override string) (string, error) {
	binary := override
	if binary == "" {
		binary = os.Getenv(BinaryEnv)
	}
	if binary == "" {
		return "", fmt.Errorf("set %s to a release-built niac binary", BinaryEnv)
	}
	absolute, err := filepath.Abs(binary)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", binary, err)
	}
	if _, err = os.Stat(absolute); err != nil {
		return "", fmt.Errorf("%s: %w", BinaryEnv, err)
	}

	return absolute, nil
}

func resolveRoot(root string) (string, error) {
	if root != "" {
		return root, nil
	}

	return os.MkdirTemp("", "niac-acceptance-*")
}

// freeLoopbackPort asks the kernel for a port and releases it. The daemon
// races anything else on the host for it, which is why the harness never
// hardcodes 8445: a stray daemon there would otherwise answer the test.
func freeLoopbackPort(ctx context.Context) (int, error) {
	var config net.ListenConfig
	listener, err := config.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve a port: %w", err)
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("reserved %s, which is not a TCP address", listener.Addr())
	}

	return address.Port, nil
}

func randomToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("mint a token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}
