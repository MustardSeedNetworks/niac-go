package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const runtimeGenerationBytes = 16

func newRuntimeGeneration() (string, error) {
	var bytes [runtimeGenerationBytes]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate runtime identity: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func validRuntimeGeneration(generation string) bool {
	if len(generation) != hex.EncodedLen(runtimeGenerationBytes) {
		return false
	}
	for _, char := range generation {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

// The manifest is the commit point: a replacement's snapshot must exist first,
// without replacing the previous generation's recoverable state.
func (d *Daemon) commitSimulationGeneration(sim *Simulation, recovering bool) error {
	if recovering || d.cfg.RecoveryPath == "" {
		return nil
	}
	if err := d.writeRuntimeState(sim.SessionID, sim.runtimeGeneration, sim.stack); err != nil {
		return err
	}
	return d.persistActiveSimulation(sim)
}

func (d *Daemon) discardPendingGeneration(sim *Simulation, inline bool) {
	d.clearRuntimeState(sim.SessionID, sim.runtimeGeneration)
	if inline {
		if err := os.Remove(sim.ConfigPath); err != nil {
			logging.Warningf("Could not remove uncommitted inline configuration: %v", err)
		}
	}
}
