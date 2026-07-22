package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestPreflightSimulationDoesNotPersistInlineConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", dir)
	d := routedPolicyDaemon()
	req := routedRequest(2)

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	if !report.Safe {
		t.Fatalf("safe = false, diagnostics = %#v", report.Diagnostics)
	}
	if _, err = os.Stat(filepath.Join(dir, inlineConfigName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inline config stat error = %v, want not exist", err)
	}
}

func TestPhysicalAttachmentPolicyRequiresExactMatch(t *testing.T) {
	tests := []struct {
		name     string
		policy   fabric.PhysicalAttachmentPolicy
		request  api.SimulationRequest
		approved bool
	}{
		{
			name: "exact access policy", policy: fabric.PhysicalAttachmentPolicy{
				Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
			},
			request: api.SimulationRequest{
				Interface: "eth0", AttachmentMode: fabric.ModeAccess, AccessVLAN: 200,
			},
			approved: true,
		},
		{
			name: "exact direct policy", policy: fabric.PhysicalAttachmentPolicy{
				Interface: "eth1", Mode: fabric.ModeDirect,
			},
			request:  api.SimulationRequest{Interface: "eth1", AttachmentMode: fabric.ModeDirect},
			approved: true,
		},
		{
			name: "wrong interface", policy: fabric.PhysicalAttachmentPolicy{
				Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
			},
			request: api.SimulationRequest{
				Interface: "eth1", AttachmentMode: fabric.ModeAccess, AccessVLAN: 200,
			},
		},
		{
			name: "wrong mode", policy: fabric.PhysicalAttachmentPolicy{
				Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
			},
			request: api.SimulationRequest{Interface: "eth0", AttachmentMode: fabric.ModeDirect},
		},
		{
			name: "wrong VLAN", policy: fabric.PhysicalAttachmentPolicy{
				Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
			},
			request: api.SimulationRequest{
				Interface: "eth0", AttachmentMode: fabric.ModeAccess, AccessVLAN: 201,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Daemon{cfg: Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{tt.policy}}}
			if got := d.bindingFromRequest(tt.request).PolicyApproved; got != tt.approved {
				t.Fatalf("PolicyApproved = %t, want %t", got, tt.approved)
			}
		})
	}
}

func TestStartSimulationRecompilesUnsafeRoutedRequest(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := routedPolicyDaemon()
	req := routedRequest(0)

	err := d.StartSimulation(req, fullSimulationEntitlements())

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
}

func TestStartSimulationRejectsRoutedConfigInsideLockedTransaction(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := routedPolicyDaemon()

	err := d.StartSimulation(routedRequest(2), api.SimulationEntitlements{})

	if !errors.Is(err, api.ErrRoutedLabsLicenseRequired) {
		t.Fatalf("StartSimulation() error = %v, want ErrRoutedLabsLicenseRequired", err)
	}
	if d.simulation != nil {
		t.Fatal("unlicensed routed configuration changed simulation state")
	}
}

func TestStartSimulationRejectsUnlicensedSSHManagement(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: `
devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    ips:
      - 192.0.2.1
    ssh:
      enabled: true
      username: admin
      password_env: NIAC_TEST_SSH_PASSWORD
`}

	_, _, err := loadAuthorizedSimulationConfig(req, api.SimulationEntitlements{})

	if !errors.Is(err, api.ErrRoutedLabsLicenseRequired) {
		t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want ErrRoutedLabsLicenseRequired", err)
	}
}

func TestStartSimulationRejectsUnlicensedSSHManagementInSegment(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: `
segments:
  - tag: 10
    devices:
      - name: edge
        type: router
        mac: 02:00:00:00:00:01
        ips:
          - 192.0.2.1
        ssh:
          enabled: true
          username: admin
          password_env: NIAC_TEST_SSH_PASSWORD
`}

	_, _, err := loadAuthorizedSimulationConfig(req, api.SimulationEntitlements{})

	if !errors.Is(err, api.ErrRoutedLabsLicenseRequired) {
		t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want ErrRoutedLabsLicenseRequired", err)
	}
}

func TestLoadAuthorizedSimulationConfigEnforcesFreeDeviceCap(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: simulationConfigWithDevices(
		api.FreeTierDeviceCount + 1,
	)}

	_, _, err := loadAuthorizedSimulationConfig(req, api.SimulationEntitlements{})

	if !errors.Is(err, api.ErrUnlimitedDevicesLicenseRequired) {
		t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want ErrUnlimitedDevicesLicenseRequired", err)
	}
}

