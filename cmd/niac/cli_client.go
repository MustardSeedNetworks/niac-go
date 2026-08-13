package main

import (
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// newCLIClient builds the API client the read-only commands share.
//
// The daemon self-signs its certificate on first start, so a same-host CLI can
// verify the connection by trusting that certificate rather than skipping the
// check. --insecure remains for a daemon whose certificate this host cannot
// see, and has to be asked for.
func newCLIClient(apiURL string, insecure bool) (*cliclient.Client, error) {
	return cliclient.New(cliclient.Config{
		BaseURL:  apiURL,
		CertPath: daemonCertPath(),
		Insecure: insecure,
	})
}

// daemonCertPath finds the certificate the local daemon serves, if this host is
// the one running it. An empty result falls back to the system trust store,
// which is right for a daemon fronted by a real certificate.
func daemonCertPath() string {
	certPath, _ := api.DefaultCertPaths(defaultCertDir())
	if !filepath.IsAbs(certPath) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		certPath = filepath.Join(home, ".niac", certPath)
	}
	if _, err := os.Stat(certPath); err != nil {
		return ""
	}

	return certPath
}
