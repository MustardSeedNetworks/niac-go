package config

import (
	"strings"
	"testing"
)

func TestYAMLDNSServicePresenceRoundTrip(t *testing.T) {
	for _, service := range []string{"", "    dns: {}\n"} {
		t.Run(strings.TrimSpace(service), func(t *testing.T) {
			cfg, err := LoadYAMLBytes(
				[]byte("devices:\n  - name: endpoint\n    mac: '02:00:00:00:00:01'\n    ips: [192.0.2.1]\n" + service),
			)
			if err != nil {
				t.Fatal(err)
			}
			wantDNS := service != ""
			if (cfg.Devices[0].DNSConfig != nil) != wantDNS {
				t.Fatalf("DNS configuration presence = %t, want %t", cfg.Devices[0].DNSConfig != nil, wantDNS)
			}
			data, err := MarshalConfigYAML(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "dns:") != wantDNS {
				t.Fatalf("DNS presence changed during export:\n%s", data)
			}
		})
	}
}
