package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestSimulationCommandExposesDaemonLifecycle(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "niac"}
	addSimulationCommand(root, new(serviceOptions))
	command := findSubcommand(root, "simulation")
	if command == nil {
		t.Fatal("simulation command is not registered")
	}
	for _, name := range []string{"preflight", "start", "select", "stop"} {
		if findSubcommand(command, name) == nil {
			t.Errorf("simulation %s is not registered", name)
		}
	}
	for _, name := range []string{"api", "cacert", "insecure"} {
		if command.PersistentFlags().Lookup(name) == nil {
			t.Errorf("simulation command has no --%s flag", name)
		}
	}
}

func TestWritePreflightReportPrintsDiagnosticsAndFailsWhenUnsafe(t *testing.T) {
	t.Parallel()

	report := fabric.NewReport()
	report.Diagnostics = append(report.Diagnostics, fabric.Diagnostic{Message: "interface unavailable"})
	var output bytes.Buffer
	err := writePreflightReport(&output, &report)
	if !errors.Is(err, errUnsafePreflight) {
		t.Fatalf("error = %v, want unsafe preflight", err)
	}
	if !strings.Contains(output.String(), "interface unavailable") {
		t.Fatalf("output omitted diagnostics: %s", output.String())
	}
}

// `daemon --once` names the binding --attachment-mode, and an operator who
// reached for that name on `simulation start` got an unknown-flag error
// (niac-go#2211). One name on every surface that binds an attachment.
func TestSimulationRequestCommandsNameTheBindingLikeDaemonOnce(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "niac"}
	addSimulationCommand(root, new(serviceOptions))
	simulation := findSubcommand(root, "simulation")
	for _, name := range []string{"preflight", "start"} {
		command := findSubcommand(simulation, name)
		for _, flag := range []string{"attachment", "attachment-mode", "access-vlan"} {
			if command.Flags().Lookup(flag) == nil {
				t.Errorf("simulation %s has no --%s flag", name, flag)
			}
		}
		if command.Flags().Lookup("mode") != nil {
			t.Errorf("simulation %s still spells the binding --mode", name)
		}
	}
}

func TestReportsUnsetModeFindsOnlyTheMissingBinding(t *testing.T) {
	t.Parallel()

	unset := fabric.NewReport()
	unset.Diagnostics = []fabric.Diagnostic{{Code: fabric.CodeInvalidAttachmentMode}}
	denied := fabric.NewReport()
	denied.Diagnostics = []fabric.Diagnostic{{Code: fabric.CodeAttachmentPolicyDenied}}
	if !reportsUnsetMode(&unset) {
		t.Error("reportsUnsetMode(invalid_attachment_mode) = false")
	}
	if reportsUnsetMode(&denied) {
		t.Error("reportsUnsetMode(attachment_policy_denied) = true")
	}
}
