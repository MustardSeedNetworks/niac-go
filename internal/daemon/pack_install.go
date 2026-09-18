package daemon

import (
	"io"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
)

// InstallPack extracts a content bundle into root. It is the one production
// path that writes library content.
//
// `niac content install` and POST /api/v1/library/install each called
// content.Extract themselves, so a CLI install landed in the very tree a
// running daemon had already opened, with nothing serialising the two and
// nothing telling the daemon its library had changed underneath it.
//
// The API handler reaches this through api.ServerConfig.InstallPack, which
// the daemon fills with its own method; the CLI reaches it directly, and only
// while it holds the single-instance lock a running daemon denies it.
func InstallPack(bundle io.Reader, root string, opts content.ExtractOptions) (content.Manifest, error) {
	return content.Extract(bundle, root, opts)
}

// InstallPack installs a bundle into root on behalf of the daemon.
//
// Extraction decides per file whether to overwrite or preserve, so two
// installs running at once against one root interleave those decisions and
// each land half-applied. The daemon owns the tree, so the daemon is where
// they queue.
func (d *Daemon) InstallPack(bundle io.Reader, root string, opts content.ExtractOptions) (content.Manifest, error) {
	d.packInstallMu.Lock()
	defer d.packInstallMu.Unlock()

	return InstallPack(bundle, root, opts)
}
