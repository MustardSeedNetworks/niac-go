//go:build windows

package library

// Windows has no portable equivalent of fsync for a directory handle. Draft
// contents are flushed before the atomic rename; returning success here avoids
// reporting a failed mutation after Windows has already committed the rename.
func (l *Library) syncDraftDirectory() error {
	return nil
}
