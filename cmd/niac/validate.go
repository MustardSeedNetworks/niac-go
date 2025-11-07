package main

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	validateVerbose bool
	validateJSON    bool
)

var validateCmd = &cobra.Command{
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
	Run:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolVarP(&validateVerbose, "verbose", "v", false, "Show detailed validation information")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output validation results as JSON")
}

func runValidate(cmd *cobra.Command, args []string) {
	configFile := args[0]

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logging.Error("Configuration file not found: %s", configFile)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		logging.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Validate configuration
	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	// Output results
	if validateJSON {
		jsonOutput, err := result.ToJSON()
		if err != nil {
			logging.Error("Failed to generate JSON output: %v", err)
			os.Exit(1)
		}
		fmt.Println(jsonOutput)
	} else {
		if result.HasErrors() || result.HasWarnings() {
			fmt.Println(result.Format())
		} else {
			logging.Success("Configuration is valid: %s", configFile)
			if validateVerbose {
				fmt.Printf("\nDevices: %d\n", len(cfg.Devices))
			}
		}
	}

	// Exit with appropriate code
	if !result.Valid {
		os.Exit(1)
	}
}
