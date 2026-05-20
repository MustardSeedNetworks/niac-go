package api

// tls.go provides the self-signed certificate generation and HTTP→HTTPS
// redirect helpers used when niac binds with TLS enabled (the new default,
// Wave 1). The cert template mirrors seed's: single-tier self-signed CA so
// `niac install-ca` can install the same file into the OS trust store.

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// rsaKeyBits is the RSA modulus size for the self-signed certificate
// niac generates on first start. 4096 matches seed's #533 hardening and
// keeps the cert acceptable to modern browsers / system trust stores.
const rsaKeyBits = 4096

// defaultCertDir is the directory the self-signed cert+key are written to
// when --cert-dir is not set. Relative to the working directory so a
// developer running `./niac daemon` from the repo root sees the artifact.
const defaultCertDir = "certs"

// defaultCertFileName / defaultKeyFileName are the filenames used inside
// the cert directory. install-ca defaults to <cert-dir>/server.crt.
const (
	defaultCertFileName = "server.crt"
	defaultKeyFileName  = "server.key"
)

// defaultTLSPort is the canonical port niac binds when TLS is enabled
// (mirrors seed:8443 and stem:8444 — niac gets 8445). cmd/niac mirrors
// this via daemonTLSPort; the duplication is intentional so the api
// package doesn't have to import cmd-level state.
const defaultTLSPort = 8445

// httpsStandardPort is RFC 1700's well-known https port. Used by the
// HTTP→HTTPS redirect handler to decide whether to emit an explicit
// :port suffix in the Location header.
const httpsStandardPort = 443

// EnsureSelfSignedCert generates a self-signed certificate at the configured
// paths if one does not already exist. Reuses existing files when both are
// present. Returns the (possibly newly created) certPath and keyPath.
//
// The cert is a single-tier self-signed CA: it acts as both the root
// (Issuer == Subject) and the leaf the TLS listener serves. This lets
// `niac install-ca` install the same file into the OS trust store so
// browsers stop showing the self-signed warning. Without IsCA=true and
// KeyUsageCertSign, OS trust stores will reject the cert as not eligible
// to act as a root.
func EnsureSelfSignedCert(certPath, keyPath string) (string, string, error) {
	return ensureSelfSignedCert(certPath, keyPath)
}

// ensureSelfSignedCert is the unexported worker called via the exported
// wrapper (kept exported for test convenience and future use by the
// install-ca command).
func ensureSelfSignedCert(certPath, keyPath string) (string, string, error) {
	if certPath == "" || keyPath == "" {
		return "", "", errors.New("cert and key paths must both be set")
	}

	// Reuse existing cert+key when both are present on disk.
	if _, certErr := os.Stat(certPath); certErr == nil {
		if _, keyErr := os.Stat(keyPath); keyErr == nil {
			return certPath, keyPath, nil
		}
	}

	// Ensure the parent directory exists with safe perms.
	certDir := filepath.Dir(certPath)
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create certs directory: %w", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return "", "", fmt.Errorf("generate RSA key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NIAC"},
			CommonName:   "NIAC Self-Signed",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost", "niac.local"},
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	// Cert + key both 0o600. niac runs as a single user; there is no need
	// to expose the cert beyond that user. `niac install-ca` runs as the
	// same user (or root) and reads the file directly.
	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", fmt.Errorf("create cert file: %w", err)
	}
	defer func() { _ = certOut.Close() }()
	if encodeErr := pem.Encode(certOut, &pem.Block{Type: pemCertBlockType, Bytes: certDER}); encodeErr != nil {
		return "", "", fmt.Errorf("encode certificate PEM: %w", encodeErr)
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", fmt.Errorf("create key file: %w", err)
	}
	defer func() { _ = keyOut.Close() }()
	if keyEncodeErr := pem.Encode(
		keyOut,
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)},
	); keyEncodeErr != nil {
		return "", "", fmt.Errorf("encode private key PEM: %w", keyEncodeErr)
	}

	return certPath, keyPath, nil
}

// DefaultCertPaths returns the default cert+key paths for a given certDir.
// When certDir is empty, the package default (`certs/`) is used.
func DefaultCertPaths(certDir string) (string, string) {
	if certDir == "" {
		certDir = defaultCertDir
	}
	return filepath.Join(certDir, defaultCertFileName),
		filepath.Join(certDir, defaultKeyFileName)
}

// httpToHTTPSRedirectHandler returns an [http.Handler] that permanently
// redirects every request to the equivalent HTTPS URL on the given target
// port. Extracted so unit tests can exercise it without bringing up a
// real listener.
func httpToHTTPSRedirectHandler(httpsPort int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the source port from r.Host so localhost:8044 → localhost
		// before re-joining with the HTTPS port. Without this we'd emit
		// https://localhost:8044:8445 which most clients reject.
		host := r.Host
		if colonPos := strings.LastIndex(host, ":"); colonPos != -1 {
			host = host[:colonPos]
		}

		// r.URL.RequestURI() returns just /path?query regardless of how
		// the client encoded the request line (origin-form vs absolute-
		// form). r.RequestURI carries the raw client value and may be a
		// full URL — which would double-up scheme+host when concatenated
		// and produces a broken Location header.
		requestURI := r.URL.RequestURI()

		var httpsURL string
		if httpsPort == httpsStandardPort {
			httpsURL = "https://" + host + requestURI
		} else {
			httpsURL = "https://" + net.JoinHostPort(host, strconv.Itoa(httpsPort)) + requestURI
		}

		// 308 Permanent Redirect (RFC 7538) preserves method and body so
		// POST/PUT/DELETE survive the redirect. 301 silently downgrades
		// state-changing requests to GET.
		http.Redirect(w, r, httpsURL, http.StatusPermanentRedirect)
	})
}
