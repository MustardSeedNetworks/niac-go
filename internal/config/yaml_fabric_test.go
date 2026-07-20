package config

import "testing"

func TestLoadYAMLBytesRoutedAuthoring(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`
networks:
  - name: lab-access
    subnet: 10.10.200.0/24
    virtual_vlan: 200
attachments:
  - name: tester
    connect: lab-access
devices:
  - name: LAB-EDGE-R1
    type: router
    mac: 02:00:00:00:00:01
    interfaces:
      - name: outside
        network: lab-access
        address: 10.10.200.1/24
    routes:
      - destination: 10.20.0.0/16
        via: outside
`))
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v", err)
	}
	if len(cfg.Networks) != 1 || cfg.Networks[0].Name != "lab-access" {
		t.Fatalf("networks = %#v", cfg.Networks)
	}
	if len(cfg.Attachments) != 1 || cfg.Attachments[0].Network != "lab-access" {
		t.Fatalf("attachments = %#v", cfg.Attachments)
	}
	if got := cfg.Devices[0].Interfaces[0].Address; got != "10.10.200.1/24" {
		t.Fatalf("interface address = %q", got)
	}
	if got := cfg.Devices[0].Routes[0].Destination; got != "10.20.0.0/16" {
		t.Fatalf("route destination = %q", got)
	}
}
