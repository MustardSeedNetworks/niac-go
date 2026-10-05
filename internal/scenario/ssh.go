package scenario

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// ciscoSSHVersion is the identification string the Cisco IOS XE SSH server
// sends on every release the packs model: a scanner reports it as "Cisco SSH
// 1.25 (protocol 2.0)".
const ciscoSSHVersion = "SSH-2.0-Cisco-1.25"

// packSSH serves SSH on the IOS XE routers, switches and wireless controllers,
// the boxes an operator manages over SSH. It authors no account: the device
// answers key exchange with its banner and host key and refuses every login,
// so a pack needs no daemon secret and keeps the no-default-credentials rule.
// An operator who wants the CLI adds username and password_env.
//
// NX-OS, PAN-OS and the access points stay unauthored until their banners are
// taken from real hardware, and endpoints are not managed this way.
func packSSH(role, software string) (*converter.SSHConfig, *converter.OSFingerprintConfig) {
	if role == "ap" || !strings.HasPrefix(software, "IOS XE") {
		return nil, nil
	}
	return &converter.SSHConfig{Enabled: true}, &converter.OSFingerprintConfig{SSHBanner: ciscoSSHVersion}
}
