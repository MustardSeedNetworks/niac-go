// SPDX-License-Identifier: BUSL-1.1

package attachpolicy

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// FileName is the policy file's name inside its directory.
const FileName = "attachment-policy.conf"

// maxFileBytes bounds what the daemon reads at startup. A real policy is a few
// lines; anything near this is not one.
const maxFileBytes = 64 << 10

const (
	filePerm = 0o644
	dirPerm  = 0o755
)

// DefaultPath is where the daemon looks for the policy file.
//
// The location is fixed per OS and deliberately ignores HOME, XDG_DATA_HOME
// and NIAC_LIBRARY_ROOT, which move the daemon's data directory: the file is
// the operator's, so the daemon and the CLI must find the same one however the
// service was launched, and a user-controlled variable must not be able to
// point the daemon at a file of its own choosing.
//
//   - macOS: /usr/local/niac, beside the binary and the launchd plist's working
//     directory, which is root-owned.
//   - Windows: %ProgramData%\NiAC, where the daemon keeps its certificates.
//   - elsewhere: /etc/niac, which the Linux package creates.
func DefaultPath() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "NiAC", FileName)
	case "darwin":
		return filepath.Join("/usr/local/niac", FileName)
	default:
		return filepath.Join("/etc/niac", FileName)
	}
}

// ErrUntrusted marks a file the daemon refuses because someone other than the
// operator could have written it.
var ErrUntrusted = errors.New("attachment policy file is not trusted")

// Load reads and parses the policy file at path. A file that does not exist
// holds no policies and is not an error.
func Load(path string, trust Trust) ([]Policy, error) {
	lines, err := readLines(path, trust)
	if err != nil {
		return nil, err
	}
	return parseLines(lines)
}

func untrusted(path string, reason error) error {
	return fmt.Errorf("%w: %s: %w", ErrUntrusted, path, reason)
}

// readLines returns the file's raw lines after the trust check, or nil for a
// file that does not exist.
func readLines(path string, trust Trust) ([]string, error) {
	file, err := openTrusted(path, trust)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if errors.Is(err, ErrUntrusted) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxFileBytes)
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), nil
}

// parseLines reads one policy per line; blank lines and lines starting with #
// are skipped. The result is checked for contradictions so a bad file is
// refused whole rather than half-applied.
func parseLines(lines []string) ([]Policy, error) {
	var policies []Policy
	for i, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		policy, err := Parse(entry)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		policies = append(policies, policy)
	}
	if err := Validate(policies); err != nil {
		return nil, err
	}
	return policies, nil
}

// Edit rewrites the policy file at path with the lines edit returns, keeping
// the operator's comments. The current file must pass the trust check (a
// missing one starts empty) and the result must parse and be consistent, so a
// bad edit never replaces a good file. The write is atomic and leaves the file
// readable by all and writable only by its owner.
func Edit(path string, trust Trust, edit func(lines []string) ([]string, error)) error {
	lines, err := readLines(path, trust)
	if err != nil {
		return err
	}
	current := trimTrailingBlank(lines)
	lines, err = edit(slices.Clone(current))
	if err != nil {
		return err
	}
	if slices.Equal(lines, current) && len(current) > 0 {
		return nil
	}
	if _, parseErr := parseLines(lines); parseErr != nil {
		return parseErr
	}
	return writeAtomic(path, []byte(strings.Join(lines, "\n")+"\n"))
}

func trimTrailingBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	name := filepath.Base(path)
	temp := "." + name + ".tmp"
	if removeErr := root.Remove(temp); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return fmt.Errorf("clear %s: %w", temp, removeErr)
	}
	file, err := createPrivateTemp(root, dir, temp)
	if err != nil {
		return fmt.Errorf("create %s: %w", temp, err)
	}
	if err = writeAndClose(file, data); err != nil {
		_ = root.Remove(temp)
		return fmt.Errorf("write %s: %w", temp, err)
	}
	if err = root.Rename(temp, name); err != nil {
		_ = root.Remove(temp)
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func writeAndClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
