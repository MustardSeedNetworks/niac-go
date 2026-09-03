package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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
