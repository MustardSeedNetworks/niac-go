package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const errWindowsSharingViolation = syscall.Errno(32)

func makeTestDirectoryLink(target, link string) error {
	err := os.Symlink(target, link)
	if !errors.Is(err, syscall.ERROR_PRIVILEGE_NOT_HELD) {
		return err
	}
	// Junction creation does not require symlink privilege. Paths remain data,
	// supplied only by the calling test's temporary-directory fixtures.
	command := exec.Command(
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		"New-Item -ItemType Junction -Path $env:NIAC_TEST_LINK -Target $env:NIAC_TEST_TARGET -ErrorAction Stop | Out-Null",
	)
	command.Env = append(os.Environ(), "NIAC_TEST_LINK="+link, "NIAC_TEST_TARGET="+target)
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		return fmt.Errorf("create test directory junction: %w: %s", commandErr, output)
	}
	return nil
}

func moveOpenedTestDirectory(source, destination string) (bool, error) {
	err := os.Rename(source, destination)
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) || errors.Is(err, errWindowsSharingViolation) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return false, errors.New("Windows renamed a directory held by OpenRoot without delete sharing")
}
