package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestAddDaemonCommandFlags(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	info := versionInfo{version: "test", commit: "abc", date: "now"}
	addDaemonCommand(root, info)

	cmd := findSubcommand(root, "daemon")
	if cmd == nil {
		t.Fatal("Expected daemon command to be registered")
	}

	expectedFlags := []string{"attachment-policy", "listen", "token", "storage"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag on daemon command", flag)
		}
	}
	if flagType := cmd.Flags().Lookup("attachment-policy").Value.Type(); flagType != "stringArray" {
		t.Fatalf("--attachment-policy type = %q, want stringArray", flagType)
	}
}

func TestParseAttachmentPolicies(t *testing.T) {
	got, err := parseAttachmentPolicies([]string{
		"eth0=access:200",
		"eth1=direct",
		"eth2=trunk:200,201,299",
	})
	if err != nil {
		t.Fatalf("parseAttachmentPolicies() error = %v", err)
	}
	want := []fabric.PhysicalAttachmentPolicy{
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200},
		{Interface: "eth1", Mode: fabric.ModeDirect},
		{Interface: "eth2", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201, 299}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAttachmentPolicies() = %#v, want %#v", got, want)
	}
}

func TestParseAttachmentPoliciesRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value []string
		match string
	}{
		{name: "missing interface", value: []string{"=direct"}, match: "expected INTERFACE"},
		{name: "missing mode", value: []string{"eth0="}, match: "expected INTERFACE"},
		{name: "direct VLAN", value: []string{"eth0=direct:200"}, match: "expected INTERFACE"},
		{name: "zero VLAN", value: []string{"eth0=access:0"}, match: "between 1 and 4094"},
		{name: "reserved VLAN", value: []string{"eth0=access:4095"}, match: "between 1 and 4094"},
		{name: "non-numeric VLAN", value: []string{"eth0=access:lab"}, match: "between 1 and 4094"},
		{name: "empty trunk", value: []string{"eth0=trunk:"}, match: "at least one VLAN"},
		{name: "trunk zero VLAN", value: []string{"eth0=trunk:0"}, match: "between 1 and 4094"},
		{
			name:  "trunk reserved VLAN",
			value: []string{"eth0=trunk:4095"},
			match: "between 1 and 4094",
		},
		{
			name:  "trunk non-numeric VLAN",
			value: []string{"eth0=trunk:lab"},
			match: "between 1 and 4094",
		},
		{
			name:  "trunk duplicate VLAN",
			value: []string{"eth0=trunk:200,200"},
			match: "duplicate VLAN",
		},
		{
			name:  "trunk unordered VLANs",
			value: []string{"eth0=trunk:201,200"},
			match: "ascending order",
		},
		{name: "whitespace", value: []string{" eth0=direct"}, match: "expected INTERFACE"},
		{name: "duplicate", value: []string{"eth0=direct", "eth0=direct"}, match: "duplicate"},
		{
			name:  "duplicate interface",
			value: []string{"eth0=direct", "eth0=trunk:200"},
			match: "duplicate interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAttachmentPolicies(tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("parseAttachmentPolicies() error = %v, want containing %q", err, tt.match)
			}
		})
	}
}

func TestDaemonOptionsStruct(t *testing.T) {
	opts := &daemonOptions{
		listen:      ":9090",
		token:       "secret",
		storagePath: "/custom/path.db",
	}

	if opts.listen != ":9090" {
		t.Errorf("listen = %q, want %q", opts.listen, ":9090")
	}
	if opts.token != "secret" {
		t.Errorf("token = %q, want %q", opts.token, "secret")
	}
	if opts.storagePath != "/custom/path.db" {
		t.Errorf("storagePath = %q, want %q", opts.storagePath, "/custom/path.db")
	}
}
