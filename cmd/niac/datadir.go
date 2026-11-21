package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// serviceDirPerm is the mode the service creates its directories with when
// install.ps1 has not already made them.
const serviceDirPerm = 0o750

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

// prepareServiceDirs makes sure the service's data and certificate directories
// exist and are plain directories, so a symlink or junction planted where one
// belongs stops the service instead of redirecting where it writes its key.
func prepareServiceDirs() error {
	for _, dir := range []string{windowsDataDir(), defaultCertDir()} {
		if err := ensureDataDir(dir); err != nil {
			return err
		}
		if err := verifyDataDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// ensureDataDir and verifyDataDir are the seam for foundation's pkg/securedir,
// which will create the directory with a fail-closed protected DACL and judge
// owner and DACL on an opened handle. Until it ships they keep the daemon's
// existing behaviour and refuse anything that is not a plain directory.
//
// TODO(#3092): adopt foundation securedir.Ensure and securedir.Verify here.
func ensureDataDir(path string) error {
	if err := os.MkdirAll(path, serviceDirPerm); err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	return nil
}

func verifyDataDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		return fmt.Errorf("%s is a link or reparse point, not a directory: %w", path, errUnsafeDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory: %w", path, errUnsafeDir)
	}
	return nil
}

var errUnsafeDir = errors.New("unsafe data directory")
