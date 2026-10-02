package daemon

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// inlineConfigName names the file a session's inline ConfigData is written to.
// Keying it on the runtime generation, which the recovery record persists,
// lets the session's owner find the file again after a restart. A request's
// own ConfigPath can never carry this name: the generation is minted at start.
func inlineConfigName(sessionID, generation string) string {
	return fmt.Sprintf("_running.%s.%s.inline.yaml", sessionID, generation)
}

// The finish callback owns only this newly created file, not a request's
// ConfigPath. Retaining the root also prevents directory swaps redirecting cleanup.
func stageInlineSessionConfig(content, sessionID, generation string) (string, func(bool), error) {
	if !api.ValidSessionID(sessionID) {
		return "", nil, fmt.Errorf("%w: %q", errInvalidInlineSessionID, sessionID)
	}
	if !validRuntimeGeneration(generation) {
		return "", nil, fmt.Errorf("invalid runtime generation %q", generation)
	}
	directory, err := inlineConfigDir()
	if err != nil {
		return "", nil, err
	}
	name := inlineConfigName(sessionID, generation)
	path, err := filepath.Abs(filepath.Join(directory, name))
	if err != nil {
		return "", nil, err
	}
	root, err := openStateRoot(directory)
	if err != nil {
		return "", nil, err
	}
	if err = writeRootStateFile(root, name, []byte(content)); err != nil {
		_ = root.Close()
		return "", nil, err
	}
	return path, func(committed bool) {
		defer func() { _ = root.Close() }()
		if !committed {
			if removeErr := root.Remove(name); removeErr != nil {
				logging.Warningf("Could not remove uncommitted inline configuration: %v", removeErr)
			}
		}
	}, nil
}

// removeInlineConfig deletes the inline file a session generation wrote, once
// that generation is replaced or stopped. A generation that started from a
// ConfigPath or a scenario wrote none, so a missing file is not an error. The
// configs directory is also a config root, so a running session may have been
// started from this file by path; it then stays until that session lets go.
func (d *Daemon) removeInlineConfig(sessionID, generation string) {
	directory, err := inlineConfigDir()
	if err != nil {
		logging.Warningf("Could not locate inline configuration for session %s: %v", sessionID, err)
		return
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			logging.Warningf("Could not open inline configuration directory: %v", err)
		}
		return
	}
	defer func() { _ = root.Close() }()
	name := inlineConfigName(sessionID, generation)
	info, err := root.Stat(name)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			logging.Warningf("Could not inspect inline configuration for session %s: %v", sessionID, err)
		}
		return
	}
	if d.inlineConfigInUse(info) {
		return
	}
	if removeErr := root.Remove(name); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		logging.Warningf("Could not remove inline configuration for session %s: %v", sessionID, removeErr)
	}
}

func (d *Daemon) inlineConfigInUse(info fs.FileInfo) bool {
	for _, sim := range d.sessions.sessions {
		if sim.ConfigPath == "" {
			continue
		}
		if other, err := os.Stat(sim.ConfigPath); err == nil && os.SameFile(info, other) {
			return true
		}
	}
	return false
}
