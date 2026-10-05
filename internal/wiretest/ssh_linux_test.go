//go:build linux && integration

package wiretest_test

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshPackDevice is an IOS XE access switch on the hospital pack's first site.
const sshPackDevice = "MED-ACC-SW01"

// D-NIAC-43: a pack serves SSH on its IOS XE gear with no daemon secret set.
// The stock OpenSSH client must reach key exchange and see the Cisco banner,
// and a login must be refused, because the pack authors no account.
func TestPackSSHRefusesLoginWithoutASecret(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	var host string
	for index := range authored.Devices {
		if device := &authored.Devices[index]; device.Name == sshPackDevice && len(device.IPAddresses) > 0 {
			host = device.IPAddresses[0].String()
		}
	}
	if host == "" {
		t.Fatalf("the hospital pack has no addressed %s", sshPackDevice)
	}
	target := net.JoinHostPort(host, "22")

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	// BatchMode: no prompt, so the client offers only what it has and stops.
	command := exec.CommandContext(ctx, "ssh", "-v",
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null", "-o", "ConnectTimeout=20",
		"admin@"+host, "exit")
	output, err := command.CombinedOutput()
	transcript := string(output)
	t.Logf("ssh -v %s:\n%s", target, transcript)
	if err == nil {
		t.Fatal("ssh logged in to a pack device that authors no account")
	}
	for _, want := range []string{
		"remote software version Cisco-1.25",
		"SSH2_MSG_KEX_ECDH_REPLY received",
		"Server host key:",
		"Permission denied",
	} {
		if !strings.Contains(transcript, want) {
			t.Errorf("ssh -v output lacks %q", want)
		}
	}

	// BatchMode offers no password, so try one too: no password opens the CLI.
	_, err = ssh.Dial("tcp", target, &ssh.ClientConfig{
		User: "admin", Auth: []ssh.AuthMethod{ssh.Password("admin")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 20 * time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("password login error = %v, want an authentication refusal", err)
	}
}