func TestLoadAuthorizedSimulationConfigEnforcesAbsoluteDeviceCap(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: simulationConfigWithDevices(
		api.MaxDeviceCount + 1,
	)}

	_, _, err := loadAuthorizedSimulationConfig(req, fullSimulationEntitlements())

	if !errors.Is(err, api.ErrSimulationDeviceLimitExceeded) {
		t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want ErrSimulationDeviceLimitExceeded", err)
	}
}

func simulationConfigWithDevices(count int) string {
	var content strings.Builder
	content.WriteString("devices:\n")
	for index := range count {
		_, _ = fmt.Fprintf(&content,
			"  - name: device-%d\n    type: host\n    mac: 02:00:00:%02x:%02x:%02x\n    ips: [\"198.18.%d.%d\"]\n",
			index, index>>16, index>>8&0xff, index&0xff, index/254, index%254+1,
		)
	}
	return content.String()
}

func TestStartSimulationRejectsAttachmentWithoutRoutedNetworks(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := routedPolicyDaemon()
	req := api.SimulationRequest{
		Interface: "eth0", Attachment: "tester", AttachmentMode: fabric.ModeAccess,
		AccessVLAN: 200, ConfigData: `
attachments:
  - name: tester
    connect: missing-network
devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    ips:
      - 192.0.2.1
`,
	}

	err := d.StartSimulation(req, fullSimulationEntitlements())

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
}

func TestUnsafeReplacementLeavesRunningSimulationIntact(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	running := &Simulation{Interface: "eth0"}
	d := routedPolicyDaemon()
	d.simulation = running

	err := d.StartSimulation(routedRequest(0), fullSimulationEntitlements())

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
	if d.simulation != running {
		t.Fatal("unsafe replacement stopped the running simulation")
	}
}

func TestUnsafeReplacementDoesNotPersistRejectedInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	dir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", dir)
	path := filepath.Join(dir, inlineConfigName)
	const accepted = "accepted configuration\n"
	if err := os.WriteFile(path, []byte(accepted), 0o600); err != nil {
		t.Fatalf("seed running config: %v", err)
	}
	d := routedPolicyDaemon()
	d.simulation = &Simulation{Interface: "eth0"}

	err := d.StartSimulation(routedRequest(0), fullSimulationEntitlements())

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read running config: %v", readErr)
	}
	if string(got) != accepted {
		t.Fatalf("running config = %q, want %q", got, accepted)
	}
}

func TestPreflightSimulationPreservesFlatScenarioWorkflow(t *testing.T) {
	d := &Daemon{}
	req := api.SimulationRequest{
		Interface: "eth0", Attachment: "tester", AttachmentMode: fabric.ModeAccess,
		AccessVLAN: 200, ConfigData: `
devices:
  - name: legacy-router
    type: router
    mac: 02:00:00:00:00:01
    ips:
      - 10.10.200.1
`,
	}

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	if !report.Safe {
		t.Fatalf("safe = false, diagnostics = %#v", report.Diagnostics)
	}
}

func TestPreflightSimulationRejectsDuplicateSegmentIdentity(t *testing.T) {
	d := &Daemon{}
	req := api.SimulationRequest{Interface: "eth0", ConfigData: `
segments:
  - tag: "10"
    devices:
      - name: duplicate
        type: switch
        mac: 02:00:00:00:00:01
        ips:
          - 192.0.2.10
  - tag: "20"
    devices:
      - name: duplicate
        type: switch
        mac: 02:00:00:00:00:01
        ips:
          - 192.0.2.10
`}

	_, err := d.PreflightSimulation(req)

	if !errors.Is(err, ErrInvalidSimulationConfig) {
		t.Fatalf("PreflightSimulation() error = %v, want ErrInvalidSimulationConfig", err)
	}
}

func routedRequest(accessVLAN uint16) api.SimulationRequest {
	return api.SimulationRequest{
		Interface:      "eth0",
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData: `
networks:
  - name: lab-access
    subnet: 10.10.200.0/24
attachments:
  - name: tester
    connect: lab-access
devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    interfaces:
      - name: outside
        network: lab-access
        address: 10.10.200.1/24
`,
	}
}

func routedPolicyDaemon() *Daemon {
	return &Daemon{cfg: Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 2,
	}}}}
}
