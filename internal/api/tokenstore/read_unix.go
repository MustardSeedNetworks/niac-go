//go:build !windows

package tokenstore

import (
	"fmt"
	"os"
)

// readTokenFile refuses a file with any group or other permission bit, then
// reads it. The 0o077 mask tolerates the owner-execute bit (0o100), which
// would be unusual but harmless on a regular file.
func readTokenFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat token file %q: %w", path, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: %q has mode %o", ErrTokenFileWorldReadable, path, info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token file %q: %w", path, err)
	}
	return raw, nil
}
