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
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	activeSimulationSchemaVersion = 2
	activeSimulationFileName      = "active-simulation.json"
	maxActiveSimulationStateSize  = 64 * 1024
	recoveryFileMode              = 0o600
	recoveryStateRecovered        = "recovered"
	recoveryStateFailed           = "failed"
)

type activeSimulationState struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Sessions      []activeSimulationEntry `json:"sessions"`
	SavedAt       time.Time               `json:"savedAt"`
}

type activeSimulationEntry struct {
	Request api.SimulationRequest `json:"request"`
}

// DefaultRecoveryPath returns the platform-aware daemon recovery record path.
func DefaultRecoveryPath() string {
	return filepath.Join(filepath.Dir(library.DefaultRoot()), "state", activeSimulationFileName)
}

func (d *Daemon) persistActiveSimulation(sessionID string, request api.SimulationRequest) error {
	if d.cfg.RecoveryPath == "" || d.recovering {
		return nil
	}
	requests, err := d.persistedSimulationRequests()
	if err != nil {
		return err
	}
	for id, simulation := range d.sessions.sessions {
		requests[id] = simulation.Request
	}
	request.SessionID = sessionID
	requests[sessionID] = request
	return d.persistSimulationRequests(requests)
}

func (d *Daemon) persistSessionsExcluding(sessionID string) error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	requests, err := d.persistedSimulationRequests()
	if err != nil {
		return err
	}
	for id, simulation := range d.sessions.sessions {
		requests[id] = simulation.Request
	}
	delete(requests, sessionID)
	if len(requests) == 0 {
		return d.clearActiveSimulation()
	}
	return d.persistSimulationRequests(requests)
}

func (d *Daemon) persistedSimulationRequests() (map[string]api.SimulationRequest, error) {
	requests := make(map[string]api.SimulationRequest, d.sessions.len()+1)
	state, err := readRecoveryState(d.cfg.RecoveryPath)
	if errors.Is(err, fs.ErrNotExist) {
		return requests, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read recovery state: %w", err)
	}
	for _, session := range state.Sessions {
		requests[session.Request.SessionID] = session.Request
	}
	return requests, nil
}

func (d *Daemon) persistSimulationRequests(requests map[string]api.SimulationRequest) error {
	ids := make([]string, 0, len(requests))
	for id := range requests {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	state := activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		SavedAt:       time.Now().UTC(),
	}
	for _, id := range ids {
		state.Sessions = append(state.Sessions, activeSimulationEntry{Request: requests[id]})
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

func (d *Daemon) recoverActiveSimulation() {
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
	d.mu.Lock()
	d.recovering = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.recovering = false
		d.mu.Unlock()
	}()
	var failures []error
	for _, session := range state.Sessions {
		if startErr := d.StartSimulation(session.Request); startErr != nil {
			failures = append(
				failures,
				fmt.Errorf("session %q: %w", session.Request.SessionID, startErr),
			)
		}
	}
	if len(failures) != 0 {
		d.setRecoveryFailure(attemptedAt, errors.Join(failures...))
		return
	}
	d.mu.Lock()
	d.recovery = &api.SimulationRecovery{
		State:       recoveryStateRecovered,
		Message:     fmt.Sprintf("%d active simulation sessions restored.", len(state.Sessions)),
		AttemptedAt: attemptedAt,
	}
	d.mu.Unlock()
	logging.Successf("✓ %d active simulation sessions recovered", len(state.Sessions))
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
		return activeSimulationState{}, fmt.Errorf(
			"unsupported recovery schema version %d",
			state.SchemaVersion,
		)
	}
	if len(state.Sessions) == 0 {
		return activeSimulationState{}, errors.New("recovery state contains no sessions")
	}
	seen := make(map[string]struct{}, len(state.Sessions))
	for _, session := range state.Sessions {
		request := session.Request
		if request.SessionID == "" || request.Interface == "" || request.ConfigPath == "" {
			return activeSimulationState{}, errors.New(
				"recovery state is missing session, interface, or configuration path",
			)
		}
		if _, duplicate := seen[request.SessionID]; duplicate {
			return activeSimulationState{}, errors.New(
				"recovery state contains a duplicate session ID",
			)
		}
		seen[request.SessionID] = struct{}{}
		if request.ConfigData != "" || request.TemplateName != "" {
			return activeSimulationState{}, errors.New(
				"recovery state must reference persisted configuration paths",
			)
		}
	}
	return state, nil
}
