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
	activeSimulationSchemaVersion = 3
	// activeSimulationSchemaV2 is the last schema that predates the
	// per-session runtime generation. It migrates forward rather than
	// stranding the sessions it carries.
	activeSimulationSchemaV2     = 2
	activeSimulationFileName     = "active-simulation.json"
	maxActiveSimulationStateSize = 64 * 1024
	recoveryFileMode             = 0o600
	recoveryStateRecovered       = "recovered"
	recoveryStateFailed          = "failed"
)

type activeSimulationState struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Sessions      []activeSimulationEntry `json:"sessions"`
	SavedAt       time.Time               `json:"savedAt"`
}

type activeSimulationEntry struct {
	Request    api.SimulationRequest `json:"request"`
	Generation string                `json:"generation"`
}

// DefaultRecoveryPath returns the platform-aware daemon recovery record path.
func DefaultRecoveryPath() string {
	return filepath.Join(filepath.Dir(library.DefaultRoot()), "state", activeSimulationFileName)
}

func (d *Daemon) persistActiveSimulation(sim *Simulation) error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	requests, err := d.persistedSimulationRequests()
	if err != nil {
		return err
	}
	for id, simulation := range d.sessions.sessions {
		requests[id] = activeSimulationEntry{Request: simulation.Request, Generation: simulation.runtimeGeneration}
	}
	requests[sim.SessionID] = activeSimulationEntry{Request: sim.Request, Generation: sim.runtimeGeneration}
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
		requests[id] = activeSimulationEntry{Request: simulation.Request, Generation: simulation.runtimeGeneration}
	}
	delete(requests, sessionID)
	if len(requests) == 0 {
		return d.clearActiveSimulation()
	}
	return d.persistSimulationRequests(requests)
}

func (d *Daemon) persistedSimulationRequests() (map[string]activeSimulationEntry, error) {
	requests := make(map[string]activeSimulationEntry, d.sessions.len()+1)
	state, err := readRecoveryState(d.cfg.RecoveryPath)
	if errors.Is(err, fs.ErrNotExist) {
		return requests, nil
	}
	if errors.Is(err, errUnusableRecoveryState) {
		// Startup sets this aside, so reaching here means something wrote an
		// unusable file afterwards. A start must still succeed: the whole
		// defect in #2092 was a stale file turning every start into a 500.
		logging.Warningf("ignoring unusable recovery state %s: %v", d.cfg.RecoveryPath, err)

		return requests, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read recovery state: %w", err)
	}
	for _, session := range state.Sessions {
		requests[session.Request.SessionID] = session
	}
	return requests, nil
}

func (d *Daemon) persistSimulationRequests(requests map[string]activeSimulationEntry) error {
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
		state.Sessions = append(state.Sessions, requests[id])
	}
	data, marshalErr := json.MarshalIndent(state, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("encode recovery state: %w", marshalErr)
	}
	data = append(data, '\n')
	return writeRecoveryState(d.cfg.RecoveryPath, data)
}

func writeRecoveryState(path string, data []byte) error {
	return writeStateFile(filepath.Dir(path), filepath.Base(path), data)
}

func (d *Daemon) clearActiveSimulation() error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	removeErr := removeStateFile(filepath.Dir(d.cfg.RecoveryPath), filepath.Base(d.cfg.RecoveryPath))
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
		if errors.Is(readErr, errUnusableRecoveryState) {
			d.quarantineUnusableState(attemptedAt, readErr)

			return
		}
		d.setRecoveryFailure(attemptedAt, readErr)

		return
	}
	var failures []error
	for _, session := range state.Sessions {
		if startErr := d.startGeneration(session.Request, session.Generation, true); startErr != nil {
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

// quarantineUnusableState sets stale state aside and reports where it went.
// The previous wording asked the operator to remove the file themselves, which
// left the daemon unable to start anything until they did (#2092).
func (d *Daemon) quarantineUnusableState(attemptedAt time.Time, recoveryErr error) {
	aside, renameErr := quarantineRecoveryState(d.cfg.RecoveryPath)
	if renameErr != nil {
		d.setRecoveryFailure(attemptedAt, fmt.Errorf("%w; and it could not be set aside: %w", recoveryErr, renameErr))

		return
	}
	message := fmt.Sprintf(
		"Previous simulation state could not be read (%v) and was set aside as %s. Simulations can be started normally.",
		recoveryErr,
		aside,
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

// errUnusableRecoveryState marks state this build can neither read nor migrate
// forward. Such a file is stale rather than authoritative: the daemon sets it
// aside and carries on, because leaving it in place makes every later start
// fail while the install otherwise looks healthy (#2092).
var errUnusableRecoveryState = errors.New("recovery state is unusable")

func unusableRecoveryState(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errUnusableRecoveryState, fmt.Sprintf(format, args...))
}

// migrateRecoveryState brings state written by an older build up to the current
// schema. Every step so far is additive, so a field the older schema lacked is
// derived here rather than costing the operator their running sessions.
func migrateRecoveryState(state *activeSimulationState) error {
	for state.SchemaVersion < activeSimulationSchemaVersion {
		switch state.SchemaVersion {
		case activeSimulationSchemaV2:
			// Schema 3 gave every session a runtime generation so committed
			// state can be matched to the run that wrote it. Sessions saved
			// before that have none and mint one on the way forward.
			for i := range state.Sessions {
				generation, genErr := newRuntimeGeneration()
				if genErr != nil {
					return genErr
				}
				state.Sessions[i].Generation = generation
			}
		default:
			return unusableRecoveryState(
				"no migration from schema version %d", state.SchemaVersion,
			)
		}
		state.SchemaVersion++
	}
	if state.SchemaVersion != activeSimulationSchemaVersion {
		return unusableRecoveryState(
			"written by a newer build (schema version %d)", state.SchemaVersion,
		)
	}

	return nil
}

// quarantineRecoveryState renames unusable state aside, keeping the bytes for
// diagnosis while leaving nothing behind that a later start would read.
func quarantineRecoveryState(path string) (string, error) {
	aside := fmt.Sprintf("%s.unusable-%s", path, time.Now().UTC().Format("20060102T150405Z"))
	if err := os.Rename(path, aside); err != nil {
		return "", err
	}

	return aside, nil
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
		return activeSimulationState{}, unusableRecoveryState("decode: %v", decodeErr)
	}
	if trailingErr := decoder.Decode(new(struct{})); !errors.Is(trailingErr, io.EOF) {
		return activeSimulationState{}, unusableRecoveryState("contains trailing data")
	}
	if migrateErr := migrateRecoveryState(&state); migrateErr != nil {
		return activeSimulationState{}, migrateErr
	}
	if len(state.Sessions) == 0 {
		return activeSimulationState{}, errors.New("recovery state contains no sessions")
	}
	seen := make(map[string]struct{}, len(state.Sessions))
	for _, session := range state.Sessions {
		request := session.Request
		if !validRuntimeGeneration(session.Generation) {
			return activeSimulationState{}, errors.New("recovery state contains an invalid generation")
		}
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
