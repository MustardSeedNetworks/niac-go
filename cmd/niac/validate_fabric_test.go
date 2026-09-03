package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// routedScenarioWithFabricDefects carries two defects only the fabric compiler
// finds: an interface address that parses but sits outside its own network,
// and a DHCP pool outside the network it serves. `niac validate` used to call
// the semantic validator alone and pronounce this file valid, while the daemon
// refused to start it.
const routedScenarioWithFabricDefects = `networks:
  - name: clinical
    subnet: 10.20.0.0/24
devices:
  - name: core
    type: router
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.9.1/24
  - name: edge
    type: switch
    mac: "00:11:22:33:44:66"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.0.2/24
    dhcp:
      pool_start: 10.99.0.10
      pool_end: 10.99.0.20
`

func writeScenario(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestValidateRejectsFabricDefects is the CLI half of P1b-4: validate must
// refuse what preflight refuses.
func TestValidateRejectsFabricDefects(t *testing.T) {
	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", writeScenario(t, routedScenarioWithFabricDefects)})

	err := root.Execute()

	if err == nil {
		t.Fatal("validate accepted a scenario the daemon refuses to start")
	}
	if !errors.Is(err, errConfigInvalid) {
		t.Fatalf("error = %v, want errConfigInvalid", err)
	}
}

// TestValidateReportsPreflightCodes pins the vocabulary: the codes validate
// prints are the codes the fabric compiler emits for the same file, so an
// operator can match a validate finding to a preflight finding.
func TestValidateReportsPreflightCodes(t *testing.T) {
	path := writeScenario(t, routedScenarioWithFabricDefects)
	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", "--json", path})

	stdout := os.Stdout
	read, write, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stdout = write
	execErr := root.Execute()
	if closeErr := write.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	os.Stdout = stdout
	if execErr == nil {
		t.Fatal("validate --json accepted a scenario the daemon refuses to start")
	}

	var report struct {
		Errors []struct {
			Code  string `json:"code"`
			Field string `json:"field"`
		} `json:"errors"`
	}
	if decodeErr := json.NewDecoder(read).Decode(&report); decodeErr != nil {
		t.Fatalf("decode validate --json output: %v", decodeErr)
	}
	got := make([]string, 0, len(report.Errors))
	for _, finding := range report.Errors {
		got = append(got, finding.Code)
	}
	slices.Sort(got)
	want := []string{
		string(fabric.CodeAddressOutsideNetwork),
		string(fabric.CodeDHCPPoolOutsideNetwork),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("codes = %v, want %v", got, want)
	}
}

// TestValidateAcceptsFlatScenario guards the routed gate. A flat scenario's
// interfaces name no network, and compiling it anyway invents an
// unknown_network finding no other surface reports.
func TestValidateAcceptsFlatScenario(t *testing.T) {
	const flat = `devices:
  - name: sw1
    type: switch
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        address: 192.168.1.10/24
`
	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", writeScenario(t, flat)})

	if err := root.Execute(); err != nil {
		t.Fatalf("validate rejected a valid flat scenario: %v", err)
	}
}
