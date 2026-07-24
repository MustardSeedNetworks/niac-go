package daemon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	activeSimulationSchemaVersion = 1
	activeSimulationFileName      = "active-simulation.json"
	maxActiveSimulationStateSize  = 64 * 1024
	recoveryFileMode              = 0o600
	recoveryStateRecovered        = "recovered"
	recoveryStateFailed           = "failed"
)

type activeSimulationState struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Request       api.SimulationRequest `json:"request"`
	SavedAt       time.Time             `json:"savedAt"`
}

// DefaultRecoveryPath returns the platform-aware daemon recovery record path.
func DefaultRecoveryPath() string {
	return filepath.Join(filepath.Dir(library.DefaultRoot()), "state", activeSimulationFileName)
}

func (d *Daemon) persistActiveSimulation(request api.SimulationRequest) error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	state := activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		Request:       request,
		SavedAt:       time.Now().UTC(),
	}
	data, marshalErr := json.MarshalIndent(state, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("encode recovery state: %w", marshalErr)
	}
	data = append(data, '\n')
	return writeRecoveryState(d.cfg.RecoveryPath, data)
}

func writeRecoveryState(path string, data []byte) error {
	directory := filepath.Dir(path)
	if mkdirErr := os.MkdirAll(directory, 0o750); mkdirErr != nil {
		return fmt.Errorf("create recovery directory: %w", mkdirErr)
	}
	temp, createErr := os.CreateTemp(directory, ".active-simulation-")
	if createErr != nil {
		return fmt.Errorf("create recovery state: %w", createErr)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if chmodErr := temp.Chmod(recoveryFileMode); chmodErr != nil {
		_ = temp.Close()
		return fmt.Errorf("secure recovery state: %w", chmodErr)
	}
	if _, writeErr := temp.Write(data); writeErr != nil {
		_ = temp.Close()
		return fmt.Errorf("write recovery state: %w", writeErr)
	}
	if syncErr := temp.Sync(); syncErr != nil {
		_ = temp.Close()
		return fmt.Errorf("sync recovery state: %w", syncErr)
	}
	if closeErr := temp.Close(); closeErr != nil {
		return fmt.Errorf("close recovery state: %w", closeErr)
	}
	if renameErr := os.Rename(tempPath, path); renameErr != nil {
		return fmt.Errorf("replace recovery state: %w", renameErr)
	}
	return nil
}

func (d *Daemon) clearActiveSimulation() error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	removeErr := os.Remove(d.cfg.RecoveryPath)
	if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return removeErr
	}
	d.recovery = nil
	return nil
}

func (d *Daemon) recoverActiveSimulation(entitlements api.SimulationEntitlements) {
	if d.cfg.RecoveryPath == "" {
		return
	}
	state, readErr := readRecoveryState(d.cfg.RecoveryPath)
	if errors.Is(readErr, fs.ErrNotExist) {
		return
	}
	attemptedAt := time.Now().UTC()
	if readErr != nil {
		d.setRecoveryFailure(attemptedAt, readErr)
		return
	}
	if startErr := d.StartSimulation(state.Request, entitlements); startErr != nil {
		d.setRecoveryFailure(attemptedAt, startErr)
		return
	}
	d.mu.Lock()
	d.recovery = &api.SimulationRecovery{
		State:       recoveryStateRecovered,
		Message:     "Active simulation restored from persisted launch intent.",
		AttemptedAt: attemptedAt,
	}
	d.mu.Unlock()
	logging.Successf("✓ Active simulation recovered on %s", state.Request.Interface)
}

func (d *Daemon) setRecoveryFailure(attemptedAt time.Time, recoveryErr error) {
	message := fmt.Sprintf(
		"Recovery failed: %v. Correct or remove %s before restarting.",
		recoveryErr,
		d.cfg.RecoveryPath,
	)
	d.mu.Lock()
	d.recovery = &api.SimulationRecovery{
		State:       recoveryStateFailed,
		Message:     message,
		AttemptedAt: attemptedAt,
	}
	d.mu.Unlock()
	logging.Warningf("%s", message)
}

func readRecoveryState(path string) (activeSimulationState, error) {
	info, statErr := os.Lstat(path)
	if statErr != nil {
		return activeSimulationState{}, statErr
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return activeSimulationState{}, errors.New("recovery state cannot be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return activeSimulationState{}, errors.New("recovery state is not a regular file")
	}
	if info.Size() > maxActiveSimulationStateSize {
		return activeSimulationState{}, errors.New("recovery state exceeds the maximum size")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return activeSimulationState{}, readErr
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var state activeSimulationState
	if decodeErr := decoder.Decode(&state); decodeErr != nil {
		return activeSimulationState{}, fmt.Errorf("decode recovery state: %w", decodeErr)
	}
	if trailingErr := decoder.Decode(new(struct{})); !errors.Is(trailingErr, io.EOF) {
		return activeSimulationState{}, errors.New("recovery state contains trailing data")
	}
	if state.SchemaVersion != activeSimulationSchemaVersion {
		return activeSimulationState{}, fmt.Errorf("unsupported recovery schema version %d", state.SchemaVersion)
	}
	if state.Request.Interface == "" || state.Request.ConfigPath == "" {
		return activeSimulationState{}, errors.New("recovery state is missing interface or configuration path")
	}
	if state.Request.ConfigData != "" || state.Request.TemplateName != "" {
		return activeSimulationState{}, errors.New("recovery state must reference a persisted configuration path")
	}
	return state, nil
}
