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
		IP:   "10.0.0.1",
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

func TestValidateConfig_InvalidIP(t *testing.T) {
	d := validDevice()
	d.IP = "not.an.ip.address"
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid IP")
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

func TestValidateConfig_IPandIPsMutuallyExclusive(t *testing.T) {
	d := validDevice()
	d.IPs = []string{"10.0.0.2", "10.0.0.3"} // already has IP from validDevice()
	cfg := &Config{Devices: []Device{d}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error when both ip and ips are set")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected ErrConfigInvalid, got %v", err)
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
