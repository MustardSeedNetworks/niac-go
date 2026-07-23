package protocols

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestInvokingUserConfigDirUsesSudoOperatorHome(t *testing.T) {
	invoking, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current() error = %v", err)
	}
	root := &user.User{Uid: "0", Username: "root", HomeDir: filepath.Join(t.TempDir(), "root")}
	currentConfigDir := filepath.Join(root.HomeDir, "Library", "Application Support")
	want := filepath.Join(invoking.HomeDir, "Library", "Application Support")
	if got := invokingUserConfigDir(root, invoking.Username, currentConfigDir); got != want {
		t.Fatalf("invokingUserConfigDir() = %q, want %q", got, want)
	}
}

func TestSSHHostSignerPersistsPerDevice(t *testing.T) {
	directory := t.TempDir()
	first, err := loadOrCreateSSHHostSignerIn(directory, "edge-1")
	if err != nil {
		t.Fatalf("first loadOrCreateSSHHostSignerIn() error = %v", err)
	}
	second, err := loadOrCreateSSHHostSignerIn(directory, "edge-1")
	if err != nil {
		t.Fatalf("second loadOrCreateSSHHostSignerIn() error = %v", err)
	}
	other, err := loadOrCreateSSHHostSignerIn(directory, "edge-2")
	if err != nil {
		t.Fatalf("other loadOrCreateSSHHostSignerIn() error = %v", err)
	}

	if !bytes.Equal(first.PublicKey().Marshal(), second.PublicKey().Marshal()) {
		t.Fatal("host key changed for the same device")
	}
	if bytes.Equal(first.PublicKey().Marshal(), other.PublicKey().Marshal()) {
		t.Fatal("different devices share a host key")
	}
}

func TestSSHHostSignerRejectsCorruptPersistedKey(t *testing.T) {
	directory := t.TempDir()
	digest := sha256.Sum256([]byte("edge-1"))
	path := filepath.Join(directory, fmt.Sprintf("%x.key", digest))
	if err := os.WriteFile(path, []byte("invalid"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := loadOrCreateSSHHostSignerIn(directory, "edge-1"); err == nil {
		t.Fatal("loadOrCreateSSHHostSignerIn() accepted a corrupt key")
	}
}

func TestSSHHostSignerConcurrentCreationUsesOneKey(t *testing.T) {
	directory := t.TempDir()
	const callers = 16
	keys := make(chan []byte, callers)
	errors := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Go(func() {
			signer, err := loadOrCreateSSHHostSignerIn(directory, "edge-1")
			if err != nil {
				errors <- err
				return
			}
			keys <- signer.PublicKey().Marshal()
		})
	}
	group.Wait()
	close(keys)
	close(errors)
	for err := range errors {
		t.Fatalf("loadOrCreateSSHHostSignerIn() error = %v", err)
	}
	var first []byte
	for key := range keys {
		if first == nil {
			first = key
			continue
		}
		if !bytes.Equal(first, key) {
			t.Fatal("concurrent callers received different host keys")
		}
	}
}

func useTemporarySSHHostKeys(t *testing.T, stack *Stack) {
	t.Helper()
	directory := t.TempDir()
	stack.tcpHandler.ssh.hostKey = func(deviceName string) (ssh.Signer, error) {
		return loadOrCreateSSHHostSignerIn(directory, deviceName)
	}
}
