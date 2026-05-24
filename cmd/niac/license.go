package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/license"
)

func addLicenseCommand(root *cobra.Command, _ *serviceOptions) {
	licenseCmd := &cobra.Command{
		Use:   "license",
		Short: "Manage license activation",
		Long: `The license command handles offline license activation and status
for NIAC. Without a license, NIAC runs in the Free tier (up to 10
simulated devices, common protocols). NIAC Pro ($599/yr) unlocks:

  • Unlimited simulated devices
  • Advanced protocols: BGP, OSPF, SNMPv3, NetBIOS, FTP, STP
  • Advanced IPv4/IPv6 stack features
  • Error injection (latency, loss, jitter, protocol faults)
  • Traffic shaping
  • Config templates
  • Multi-IP per simulated endpoint
  • PCAP ingest + analysis
  • REST API access (programmatic control)

A 14-day trial of the full Pro tier is available without a key.`,
	}

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show current license status",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseStatus() },
	})

	activateCmd := &cobra.Command{
		Use:   "activate",
		Short: "Activate a license key",
		Run:   func(cmd *cobra.Command, _ []string) { runLicenseActivate(cmd) },
	}
	activateCmd.Flags().StringP("key", "k", "", "License key to activate (XXXX-XXXX-XXXX-XXXX)")
	_ = activateCmd.MarkFlagRequired("key")
	licenseCmd.AddCommand(activateCmd)

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "deactivate",
		Short: "Remove the current license from this device",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseDeactivate() },
	})

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "trial",
		Short: "Start the 14-day Pro trial",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseTrial() },
	})

	root.AddCommand(licenseCmd)
}

func runLicenseStatus() {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	state := mgr.GetState()
	if state == nil {
		fmt.Fprintln(os.Stdout, "Tier:        Free (no license activated)")
		fmt.Fprintln(os.Stdout, "Run `niac license trial` to start a 14-day Pro trial,")
		fmt.Fprintln(os.Stdout, "or `niac license activate -k <KEY>` to enter a key.")
		return
	}

	if state.IsTrialMode {
		remaining := mgr.TrialDaysRemaining()
		fmt.Fprintln(os.Stdout, "Tier:        Trial (Pro features)")
		fmt.Fprintf(os.Stdout, "Days left:   %d of %d\n", remaining, license.TrialDays)
		if remaining <= 0 {
			fmt.Fprintln(os.Stdout, "Trial expired. Run `niac license activate -k <KEY>` to continue.")
		}
		return
	}

	fmt.Fprintf(os.Stdout, "Tier:        %s\n", state.Tier)
	fmt.Fprintf(os.Stdout, "Key:         %s\n", license.FormatKey(state.LicenseKey))
	fmt.Fprintf(os.Stdout, "Activated:   %s\n", state.ActivatedAt.Format("2006-01-02"))
	fmt.Fprintf(os.Stdout, "Expires:     %s\n", state.ExpiresAt.Format("2006-01-02"))
	fmt.Fprintf(os.Stdout, "Device:      %s\n", state.DeviceHash)
	if len(state.Features) > 0 {
		fmt.Fprintf(os.Stdout, "Features:    %d unlocked\n", len(state.Features))
	}
}

func runLicenseActivate(cmd *cobra.Command) {
	key, err := cmd.Flags().GetString("key")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading --key flag: %v\n", err)
		os.Exit(1)
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key is required")
		os.Exit(1)
	}

	mgr, mgrErr := license.NewManager()
	if mgrErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", mgrErr)
		os.Exit(1)
	}

	res := mgr.Activate(key)
	if !res.Success {
		fmt.Fprintf(os.Stderr, "Activation failed: %s\n", res.Message)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, res.Message)
}

func runLicenseDeactivate() {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if deactErr := mgr.Deactivate(); deactErr != nil {
		fmt.Fprintf(os.Stderr, "Deactivation failed: %v\n", deactErr)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "License removed. NIAC will run in the Free tier.")
}

func runLicenseTrial() {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	res := mgr.StartTrial()
	if !res.Success {
		fmt.Fprintf(os.Stderr, "Trial failed: %s\n", res.Message)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, res.Message)
}
