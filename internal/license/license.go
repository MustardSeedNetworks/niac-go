// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"os"
	"os/user"
	"path/filepath"
	"slices"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/niac-go/internal/userdir"
)

// The manager, activation state and parsed-license types are the foundation
// core's, aliased so NIAC callers keep referring to them as license.Manager
// etc. Their Tier fields are plain ints on the wire; convert with license.Tier
// when a NIAC tier name is needed (see Tier.String).
type (
	// ActivationState is the persisted activation snapshot GetState returns.
	ActivationState = fnd.ActivationState
	// ActivationResult is returned by Activate/StartTrial/CheckIn.
	ActivationResult = fnd.ActivationResult
	// Info is a parsed, validated license token.
	Info = fnd.Info
	// DeviceFingerprint identifies the host a license is bound to.
	DeviceFingerprint = fnd.DeviceFingerprint
)

// Manager owns NIAC's activation state and derives feature grants from the
// current product policy, so installed licenses gain newly introduced features
// without reactivation.
type Manager struct {
	*fnd.Manager
}

// NewManager creates a license manager rooted at NIAC's default config
// directory, verifying tokens against the embedded production key.
func NewManager() (*Manager, error) {
	manager, err := fnd.NewManager(fnd.NewProductionVerifier(Policy()), Policy())
	if err != nil {
		return nil, err
	}
	return &Manager{Manager: manager}, nil
}

// NewRuntimeManager loads license state from the invoking operator when a
// simulation needs elevated packet-capture privileges.
func NewRuntimeManager() (*Manager, error) {
	manager, err := fnd.NewManagerWithDir(
		fnd.NewProductionVerifier(Policy()), Policy(), runtimeConfigDir(),
	)
	if err != nil {
		return nil, err
	}
	return &Manager{Manager: manager}, nil
}

func runtimeConfigDir() string {
	current, err := user.Current()
	if err == nil {
		return filepath.Join(ResolveConfigHome(current, os.Getenv("SUDO_USER")), ".config", configSubdir)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".config", configSubdir)
}

// ResolveConfigHome returns the operator home that owns license state.
func ResolveConfigHome(current *user.User, sudoUser string) string {
	return userdir.ResolveHome(current, sudoUser)
}

// NewManagerWithDir creates a license manager that persists state in configDir.
// Used by tests to isolate activation state in a temp directory.
func NewManagerWithDir(configDir string) (*Manager, error) {
	manager, err := fnd.NewManagerWithDir(fnd.NewProductionVerifier(Policy()), Policy(), configDir)
	if err != nil {
		return nil, err
	}
	return &Manager{Manager: manager}, nil
}

// GetState returns activation state with features re-derived from this build's
// policy rather than trusting the feature snapshot stored by an older build.
func (m *Manager) GetState() *ActivationState {
	if m == nil || m.Manager == nil {
		return nil
	}
	state := m.Manager.GetState()
	if state == nil {
		return nil
	}
	result := *state
	result.Features = nil
	if m.Manager.IsActivated() {
		features, _, ok := Policy().FeaturesForTier(state.Tier)
		if ok {
			result.Features = slices.Clone(features)
		}
	}
	return &result
}

// HasFeature checks the current tier against this build's policy catalog.
func (m *Manager) HasFeature(feature string) bool {
	if m == nil || m.Manager == nil || !m.Manager.IsActivated() {
		return false
	}
	state := m.GetState()
	return state != nil && slices.Contains(state.Features, feature)
}

// FormatKey returns a signed token trimmed for display.
func FormatKey(key string) string {
	return fnd.FormatKey(key)
}
