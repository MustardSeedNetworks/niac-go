//go:build darwin || linux

package library

import "fmt"

func (l *Library) syncDraftDirectory() error {
	dir, err := l.draftRoot.Open(".")
	if err != nil {
		return fmt.Errorf("open drafts directory for sync: %w", err)
	}
	defer func() { _ = dir.Close() }()
	if syncErr := dir.Sync(); syncErr != nil {
		return fmt.Errorf("sync drafts directory: %w", syncErr)
	}
	return nil
}
