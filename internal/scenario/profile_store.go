package scenario

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

var (
	// ErrInvalidProfile means a scenario profile failed field validation
	// (see profile_validate.go) before it could be stored.
	ErrInvalidProfile = errors.New("invalid scenario profile")
	// ErrProfileExists means a profile with the same identity is already stored.
	ErrProfileExists = errors.New("scenario profile already exists")
)

const (
	profileDirMode         = 0o700
	profileFileMode        = 0o600
	profileTempRandomBytes = 16
	customSource           = "captured"
)

// CustomProfiles reads reviewed walk-backed profiles from the library's
// private profile directory. Missing storage is an empty catalog.
func CustomProfiles(libraryRoot string) ([]DeviceProfile, error) {
	dir := filepath.Join(libraryRoot, "profiles")
	dirInfo, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []DeviceProfile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect scenario profile directory: %w", err)
	}
	if dirInfo.Mode()&os.ModeSymlink != 0 || !dirInfo.IsDir() {
		return nil, errors.New("scenario profile directory must be a real directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read scenario profiles: %w", err)
	}

	profiles := make([]DeviceProfile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return nil, fmt.Errorf("inspect scenario profile %s: %w", entry.Name(), statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("scenario profile %s must be a regular file", entry.Name())
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read scenario profile %s: %w", entry.Name(), readErr)
		}
		var profile DeviceProfile
		if decodeErr := json.Unmarshal(content, &profile); decodeErr != nil {
			return nil, fmt.Errorf("decode scenario profile %s: %w", entry.Name(), decodeErr)
		}
		if validationErr := ValidateCustomProfile(profile); validationErr != nil {
			return nil, fmt.Errorf("decode scenario profile %s: %w", entry.Name(), validationErr)
		}
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Role < profiles[j].Role })
	return profiles, nil
}

// SaveCustomProfile atomically creates a reviewed walk-backed profile. Roles
// are immutable identifiers and cannot replace built-in or existing profiles.
func SaveCustomProfile(libraryRoot string, profile DeviceProfile) error {
	profile.Source = customSource
	if err := ValidateCustomProfile(profile); err != nil {
		return err
	}
	for _, builtin := range Profiles() {
		if builtin.Role == profile.Role {
			return fmt.Errorf("%w: role %q is built in", ErrProfileExists, profile.Role)
		}
	}

	profiles, err := openProfileRoot(libraryRoot)
	if err != nil {
		return err
	}
	defer func() { _ = profiles.Close() }()

	destination := profile.Role + ".json"
	if _, err = profiles.Lstat(destination); err == nil {
		return fmt.Errorf("%w: role %q", ErrProfileExists, profile.Role)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect scenario profile: %w", err)
	}

	content, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scenario profile: %w", err)
	}
	content = append(content, '\n')
	random := make([]byte, profileTempRandomBytes)
	if _, err = rand.Read(random); err != nil {
		return fmt.Errorf("name scenario profile temporary file: %w", err)
	}
	temporaryName := ".profile-" + hex.EncodeToString(random) + ".tmp"
	temporary, err := profiles.OpenFile(
		temporaryName,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		profileFileMode,
	)
	if err != nil {
		return fmt.Errorf("create scenario profile temporary file: %w", err)
	}
	defer func() { _ = profiles.Remove(temporaryName) }()
	_, err = temporary.Write(content)
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write scenario profile: %w", err)
	}
	if err = profiles.Link(temporaryName, destination); errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: role %q", ErrProfileExists, profile.Role)
	} else if err != nil {
		return fmt.Errorf("commit scenario profile: %w", err)
	}
	return nil
}

func openProfileRoot(libraryRoot string) (*os.Root, error) {
	if err := os.MkdirAll(libraryRoot, profileDirMode); err != nil {
		return nil, fmt.Errorf("create scenario library directory: %w", err)
	}
	library, err := os.OpenRoot(libraryRoot)
	if err != nil {
		return nil, fmt.Errorf("open scenario library directory: %w", err)
	}
	defer func() { _ = library.Close() }()
	if err = library.MkdirAll("profiles", profileDirMode); err != nil {
		return nil, fmt.Errorf("create scenario profile directory: %w", err)
	}
	info, err := library.Lstat("profiles")
	if err != nil {
		return nil, fmt.Errorf("inspect scenario profile directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("scenario profile directory must be a real directory")
	}
	profiles, err := library.OpenRoot("profiles")
	if err != nil {
		return nil, fmt.Errorf("open scenario profile directory: %w", err)
	}
	return profiles, nil
}
