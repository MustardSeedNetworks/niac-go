package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

var errUnsafePreflight = errors.New("scenario preflight is unsafe")

func simulationClient(options *simulationCLIOptions) (*cliclient.Client, error) {
	return newCLIClient(options.api, options.caCert, options.insecure)
}

func simulationRequest(options *simulationCLIOptions) cliclient.SimulationRequest {
	return cliclient.SimulationRequest{
		SessionID: options.session, Interface: options.iface, ConfigPath: options.config,
		TemplateName: options.template, Attachment: options.attachment,
		AttachmentMode: fabric.AttachmentMode(options.mode), AccessVLAN: options.accessVLAN,
	}
}

func runSimulationPreflight(ctx context.Context, options *simulationCLIOptions) error {
	client, err := simulationClient(options)
	if err != nil {
		return err
	}
	report, err := client.PreflightSimulation(ctx, simulationRequest(options))
	if err != nil {
		return err
	}
	return writePreflightReport(os.Stdout, report)
}

func runSimulationStart(ctx context.Context, options *simulationCLIOptions) error {
	client, err := simulationClient(options)
	if err != nil {
		return err
	}
	status, err := client.StartSimulation(ctx, simulationRequest(options))
	if err != nil {
		return err
	}
	return writeSimulationJSON(status)
}

func runSimulationSelect(ctx context.Context, options *simulationCLIOptions, session string) error {
	client, err := simulationClient(options)
	if err != nil {
		return err
	}
	status, err := client.SelectSimulation(ctx, session)
	if err != nil {
		return err
	}
	return writeSimulationJSON(status)
}

func runSimulationStop(ctx context.Context, options *simulationCLIOptions, session string) error {
	client, err := simulationClient(options)
	if err != nil {
		return err
	}
	if err = client.StopSimulation(ctx, session); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "Stopped scenario %s\n", session)
	return err
}

func writeSimulationJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writePreflightReport(output io.Writer, report *fabric.Report) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return err
	}
	if !report.Safe {
		return errUnsafePreflight
	}
	return nil
}
