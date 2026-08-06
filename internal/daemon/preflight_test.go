package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestPreflightSimulationDoesNotPersistInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
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

func TestPreflightSimulationRejectsUnapprovedFlatTrunkVLAN(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	d, err := NewDaemon(Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := d.PreflightSimulation(trunkSessionRequest("warehouse", 201))
	if err != nil {
		t.Fatal(err)
	}
	if report.Safe || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Code != fabric.CodeAttachmentPolicyDenied {
		t.Fatalf("report = %#v", report)
	}
}

func TestPreflightSimulationValidatesHostInterface(t *testing.T) {
	t.Run("nonexistent", func(t *testing.T) {
		t.Setenv(e2eDryRunEnv, "")
		d := routedPolicyDaemon()
		req := routedRequest(2)
		req.Interface = "definitely-missing-niac-interface"

		report, err := d.PreflightSimulation(req)
		if err != nil {
			t.Fatalf("PreflightSimulation() error = %v", err)
		}
		if report.Safe || len(report.Diagnostics) != 1 ||
			report.Diagnostics[0].Code != fabric.CodeHostInterfaceUnavailable {
			t.Fatalf("report = %#v, want host-interface diagnostic", report)
		}
	})

	t.Run("available", func(t *testing.T) {
		t.Setenv(e2eDryRunEnv, "")
		interfaceName := availableInterface(t)
		d := routedPolicyDaemon()
		d.cfg.AttachmentPolicies[0].Interface = interfaceName
		req := routedRequest(2)
		req.Interface = interfaceName

		report, err := d.PreflightSimulation(req)
		if err != nil {
			t.Fatalf("PreflightSimulation() error = %v", err)
		}
		if !report.Safe {
			t.Fatalf("report = %#v, want safe preflight", report)
		}
	})

	t.Run("e2e dry run", func(t *testing.T) {
		t.Setenv(e2eDryRunEnv, "true")
		d := routedPolicyDaemon()
		req := routedRequest(2)
		req.Interface = "missing-e2e-interface"
		d.cfg.AttachmentPolicies[0].Interface = req.Interface

		report, err := d.PreflightSimulation(req)
		if err != nil {
			t.Fatalf("PreflightSimulation() error = %v", err)
		}
		if !report.Safe {
			t.Fatalf("report = %#v, want dry-run interface exemption", report)
		}
	})
}

func availableInterface(t *testing.T) string {
	t.Helper()
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error = %v", err)
	}
	for _, iface := range interfaces {
		if capture.InterfaceExists(iface.Name) {
			return iface.Name
		}
	}
	t.Fatal("no capture-capable host interfaces available")
	return ""
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
			d := &Daemon{
				cfg: Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{tt.policy}},
			}
			if got := d.bindingFromRequest(tt.request).PolicyApproved; got != tt.approved {
				t.Fatalf("PolicyApproved = %t, want %t", got, tt.approved)
			}
		})
	}
}

