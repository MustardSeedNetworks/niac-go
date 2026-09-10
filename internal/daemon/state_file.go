package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

const stateDirectoryMode = 0o750

func openStateRoot(directory string) (*os.Root, error) {
	if err := os.MkdirAll(directory, stateDirectoryMode); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	return os.OpenRoot(directory)
}

func writeStateFile(directory, name string, data []byte) error {
	root, err := openStateRoot(directory)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return writeRootStateFile(root, name, data)
}

// All names are relative to an operator-selected root. Atomic replacement and
// rollback use that same capability, even if a directory is moved during a write.
func writeRootStateFile(root *os.Root, name string, data []byte) error {
	directory := filepath.Dir(name)
	if err := root.MkdirAll(directory, stateDirectoryMode); err != nil {
		return fmt.Errorf("create state subdirectory: %w", err)
	}
	generation, err := newRuntimeGeneration()
	if err != nil {
		return err
	}
	tempName := filepath.Join(directory, ".state-"+generation)
	temp, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, recoveryFileMode)
	if err != nil {
		return fmt.Errorf("create state file: %w", err)
	}
	defer func() { _ = root.Remove(tempName) }()
	if _, err = temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write state file: %w", err)
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync state file: %w", err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close state file: %w", err)
	}
	if err = root.Rename(tempName, name); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	return nil
}

func removeStateFile(directory, name string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return root.Remove(name)
}
