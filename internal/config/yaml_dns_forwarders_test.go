package config

import (
	"net"
	"strings"
	"testing"
)

func TestYAMLDNSForwardersRoundTrip(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`devices:
  - name: site-dns
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.10]
    dns:
      forwarders: [192.0.2.1, 1.1.1.1]
`))
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Devices[0].DNSConfig.Forwarders
	if len(got) != 2 || !got[0].Equal(net.ParseIP("192.0.2.1")) || !got[1].Equal(net.ParseIP("1.1.1.1")) {
		t.Fatalf("forwarders = %v, want [192.0.2.1 1.1.1.1] in order", got)
	}
	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "forwarders:") || !strings.Contains(string(data), "1.1.1.1") {
		t.Fatalf("forwarders lost on export:\n%s", data)
	}
}

func TestYAMLDNSForwarderMustBeAnAddress(t *testing.T) {
	_, err := LoadYAMLBytes([]byte(`devices:
  - name: site-dns
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.10]
    dns:
      forwarders: [dns.example]
`))
	if err == nil {
		t.Fatal("a forwarder that is not an IP address loaded")
	}
}

func TestValidateRejectsSelfForwardingDNSServer(t *testing.T) {
	address := net.ParseIP("192.0.2.10")
	cfg := &Config{Devices: []Device{{
		Name: "site-dns", Type: "server",
		MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		IPAddresses: []net.IP{address},
		DNSConfig:   &DNSConfig{Forwarders: []net.IP{address}},
	}}}
	result := NewValidator("test.yaml").Validate(cfg)
	if result.Valid || !strings.Contains(result.Format(), "forwarder is this server's own address") {
		t.Fatalf("self-forwarding server validated:\n%s", result.Format())
	}
}
