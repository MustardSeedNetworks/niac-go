package daemon

import (
	"fmt"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func persistInlineConfig(content string) (string, error) {
	path, finish, err := stageInlineSessionConfig(content, defaultSessionID)
	if err != nil {
		return "", err
	}
	finish(true)
	return path, nil
}

// The finish callback owns only this newly created file, not a request's
// ConfigPath. Retaining the root also prevents directory swaps redirecting cleanup.
func stageInlineSessionConfig(content, sessionID string) (string, func(bool), error) {
	if !api.ValidSessionID(sessionID) {
		return "", nil, fmt.Errorf("%w: %q", errInvalidInlineSessionID, sessionID)
	}
	directory, err := inlineConfigDir()
	if err != nil {
		return "", nil, err
	}
	generation, err := newRuntimeGeneration()
	if err != nil {
		return "", nil, err
	}
	name := fmt.Sprintf("_running.%s.%s.inline.yaml", sessionID, generation)
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
