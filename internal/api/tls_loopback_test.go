package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"testing"
	"time"
)

// The daemon advertises https://127.0.0.1:8445 as where it listens, and its own
// help text says so - but the certificate it generated covered only two DNS
// names, so nothing could verify a connection to that address. A browser saw an
// unfixable warning and the CLI could not trust the daemon on the same host.
func TestGeneratedCertificateCoversTheAddressTheDaemonAdvertises(t *testing.T) {
	certPath, _ := generateIntoTempDir(t)
	cert := parseCert(t, certPath)

	for _, ip := range []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")} {
		if err := cert.VerifyHostname(ip.String()); err != nil {
			t.Errorf("certificate does not cover %s: %v", ip, err)
		}
	}
}

// The names it already carried keep working.
func TestGeneratedCertificateKeepsItsHostnames(t *testing.T) {
	certPath, _ := generateIntoTempDir(t)
	cert := parseCert(t, certPath)

	for _, name := range []string{"localhost", "niac.local"} {
		if err := cert.VerifyHostname(name); err != nil {
			t.Errorf("certificate does not cover %s: %v", name, err)
		}
	}
}

// A certificate written before loopback was covered is replaced rather than
// reused, or the daemon keeps serving one nothing can verify.
func TestACertificateMissingLoopbackIsReplaced(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := DefaultCertPaths(dir)
	if _, _, err := EnsureSelfSignedCert(certPath, keyPath); err != nil {
		t.Fatalf("first generation: %v", err)
	}
	// Stand in for the old certificate by stripping its loopback coverage.
	writeCertWithoutIPs(t, certPath, keyPath)

	if _, _, err := EnsureSelfSignedCert(certPath, keyPath); err != nil {
		t.Fatalf("regeneration: %v", err)
	}
	if err := parseCert(t, certPath).VerifyHostname("127.0.0.1"); err != nil {
		t.Errorf("stale certificate was reused: %v", err)
	}
}

func generateIntoTempDir(t *testing.T) (string, string) {
	t.Helper()
	certPath, keyPath := DefaultCertPaths(t.TempDir())
	gotCert, gotKey, err := EnsureSelfSignedCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("EnsureSelfSignedCert: %v", err)
	}

	return gotCert, gotKey
}

func parseCert(t *testing.T, path string) *x509.Certificate {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("%s holds no PEM block", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	return cert
}

// writeCertWithoutIPs replaces the certificate with one carrying only the DNS
// names, which is what every daemon installed before this change is serving.
func writeCertWithoutIPs(t *testing.T, certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"NIAC"}, CommonName: "NIAC Self-Signed"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "niac.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
	)
	if err = os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
}
