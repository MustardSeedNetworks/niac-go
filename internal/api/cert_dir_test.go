package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver"
)

// foundation's listener refuses a relative certificate directory, but the
// default is "certs" under the working directory, which a developer running
// the daemon from a checkout relies on.
func TestStartDefaultsCertDirToCertsUnderWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	s := NewServer(ServerConfig{Addr: "127.0.0.1:0", SuppressUnauthenticatedWarning: true})
	s.SetDaemonController(&nilDaemonController{})

	if err := s.Start(); err != nil {
		t.Fatalf("Start() with the default certificate directory: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	if _, err := os.Stat(filepath.Join("certs", httpserver.DefaultCertFileName)); err != nil {
		t.Fatalf("the certificate was not written under ./certs: %v", err)
	}
}