func TestPhysicalAttachmentPolicyApprovesAllowedTrunkVLAN(t *testing.T) {
	policy := fabric.PhysicalAttachmentPolicy{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201, 299},
	}
	tests := []struct {
		name    string
		binding fabric.Binding
		want    bool
	}{
		{
			name: "allowed VLAN",
			binding: fabric.Binding{
				Interface: "eth0", Mode: fabric.ModeTrunk, AccessVLAN: 201,
			},
			want: true,
		},
		{
			name: "unapproved VLAN",
			binding: fabric.Binding{
				Interface: "eth0", Mode: fabric.ModeTrunk, AccessVLAN: 202,
			},
		},
		{
			name: "wrong interface",
			binding: fabric.Binding{
				Interface: "eth1", Mode: fabric.ModeTrunk, AccessVLAN: 201,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policy.Approves(tt.binding); got != tt.want {
				t.Fatalf("Approves() = %t, want %t", got, tt.want)
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
		t.Fatalf(
			"loadAuthorizedSimulationConfig() error = %v, want ErrRoutedLabsLicenseRequired",
			err,
		)
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
		t.Fatalf(
			"loadAuthorizedSimulationConfig() error = %v, want ErrRoutedLabsLicenseRequired",
			err,
		)
	}
}

func TestLoadAuthorizedSimulationConfigRejectsMissingSSHPassword(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: `
devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    ips: [192.0.2.1]
    ssh:
      enabled: true
      username: admin
      password_env: NIAC_MISSING_SSH_PASSWORD
`}

	_, _, err := loadAuthorizedSimulationConfig(req, fullSimulationEntitlements())

	if !errors.Is(err, config.ErrSSHPasswordUnavailable) {
		t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want SSH password requirement", err)
	}
}

func TestLoadAuthorizedSimulationConfigEnforcesFreeDeviceCap(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: simulationConfigWithDevices(
		api.FreeTierDeviceCount + 1,
	)}

	_, _, err := loadAuthorizedSimulationConfig(req, api.SimulationEntitlements{})

	if !errors.Is(err, api.ErrUnlimitedDevicesLicenseRequired) {
		t.Fatalf(
			"loadAuthorizedSimulationConfig() error = %v, want ErrUnlimitedDevicesLicenseRequired",
			err,
		)
	}
}

func TestLoadAuthorizedSegmentedConfigEnforcesDeviceEntitlements(t *testing.T) {
	tests := []struct {
		name         string
		counts       []int
		entitlements api.SimulationEntitlements
		wantErr      error
	}{
		{name: "Free 10", counts: []int{4, 6}},
		{
			name: "Free 11", counts: []int{5, 6},
			wantErr: api.ErrUnlimitedDevicesLicenseRequired,
		},
		{
			name: "Pro 11", counts: []int{5, 6},
			entitlements: fullSimulationEntitlements(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := api.SimulationRequest{
				Interface: "eth0", ConfigData: segmentedSimulationConfig(test.counts...),
			}
			_, _, err := loadAuthorizedSimulationConfig(req, test.entitlements)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("loadAuthorizedSimulationConfig() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestLoadAuthorizedSimulationConfigEnforcesAbsoluteDeviceCap(t *testing.T) {
	req := api.SimulationRequest{Interface: "eth0", ConfigData: simulationConfigWithDevices(
		api.MaxDeviceCount + 1,
	)}

	_, _, err := loadAuthorizedSimulationConfig(req, fullSimulationEntitlements())

	if !errors.Is(err, api.ErrSimulationDeviceLimitExceeded) {
		t.Fatalf(
			"loadAuthorizedSimulationConfig() error = %v, want ErrSimulationDeviceLimitExceeded",
			err,
		)
	}
}

func simulationConfigWithDevices(count int) string {
	var content strings.Builder
	content.WriteString("devices:\n")
	for index := range count {
		_, _ = fmt.Fprintf(
			&content,
			"  - name: device-%d\n    type: host\n    mac: 02:00:00:%02x:%02x:%02x\n    ips: [\"198.18.%d.%d\"]\n",
			index,
			index>>16,
			index>>8&0xff,
			index&0xff,
			index/254,
			index%254+1,
		)
	}
	return content.String()
}

func segmentedSimulationConfig(counts ...int) string {
	var content strings.Builder
	content.WriteString("segments:\n")
	deviceIndex := 0
	for segmentIndex, count := range counts {
		_, _ = fmt.Fprintf(&content, "  - tag: %d\n    devices:\n", segmentIndex+1)
		for range count {
			_, _ = fmt.Fprintf(&content,
				"      - name: device-%d\n"+
					"        type: host\n"+
					"        mac: 02:00:00:%02x:%02x:%02x\n"+
					"        ips: [\"198.18.%d.%d\"]\n",
				deviceIndex, deviceIndex>>16, deviceIndex>>8&0xff, deviceIndex&0xff,
				deviceIndex/254, deviceIndex%254+1,
			)
			deviceIndex++
		}
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
	t.Setenv(e2eDryRunEnv, "true")
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
	t.Setenv(e2eDryRunEnv, "true")
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
