//go:build windows

package tokenstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// maxTokenFileBytes bounds what is read; a token file is a few short lines.
const maxTokenFileBytes = 1 << 20

// readTokenFile opens the file through securedir, which refuses it unless it
// and every directory above it belong to SYSTEM or Administrators, no one else
// can write, delete or re-ACL it, and nothing on the path is a link. That is
// the Windows form of the Unix mode check, which has nothing to read here.
//
// securedir judges who can change the file, not who can read it. The token file
// belongs in a directory whose ACL admits only SYSTEM and Administrators (the
// installer's %ProgramData%\NiAC), which is what keeps other accounts from
// reading it.
func readTokenFile(path string) ([]byte, error) {
	return readPrivateFile(path, securedir.Options{})
}

// readPrivateFile is readTokenFile with the trusted principals chosen by the
// caller. The service trusts SYSTEM and Administrators only; tests add the
// account that owns their temporary files.
func readPrivateFile(path string, opts securedir.Options) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve token file %q: %w", path, err)
	}
	dir, err := securedir.Canonical(filepath.Dir(abs), opts)
	if err != nil {
		return nil, fmt.Errorf("token file %q: %w", path, err)
	}
	file, err := securedir.OpenFile(filepath.Join(dir, filepath.Base(abs)), os.O_RDONLY, opts)
	if err != nil {
		return nil, fmt.Errorf("token file %q is not private to SYSTEM and Administrators: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, maxTokenFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read token file %q: %w", path, err)
	}
	if len(raw) > maxTokenFileBytes {
		return nil, fmt.Errorf("token file %q is larger than %d bytes", path, maxTokenFileBytes)
	}
	return raw, nil
}
