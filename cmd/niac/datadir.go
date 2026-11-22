package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// windowsDataDir is where the Windows service keeps its certificates and the
// operator's attachment policy. install.ps1 creates it with a protected DACL
// that admits only SYSTEM and Administrators.
func windowsDataDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "NiAC")
}

// serviceDirs are the service's data and certificate directories, resolved
// once with every link in them followed by securedir.Canonical. securedir
// refuses a link anywhere in the paths it is given, so everything downstream
// uses these and not the raw environment values.
type serviceDirs struct {
	data string
	cert string
}

// resolveServiceDirs canonicalises the directories without touching them.
func resolveServiceDirs(opts securedir.Options) (serviceDirs, error) {
	data, err := securedir.Canonical(windowsDataDir(), opts)
	if err != nil {
		return serviceDirs{}, fmt.Errorf("resolve data directory: %w", err)
	}
	cert, err := securedir.Canonical(defaultCertDir(), opts)
	if err != nil {
		return serviceDirs{}, fmt.Errorf("resolve certificate directory: %w", err)
	}
	return serviceDirs{data: data, cert: cert}, nil
}

func (d serviceDirs) all() []string { return []string{d.data, d.cert} }

// ensure creates the directories as private, or takes over an existing one
// that securedir can show was never untrusted. It is for service install and
// start, which run as SYSTEM or an Administrator.
func (d serviceDirs) ensure(opts securedir.Options) error {
	for _, dir := range d.all() {
		if err := securedir.Ensure(dir, opts); err != nil {
			return fmt.Errorf("secure %s: %w", dir, err)
		}
	}
	return nil
}

// verify refuses to start on a directory that is no longer private: a swapped
// owner, a planted link, or anything under it an unprivileged user can write.
func (d serviceDirs) verify(opts securedir.Options) error {
	for _, dir := range d.all() {
		if err := securedir.Verify(dir, opts); err != nil {
			return fmt.Errorf("verify %s: %w", dir, err)
		}
	}
	return nil
}

// prepareServiceDirs is the service's start-up check: resolve, ensure, verify.
func prepareServiceDirs(opts securedir.Options) (serviceDirs, error) {
	dirs, err := resolveServiceDirs(opts)
	if err != nil {
		return serviceDirs{}, err
	}
	if err = dirs.ensure(opts); err != nil {
		return serviceDirs{}, err
	}
	if err = dirs.verify(opts); err != nil {
		return serviceDirs{}, err
	}
	return dirs, nil
}
