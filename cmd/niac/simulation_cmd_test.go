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
