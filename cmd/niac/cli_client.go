package main

import (
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// newCLIClient builds the API client the read-only commands share.
//
// The daemon self-signs its certificate on first start, so a CLI on the same
// host verifies the connection by trusting that certificate rather than
// skipping the check. --cacert names it outright; otherwise the usual places
// are searched. --insecure remains for a daemon whose certificate this host
// cannot see, and has to be asked for.
func newCLIClient(apiURL, caCert string, insecure bool) (*cliclient.Client, error) {
	if caCert == "" {
		caCert = findDaemonCert()
	}

	return cliclient.New(cliclient.Config{
		BaseURL:  apiURL,
		CertPath: caCert,
		Insecure: insecure,
	})
}

// findDaemonCert looks where a local daemon keeps its certificate. The daemon
// resolves the same relative `certs/` directory against its own working
// directory, which for the packaged service is its data directory - so a CLI
// run from elsewhere has to check the data directory too.
//
// An empty result falls back to the system trust store, which is right for a
// daemon behind a certificate a real authority issued.
func findDaemonCert() string {
	certFile, _ := api.DefaultCertPaths(defaultCertDir())
	if filepath.IsAbs(certFile) {
		return existingFile(certFile)
	}

	// Order matters: an installed daemon's certificate beats one left in the
	// working directory by an earlier run. A stale certificate with the same
	// subject fails verification in a way that reads like a configuration
	// error, so the current directory is consulted last.
	home, _ := os.UserHomeDir()
	for _, dir := range []string{packagedDataDir, filepath.Join(home, ".niac"), "."} {
		if found := existingFile(filepath.Join(dir, certFile)); found != "" {
			return found
		}
	}

	return ""
}

// packagedDataDir is where the packaged service keeps its state, and therefore
// where it resolves its relative certs/ directory.
const packagedDataDir = "/var/lib/niac"

func existingFile(path string) string {
	if _, err := os.Stat(path); err != nil {
		return ""
	}

	return path
}
