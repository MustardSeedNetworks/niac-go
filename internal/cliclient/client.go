// Package cliclient talks to a running NIAC daemon over its HTTPS API.
//
// The CLI used to reach the daemon through a unix socket. That server was
// removed in #1012 as a dead subsystem — the daemon serves over internal/api —
// but the commands built on it were left behind, so `niac monitor`, `logs`,
// `neighbors`, `dump` and `topology` failed every time with "no NIAC simulation
// running" even while simulations were running. They speak to the API now,
// which is the surface the daemon actually serves.
package cliclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is where the daemon listens unless it was told otherwise.
const DefaultBaseURL = "https://127.0.0.1:8445"

const requestTimeout = 30 * time.Second

var (
	// ErrDaemonUnreachable reports that no daemon answered. It carries the
	// address tried, because the usual cause is a daemon on another port.
	ErrDaemonUnreachable = errors.New("no NIAC daemon is answering")
	// ErrUnauthorized reports that the daemon wanted a token this client did
	// not present.
	ErrUnauthorized = errors.New("the daemon requires an API token")
	// ErrRequestFailed reports a non-2xx answer that is not an auth failure.
	ErrRequestFailed = errors.New("daemon request failed")
)

// Config describes how to reach the daemon.
type Config struct {
	// BaseURL defaults to DefaultBaseURL, overridden by NIAC_API_URL.
	BaseURL string
	// Token authenticates against a daemon bound to a non-loopback address.
	// Empty is correct for the default loopback bind. NIAC_API_TOKEN is read
	// when this is empty.
	Token string
	// CertPath is the daemon's self-signed certificate, trusted as a root so
	// the connection is verified rather than waved through. Empty means the
	// system trust store.
	CertPath string
	// Insecure skips verification. It exists for a daemon whose certificate
	// this host cannot see, and says so on the command line.
	Insecure bool
}

// Client is a read-only view of a running daemon.
type Client struct {
	baseURL  string
	token    string
	certPath string
	http     *http.Client
}

// New builds a client, resolving the address, token and trust settings.
func New(cfg Config) (*Client, error) {
	base := cfg.BaseURL
	if base == "" {
		base = os.Getenv("NIAC_API_URL")
	}
	if base == "" {
		base = DefaultBaseURL
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("invalid daemon address %q: %w", base, err)
	}
	token := cfg.Token
	if token == "" {
		token = os.Getenv("NIAC_API_TOKEN")
	}
	transport, err := newTransport(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:  strings.TrimSuffix(base, "/"),
		token:    token,
		certPath: cfg.CertPath,
		http:     &http.Client{Transport: transport},
	}, nil
}

// BaseURL is the address this client talks to, for error messages that need to
// tell an operator where it looked.
func (c *Client) BaseURL() string { return c.baseURL }

func newTransport(cfg Config) (*http.Transport, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	switch {
	case cfg.Insecure:
		// Explicitly asked for on the command line: the daemon's certificate is
		// not visible from this host.
		tlsConfig.InsecureSkipVerify = true
	case cfg.CertPath != "":
		pool, err := poolFromFile(cfg.CertPath)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = pool
	}

	return &http.Transport{TLSClientConfig: tlsConfig}, nil
}

// poolFromFile trusts the daemon's own certificate. The daemon self-signs on
// first start, so this is how a same-host CLI verifies the connection instead
// of skipping the check.
func poolFromFile(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read daemon certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("%w: %s holds no certificate", ErrRequestFailed, path)
	}

	return pool, nil
}

// get fetches one JSON document from the daemon into out.
func (c *Client) get(ctx context.Context, path string, out any) error {
	// A stream stays open for as long as the caller wants; a plain fetch does
	// not, and should give up rather than hang a script.
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	body, err := c.open(ctx, path)
	if err != nil {
		return err
	}
	defer body.Close()

	if err = json.NewDecoder(body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	return nil
}

// open starts a request and hands back the body, which the caller closes. A
// stream endpoint stays open until the context ends.
func (c *Client) open(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", path, err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if _, ok := errors.AsType[*net.OpError](err); ok {
			return nil, fmt.Errorf("%w at %s: is the daemon running?",
				ErrDaemonUnreachable, c.baseURL)
		}

		var certErr *tls.CertificateVerificationError
		if errors.As(err, &certErr) && c.certPath != "" {
			// Naming the certificate that was tried turns "unknown authority"
			// into something an operator can act on: it is usually a stale one
			// left by an earlier run, and --cacert points at the right file.
			return nil, fmt.Errorf("%w: trusted %s, which the daemon at %s does not match: %w",
				ErrRequestFailed, c.certPath, c.baseURL, err)
		}

		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = resp.Body.Close()

		return nil, fmt.Errorf("%w: set NIAC_API_TOKEN", ErrUnauthorized)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_ = resp.Body.Close()

		return nil, fmt.Errorf("%w: %s returned %s", ErrRequestFailed, path, resp.Status)
	}

	return resp.Body, nil
}
