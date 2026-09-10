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
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// A session's launch request says what to replay. This record says what the
// replay had become: the faults armed on it, the configuration edits made
// through the state API, saved checkpoints and the event history a consuming
// NMS has already seen. Without it a crash returns a faulted scenario healthy,
// with no clearing event, which is exactly the fault P2-1 exists to test.
//
// It is a separate file from active-simulation.json, one per generation: the
// recovery record is a 64 KiB bound on a handful of launch requests, while a
// 74-device pack's runtime state with event history is megabytes.
const (
	runtimeStateSchemaVersion = 2
	runtimeStateDirName       = "runtime"
	maxRuntimeStateSize       = 32 << 20
	// Crash recovery uses the last completed save. Scheduling and write
	// latency can extend the interval; orderly shutdown flushes final state.
	runtimeStateInterval = 2 * time.Second
)

type runtimeStateRecord struct {
	SchemaVersion int                          `json:"schemaVersion"`
	SessionID     string                       `json:"sessionId"`
	Generation    string                       `json:"generation"`
	SavedAt       time.Time                    `json:"savedAt"`
	Devices       map[string]devicestate.State `json:"devices"`
}

func (d *Daemon) runtimeStateDir() string {
	if d.cfg.RecoveryPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(d.cfg.RecoveryPath), runtimeStateDirName)
}

func runtimeStatePath(stateDir, sessionID, generation string) string {
	return filepath.Join(stateDir, runtimeStateDirName, sessionID+"."+generation+".json")
}

func (d *Daemon) runtimeStateFile(sessionID, generation string) string {
	directory := d.runtimeStateDir()
	// Session IDs reach here from the API, so the path is built only from one
	// the API's own grammar accepts -- lowercase, digits and hyphens, which
	// cannot name a parent directory.
	if directory == "" || !api.ValidSessionID(sessionID) || !validRuntimeGeneration(generation) {
		return ""
	}
	return filepath.Join(directory, sessionID+"."+generation+".json")
}

func (d *Daemon) writeRuntimeState(sessionID, generation string, stack *protocols.Stack) error {
	if d.cfg.RecoveryPath == "" {
		return nil
	}
	path := d.runtimeStateFile(sessionID, generation)
	if path == "" || stack == nil {
		return errors.New("runtime state requires a valid session, generation and stack")
	}
	record := runtimeStateRecord{
		SchemaVersion: runtimeStateSchemaVersion,
		SessionID:     sessionID,
		Generation:    generation,
		SavedAt:       time.Now().UTC(),
		Devices:       stack.ExportDeviceStates(),
	}
	data, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		return fmt.Errorf("encode runtime state: %w", marshalErr)
	}
	if len(data)+1 > maxRuntimeStateSize {
		return fmt.Errorf("runtime state exceeds %d bytes", maxRuntimeStateSize)
	}
	return writeStateFile(path, append(data, '\n'))
}

func (d *Daemon) loadRuntimeState(sessionID, generation string) (map[string]devicestate.State, error) {
	empty := map[string]devicestate.State{}
	path := d.runtimeStateFile(sessionID, generation)
	if path == "" {
		return empty, errors.New("runtime state requires a valid session and generation")
	}
	record, err := readRuntimeStateRecord(path)
	if err != nil {
		return empty, err
	}
	if record.SessionID != sessionID {
		return empty, errors.New("runtime state belongs to another session")
	}
	if record.Generation != generation {
		return empty, errors.New("runtime state belongs to another generation")
	}
	return record.Devices, nil
}

func (d *Daemon) clearRuntimeState(sessionID, generation string) {
	path := d.runtimeStateFile(sessionID, generation)
	if path == "" {
		return
	}
	if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		logging.Warningf("Could not remove runtime state for session %s: %v", sessionID, removeErr)
	}
}

func readRuntimeStateRecord(path string) (runtimeStateRecord, error) {
	info, statErr := os.Lstat(path)
	if statErr != nil {
		return runtimeStateRecord{}, statErr
	}
	if !info.Mode().IsRegular() {
		return runtimeStateRecord{}, errors.New("runtime state is not a regular file")
	}
	if info.Size() > maxRuntimeStateSize {
		return runtimeStateRecord{}, errors.New("runtime state exceeds the maximum size")
	}
	file, openErr := os.Open(path)
	if openErr != nil {
		return runtimeStateRecord{}, openErr
	}
	defer func() { _ = file.Close() }()
	data, readErr := io.ReadAll(io.LimitReader(file, maxRuntimeStateSize+1))
	if readErr != nil {
		return runtimeStateRecord{}, readErr
	}
	if len(data) > maxRuntimeStateSize {
		return runtimeStateRecord{}, errors.New("runtime state exceeds the maximum size")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record runtimeStateRecord
	if decodeErr := decoder.Decode(&record); decodeErr != nil {
		return runtimeStateRecord{}, fmt.Errorf("decode runtime state: %w", decodeErr)
	}
	if trailingErr := decoder.Decode(new(struct{})); !errors.Is(trailingErr, io.EOF) {
		return runtimeStateRecord{}, errors.New("runtime state contains trailing data")
	}
	if record.SchemaVersion != runtimeStateSchemaVersion {
		return runtimeStateRecord{}, fmt.Errorf(
			"unsupported runtime state schema version %d",
			record.SchemaVersion,
		)
	}
	return record, nil
}

// Only recovery restores runtime state. A fresh start uses the authored scenario.
func (d *Daemon) runtimeStateRestorer(sessionID, generation string) restoreRuntimeState {
	states, err := d.loadRuntimeState(sessionID, generation)
	if err != nil {
		// Fail closed, like the recovery record itself: a corrupt runtime
		// record is reported rather than silently replaced by a healthy
		// scenario that looks like a successful recovery.
		return func(*protocols.Stack) error {
			return fmt.Errorf("read runtime state: %w", err)
		}
	}
	return func(stack *protocols.Stack) error {
		return stack.RestoreDeviceStates(states)
	}
}
