package support_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/support"
)

// These are the credentials the fixture below carries. They are deliberately
// not the defaults ("public", "niacadmin") -- a redactor that only knew the
// defaults would pass a test built on them and still ship the operator's real
// community string.
const (
	fixtureCommunity     = "s3cret-community"
	fixtureTrapCommunity = "trap-community-9f"
	fixtureAuthPassword  = "authpass-not-default"
	fixturePrivPassword  = "privpass-not-default"
	fixtureFTPPassword   = "ftp-password-42"
	fixtureIncludeCommun = "include-community-7"
	fixtureBearerToken   = "b3arer-t0ken-abcdef0123456789"
)

const bundleFixtureConfig = `devices:
  - name: core-sw
    type: switch
    mac: "02:00:00:00:00:01"
    ips:
      - "10.10.0.1"
    snmp_agent:
      community: "` + fixtureCommunity + `"
      sysname: "core-sw"
      community_includes:
        - community: "` + fixtureIncludeCommun + `"
          walk_file: "alt.walk"
      traps:
        enabled: true
        receivers:
          - "10.10.0.250:162"
        community: "` + fixtureTrapCommunity + `"
    snmpv3:
      enabled: true
      users:
        - username: "operator"
          auth_protocol: "sha"
          auth_password: "` + fixtureAuthPassword + `"
          priv_protocol: "aes"
          priv_password: "` + fixturePrivPassword + `"
    ftp:
      enabled: true
      users:
        - username: "operator"
          password: "` + fixtureFTPPassword + `"
`

// bundleFixtureLog is the shape the daemon's own log takes when it echoes a
// request: the secret arrives as free text no structural redaction can reach.
const bundleFixtureLog = `time=2026-09-08T10:00:00Z level=INFO msg="request" auth="Bearer ` +
	fixtureBearerToken + `"
time=2026-09-08T10:00:01Z level=INFO msg="snmp get" community="` + fixtureCommunity + `"
time=2026-09-08T10:00:02Z level=INFO msg="listening" addr=0.0.0.0:8445
`

func writeBundleFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "office.yaml")
	if err := os.WriteFile(configPath, []byte(bundleFixtureConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// community_includes names a walk the loader resolves relative to the
	// config, so the fixture has to carry one.
	if err := os.WriteFile(filepath.Join(dir, "alt.walk"),
		[]byte(".1.3.6.1.2.1.1.1.0 = STRING: core-sw\n"), 0o600); err != nil {
		t.Fatalf("write walk: %v", err)
	}
	logPath := filepath.Join(dir, "niac.log")
	if err := os.WriteFile(logPath, []byte(bundleFixtureLog), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return configPath, logPath
}

// archiveEntries reads a gzipped tar into a name→content map.
func archiveEntries(t *testing.T, data []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer func() { _ = gz.Close() }()

	out := map[string]string{}
	tr := tar.NewReader(gz)
	for {
		header, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			t.Fatalf("read tar: %v", nextErr)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		body, readErr := io.ReadAll(tr)
		if readErr != nil {
			t.Fatalf("read %s: %v", header.Name, readErr)
		}
		out[header.Name] = string(body)
	}
	return out
}

// The acceptance clause for P5-3: the bundle carries no token and no community
// string. Asserted over the whole archive, so a secret that survives in a file
// this test does not name still fails it.
func TestBundleCarriesNoCredential(t *testing.T) {
	configPath, logPath := writeBundleFixture(t)

	var archive bytes.Buffer
	options := support.BundleOptions{
		Configs: []string{configPath},
		LogPath: logPath,
		Token:   fixtureBearerToken,
	}
	if err := support.WriteBundle(options, &archive); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}

	entries := archiveEntries(t, archive.Bytes())
	whole := strings.Join(slicesValues(entries), "\n")
	secrets := map[string]string{
		"snmp community":       fixtureCommunity,
		"trap community":       fixtureTrapCommunity,
		"community_includes":   fixtureIncludeCommun,
		"snmpv3 auth password": fixtureAuthPassword,
		"snmpv3 priv password": fixturePrivPassword,
		"ftp password":         fixtureFTPPassword,
		"daemon bearer token":  fixtureBearerToken,
	}
	for what, secret := range secrets {
		if strings.Contains(whole, secret) {
			t.Errorf("bundle leaks the %s (%q)", what, secret)
		}
	}
}

// Redaction must not silently empty the bundle: the diagnostic content an
// operator sends support has to survive it.
func TestBundleKeepsDiagnosticContent(t *testing.T) {
	configPath, logPath := writeBundleFixture(t)

	var archive bytes.Buffer
	options := support.BundleOptions{
		Configs:    []string{configPath},
		LogPath:    logPath,
		Token:      fixtureBearerToken,
		Interfaces: []string{"en0 10.10.0.5/24"},
	}
	if err := support.WriteBundle(options, &archive); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}

	entries := archiveEntries(t, archive.Bytes())
	for _, name := range []string{"manifest.json", "configs/office.yaml", "logs/niac.log", "interfaces.txt"} {
		if _, ok := entries[name]; !ok {
			t.Errorf("bundle is missing %s (has %v)", name, sortedKeys(entries))
		}
	}
	if !strings.Contains(entries["configs/office.yaml"], "core-sw") {
		t.Error("redacted config lost the device name")
	}
	if !strings.Contains(entries["logs/niac.log"], "listening") {
		t.Error("redacted log lost its non-secret lines")
	}
	if !strings.Contains(entries["manifest.json"], runtime.GOOS) {
		t.Error("manifest does not record the platform")
	}
}

// A bundle must never carry the daemon's private key, whatever else it
// collects.
func TestBundleExcludesPrivateKeys(t *testing.T) {
	configPath, logPath := writeBundleFixture(t)
	certDir := t.TempDir()
	const keyMaterial = "-----BEGIN PRIVATE KEY-----\nMIIEv-fixture\n-----END PRIVATE KEY-----\n"
	if err := os.WriteFile(filepath.Join(certDir, "server.key"), []byte(keyMaterial), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(certDir, "server.crt"),
		[]byte("-----BEGIN CERTIFICATE-----\n"),
		0o644,
	); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	var archive bytes.Buffer
	options := support.BundleOptions{
		Configs: []string{configPath},
		LogPath: logPath,
		CertDir: certDir,
	}
	if err := support.WriteBundle(options, &archive); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}

	whole := strings.Join(slicesValues(archiveEntries(t, archive.Bytes())), "\n")
	if strings.Contains(whole, "PRIVATE KEY") {
		t.Error("bundle carries private key material")
	}
}

func slicesValues(entries map[string]string) []string {
	out := make([]string, 0, len(entries))
	for _, name := range sortedKeys(entries) {
		out = append(out, entries[name])
	}
	return out
}

func sortedKeys(entries map[string]string) []string {
	out := make([]string, 0, len(entries))
	for name := range entries {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
