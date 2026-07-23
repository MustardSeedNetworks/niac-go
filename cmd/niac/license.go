package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

func addLicenseCommand(root *cobra.Command, _ *serviceOptions) {
	licenseCmd := &cobra.Command{
		Use:   "license",
		Short: "Manage license activation",
		Long: `The license command handles offline license activation and status
for NIAC.

Free: up to 10 simulated devices; Pro removes tier soft cap; absolute ceiling 1000.

NIAC Pro ($599/yr) unlocks:

  • Simulations above the Free tier device cap
  • Advanced protocols: SNMPv3, NetBIOS, FTP, STP
  • Advanced IPv4/IPv6 stack features
  • Error injection (latency, loss, jitter, protocol faults)
  • Config templates
  • Multi-IP per simulated endpoint
  • PCAP ingest + analysis
  • REST API access (programmatic control)

A 14-day trial of the full Pro tier is available without a key.`,
		Example: `  # Check the current tier / activation
  niac license status

  # Start a 14-day Pro trial (no key required)
  niac license trial

  # Activate a paid Pro key
  niac license activate -k XXXX-XXXX-XXXX-XXXX

  # Remove the license from this device
  niac license deactivate`,
	}

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show current license status",
		Long: `Print the current license tier, key, activation date, expiry,
device hash, and unlocked feature count. With no license active, prints the
Free-tier banner.`,
		Example: `  # Show the active license / trial status
  niac license status`,
		Run: func(_ *cobra.Command, _ []string) { runLicenseStatus() },
	})

	activateCmd := &cobra.Command{
		Use:   "activate",
		Short: "Activate a license key",
		Long: `Activate a NIAC license key on this device. Validation is fully
offline (rotor cipher + device fingerprint); no phone-home. The key is bound
to this device's hash and unlocks the tier features encoded in the key.`,
		Example: `  # Activate a Pro key
  niac license activate -k XXXX-XXXX-XXXX-XXXX

  # The flag is required — this prints the usage
  niac license activate`,
		Run: func(cmd *cobra.Command, _ []string) { runLicenseActivate(cmd) },
	}
	activateCmd.Flags().StringP("key", "k", "", "License key to activate (XXXX-XXXX-XXXX-XXXX)")
	_ = activateCmd.MarkFlagRequired("key")
	licenseCmd.AddCommand(activateCmd)

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "deactivate",
		Short: "Remove the current license from this device",
		Long: `Remove the activated license from this device. After deactivation
NIAC reverts to the Free tier. The license key itself remains valid and can
be activated on another device.`,
		Example: `  # Deactivate the current license (revert to Free tier)
  niac license deactivate`,
		Run: func(_ *cobra.Command, _ []string) { runLicenseDeactivate() },
	})

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "trial",
		Short: "Start the 14-day Pro trial",
		Long: `Begin a one-time 14-day trial of the full Pro tier. The trial is
tied to this device's hash and cannot be reset after expiry — activate a
real key with ` + "`niac license activate -k ...`" + ` to continue using Pro
features.`,
		Example: `  # Start the 14-day Pro trial
  niac license trial`,
		Run: func(_ *cobra.Command, _ []string) { runLicenseTrial() },
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

	fmt.Fprintf(os.Stdout, "Tier:        %s\n", license.Tier(state.Tier))
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
