package converter

import (
	"errors"
	"strings"
	"testing"
)

func validDevice() Device {
	return Device{
		Name: "core-sw-01",
		Type: "switch",
		MAC:  "AA:BB:CC:DD:EE:FF",
		IPs:  []string{"10.0.0.1"},
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	cfg := &Config{Devices: []Device{validDevice()}}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfig_EmptyDevices_IsAllowed(t *testing.T) {
	// Backward compat: existing manual checks accept zero devices.
	cfg := &Config{}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("empty Devices should be allowed, got %v", err)
	}
}

func TestValidateConfig_MissingMAC_KeepsSentinel(t *testing.T) {
	// Legacy callers rely on errors.Is(err, ErrDeviceMissingMAC).
	cfg := &Config{Devices: []Device{{Name: "no-mac"}}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing MAC")
	}
	if !errors.Is(err, ErrDeviceMissingMAC) {
		t.Errorf("expected ErrDeviceMissingMAC, got %v", err)
	}
	if errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected sentinel path, not struct-validator path")
	}
}

func TestValidateConfig_InvalidMAC(t *testing.T) {
	d := validDevice()
	d.MAC = "not-a-mac" // unparseable
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid MAC")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
	}
	if !strings.Contains(err.Error(), "mac") {
		t.Errorf("error should mention mac: %v", err)
	}
}

func TestValidateConfig_VendorIdentity(t *testing.T) {
	device := validDevice()
	device.MAC = ""
	device.Vendor = "cisco"
	device.MACSuffix = 0x010203
	if err := ValidateConfig(&Config{Devices: []Device{device}}); err != nil {
		t.Fatalf("vendor identity rejected: %v", err)
	}
}

func TestValidateConfig_RejectsConflictingMACSources(t *testing.T) {
	device := validDevice()
	device.Vendor = "cisco"
	err := ValidateConfig(&Config{Devices: []Device{device}})
	if !errors.Is(err, ErrDeviceMACSourceConflict) {
		t.Fatalf("error = %v, want ErrDeviceMACSourceConflict", err)
	}
}

func TestValidateConfig_InvalidIP(t *testing.T) {
	d := validDevice()
	d.IPs = []string{"not.an.ip.address"}
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid IP in ips list")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
	}
	if !strings.Contains(err.Error(), "ip") {
		t.Errorf("error should mention ip: %v", err)
	}
}

func TestValidateConfig_OutOfRangeVLAN(t *testing.T) {
	d := validDevice()
	d.VLAN = 5000 // 802.1Q max is 4094
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for VLAN > 4094")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestValidateConfig_InvalidDeviceType(t *testing.T) {
	d := validDevice()
	d.Type = "toaster" // not in the allowed enum
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown device type")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
	}
	if !strings.Contains(err.Error(), "toaster") {
		t.Errorf("error should mention the offending value: %v", err)
	}
}

func TestValidateConfig_Layer3SwitchAccepted(t *testing.T) {
	d := validDevice()
	d.Type = "layer3-switch"
	if err := ValidateConfig(&Config{Devices: []Device{d}}); err != nil {
		t.Fatalf("expected layer3-switch to be valid, got %v", err)
	}
}

func TestValidateConfig_CapturedInterfaceValuesAccepted(t *testing.T) {
	device := validDevice()
	device.Interfaces = []Interface{{Name: "lo", Type: "l3ipvlan", MTU: 65536}}
	if err := ValidateConfig(&Config{Devices: []Device{device}}); err != nil {
		t.Fatalf("captured interface rejected: %v", err)
	}
}

func TestValidateConfig_MultipleIPsAccepted(t *testing.T) {
	d := validDevice()
	d.IPs = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	cfg := &Config{Devices: []Device{d}}
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("expected no error for multiple ips list, got %v", err)
	}
}

func TestValidateConfig_DNSRecord_InvalidIP(t *testing.T) {
	d := validDevice()
	d.DNS = &DNSServer{
		ForwardRecords: []DNSRecord{
			{Name: "host.example", IP: "999.999.999.999"},
		},
	}
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid DNS IP")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestValidateConfig_CapturePlayback_MissingFileName(t *testing.T) {
	// Sentinel path: empty CapturePlayback.FileName trips the manual check first.
	cfg := &Config{
		CapturePlaybacks: []CapturePlayback{{LoopTime: 100}},
		Devices:          []Device{validDevice()},
	}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing playback filename")
	}
	if !errors.Is(err, ErrCapturePlaybackMissingFile) {
		t.Errorf("expected ErrCapturePlaybackMissingFile, got %v", err)
	}
}

func TestValidateConfig_IPv6Prefixes(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		address  string
		wantPath string
	}{
		{name: "dual-stack", subnet: "2001:db8:10::/64", address: "2001:db8:10::5/64"},
		{name: "IPv4 subnet_v6", subnet: "10.10.0.0/24", wantPath: "networks[0].subnet_v6"},
		{
			name:     "IPv4 address_v6",
			subnet:   "2001:db8:10::/64",
			address:  "10.10.0.5/24",
			wantPath: "interfaces[0].address_v6",
		},
		{
			name:     "bare address_v6",
			subnet:   "2001:db8:10::/64",
			address:  "2001:db8:10::5",
			wantPath: "interfaces[0].address_v6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDevice()
			d.Interfaces = []Interface{{
				Name: "eth0", Network: "lab", Address: "10.10.0.5/24", AddressV6: tt.address,
			}}
			cfg := &Config{
				Networks: []Network{{Name: "lab", Subnet: "10.10.0.0/24", SubnetV6: tt.subnet}},
				Devices:  []Device{d},
			}
			err := ValidateConfig(cfg)
			if tt.wantPath == "" {
				if err != nil {
					t.Fatalf("ValidateConfig() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantPath) {
				t.Fatalf("ValidateConfig() = %v, want an error naming %s", err, tt.wantPath)
			}
		})
	}
}

func TestValidateConfig_SSHCredentialsArePaired(t *testing.T) {
	tests := []struct {
		name    string
		ssh     *SSHConfig
		wantErr bool
	}{
		{name: "no account", ssh: &SSHConfig{Enabled: true}},
		{name: "account", ssh: &SSHConfig{Enabled: true, Username: "admin", PasswordEnv: "NIAC_SSH_PASSWORD"}},
		{name: "username only", ssh: &SSHConfig{Enabled: true, Username: "admin"}, wantErr: true},
		{name: "password only", ssh: &SSHConfig{Enabled: true, PasswordEnv: "NIAC_SSH_PASSWORD"}, wantErr: true},
		{name: "blank username", ssh: &SSHConfig{
			Enabled: true, Username: " ", PasswordEnv: "NIAC_SSH_PASSWORD",
		}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := validDevice()
			device.SSH = test.ssh
			err := ValidateConfig(&Config{Devices: []Device{device}})
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateConfig() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
