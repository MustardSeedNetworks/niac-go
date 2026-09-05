package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type validateOptions struct {
	verbose bool
	json    bool
}

func addValidateCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(validateOptions)

	validateCmd := &cobra.Command{
		Use:   "validate <config-file>",
		Short: "Validate a NIAC configuration file",
		Long: `Validate a NIAC configuration file for errors and warnings.

This command performs comprehensive validation including:
- Device name uniqueness
- MAC address format and duplicates
- IP address duplicates
- SNMP trap configurations (thresholds, receivers)
- DNS record formats
- Protocol-specific validation

Exit codes:
  0 - Configuration is valid
  1 - Configuration has errors`,
		Example: `  # Validate a configuration file
  niac validate config.yaml

  # Verbose output with details
  niac validate config.yaml --verbose

  # JSON output for CI/CD pipeline
  niac validate config.yaml --json > validation-results.json

  # Use in a CI/CD script
  if niac validate config.yaml; then
    echo "Config is valid, deploying..."
  else
    echo "Config validation failed!"
    exit 1
  fi`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runValidate(args, options)
		},
	}

	validateCmd.Flags().BoolVarP(&options.verbose, "verbose", "v", false, "Show detailed validation information")
	validateCmd.Flags().BoolVar(&options.json, "json", false, "Output validation results as JSON")

	root.AddCommand(validateCmd)
}

// outputJSONResult outputs validation results as JSON.
func outputJSONResult(result *config.ListError) error {
	jsonOutput, jsonErr := result.ToJSON()
	if jsonErr != nil {
		return fmt.Errorf("generating JSON output: %w", jsonErr)
	}

	fmt.Fprintln(os.Stdout, jsonOutput)

	return nil
}

// outputTextResult outputs validation results as human-readable text.
func outputTextResult(result *config.ListError, configFile string, verbose bool, deviceCount int) {
	if result.HasErrors() || result.HasWarnings() {
		fmt.Fprintln(os.Stdout, result.Format())

		return
	}

	logging.Successf("Configuration is valid: %s", configFile)

	if verbose {
		fmt.Fprintf(os.Stdout, "\nDevices: %d\n", deviceCount)
	}
}

func runValidate(args []string, options *validateOptions) error {
	configFile := args[0]

	// Check if file exists
	if _, statErr := os.Stat(configFile); os.IsNotExist(statErr) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Validate configuration
	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)
	addFabricFindings(result, cfg, configFile)

	// Output results
	if options.json {
		if jsonErr := outputJSONResult(result); jsonErr != nil {
			return jsonErr
		}
	} else {
		outputTextResult(result, configFile, options.verbose, cfg.DeviceCount())
	}

	// Report an invalid configuration as a failure
	if !result.Valid {
		return errConfigInvalid
	}

	return nil
}

// errConfigInvalid marks a validation run whose findings have already been
// printed, so the caller adds an exit code rather than another message.
var errConfigInvalid = errors.New("configuration validation failed")

// addFabricFindings folds the fabric compiler's findings into the validation
// result, so `niac validate` refuses exactly what the daemon refuses to start.
//
// Semantic validation alone passed files that preflight rejected on six
// counts, and validate then printed "Configuration is valid" for a scenario
// the daemon would not run (P1b-4). The compiler's stable codes travel with
// each finding so an operator can match a validate line to a preflight line.
func addFabricFindings(result *config.ListError, cfg *config.Config, configFile string) {
	// A flat scenario has no fabric to compile; its interfaces name no
	// network, which the compiler would read as references to a network that
	// does not exist.
	if !fabric.IsRouted(cfg) {
		return
	}
	for _, diagnostic := range fabric.CompileConfig(cfg).Diagnostics {
		finding := config.NewConfigError(configFile, diagnostic.Field, diagnostic.Message)
		finding.Code = string(diagnostic.Code)
		result.Add(finding)
	}
}
