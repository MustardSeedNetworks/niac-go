package api

// tls.go provides the self-signed certificate generation used when niac
// binds with TLS enabled (HTTPS-only since Wave 1). The cert template
// mirrors seed's: single-tier self-signed CA so `niac install-ca` can
// install the same file into the OS trust store.

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
	"os"
	"path/filepath"
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

	// Reuse existing cert+key when both are present and still cover what the
	// daemon serves. A certificate written before loopback was covered cannot
	// verify the address the daemon advertises, so it is replaced rather than
	// served for another year.
	if _, certErr := os.Stat(certPath); certErr == nil {
		if _, keyErr := os.Stat(keyPath); keyErr == nil && coversLoopback(certPath) {
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
		// The daemon tells operators it listens on https://127.0.0.1:8445, so
		// the certificate has to cover that address. Without these a browser
		// shows a warning nothing can clear, and a client on the same host
		// cannot verify the daemon even holding its certificate.
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
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

// coversLoopback reports whether an existing certificate can verify the address
// the daemon advertises. It is how a certificate generated before loopback was
// covered gets replaced instead of reused.
func coversLoopback(certPath string) bool {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	return cert.VerifyHostname("127.0.0.1") == nil
}
