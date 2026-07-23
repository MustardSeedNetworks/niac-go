package protocols

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

const sshHostKeyDirectory = "ssh-host-keys"

func loadOrCreateSSHHostSigner(deviceName string) (ssh.Signer, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config directory: %w", err)
	}
	current, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("resolve current user: %w", err)
	}
	configDir = invokingUserConfigDir(current, os.Getenv("SUDO_USER"), configDir)
	return loadOrCreateSSHHostSignerIn(filepath.Join(configDir, "niac", sshHostKeyDirectory), deviceName)
}

func invokingUserConfigDir(current *user.User, sudoUser, currentConfigDir string) string {
	home := license.ResolveConfigHome(current, sudoUser)
	if home == current.HomeDir || pathWithinHome(currentConfigDir, home) {
		return currentConfigDir
	}
	relative, err := filepath.Rel(current.HomeDir, currentConfigDir)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.Join(home, relative)
	}
	return filepath.Join(home, ".config")
}

func pathWithinHome(path, home string) bool {
	relative, err := filepath.Rel(home, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func loadOrCreateSSHHostSignerIn(directory, deviceName string) (ssh.Signer, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create SSH host key directory: %w", err)
	}
	digest := sha256.Sum256([]byte(deviceName))
	path := filepath.Join(directory, fmt.Sprintf("%x.key", digest))
	privateKey, err := readSSHHostPrivateKey(path)
	if err == nil {
		return ssh.NewSignerFromKey(privateKey)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	_, generated, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate SSH host key: %w", err)
	}
	if err = publishSSHHostPrivateKey(directory, path, generated.Seed()); errors.Is(err, os.ErrExist) {
		privateKey, err = readSSHHostPrivateKey(path)
		if err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(privateKey)
	}
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(generated)
}

func publishSSHHostPrivateKey(directory, path string, seed []byte) error {
	file, err := os.CreateTemp(directory, ".ssh-host-key-*")
	if err != nil {
		return fmt.Errorf("create temporary SSH host key: %w", err)
	}
	temporaryPath := file.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if _, err = file.Write(seed); err != nil {
		_ = file.Close()
		return fmt.Errorf("write SSH host key: %w", err)
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync SSH host key: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close SSH host key: %w", err)
	}
	if err = os.Link(temporaryPath, path); err != nil {
		return fmt.Errorf("publish SSH host key: %w", err)
	}
	return nil
}

func readSSHHostPrivateKey(path string) (ed25519.PrivateKey, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(encoded) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid SSH host key length in %q", path)
	}
	return ed25519.NewKeyFromSeed(encoded), nil
}
