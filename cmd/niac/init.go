package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/krisarmstrong/niac-go/pkg/templates"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [output-file]",
	Short: "Interactive template wizard for quick configuration setup",
	Long: `Interactive wizard that helps you choose the right template and create
a configuration file for your network simulation needs.

The wizard will ask about your network type, size, and requirements,
then suggest the most appropriate template.`,
	Example: `  # Start interactive wizard
  niac init

  # Start wizard with specific output file
  niac init my-network.yaml

  # Quick workflow
  niac init && niac validate config.yaml`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	// Print header
	color.New(color.Bold, color.FgCyan).Println("\n╔════════════════════════════════════════════════════════════╗")
	color.New(color.Bold, color.FgCyan).Println("║         NIAC Configuration Template Wizard                ║")
	color.New(color.Bold, color.FgCyan).Print("╚════════════════════════════════════════════════════════════╝\n")

	fmt.Println("This wizard will help you choose the right template for your")
	fmt.Print("network simulation.\n")

	// Question 1: Network type
	fmt.Println(color.CyanString("1. What type of network are you simulating?"))
	fmt.Println("   a) Basic network (router + switch)")
	fmt.Println("   b) Small office network")
	fmt.Println("   c) Data center / enterprise core")
	fmt.Println("   d) IoT / sensor network")
	fmt.Println("   e) Enterprise campus (multi-building)")
	fmt.Println("   f) Service provider / ISP")
	fmt.Println("   g) Home network")
	fmt.Println("   h) Test lab / protocol testing")
	fmt.Println()

	networkType := mustPromptChoice(reader, "Enter your choice (a-h): ", []string{"a", "b", "c", "d", "e", "f", "g", "h"})

	// Map choice to template
	var selectedTemplate string
	var templateDesc string

	switch networkType {
	case "a":
		selectedTemplate = "basic-network"
		templateDesc = "Basic Network - Simple router and switch setup"
	case "b":
		selectedTemplate = "small-office"
		templateDesc = "Small Office - Router, switch, AP, and services"
	case "c":
		selectedTemplate = "data-center"
		templateDesc = "Data Center - Multiple routers, switches, and servers"
	case "d":
		selectedTemplate = "iot-network"
		templateDesc = "IoT Network - Sensor devices with lightweight protocols"
	case "e":
		selectedTemplate = "enterprise-campus"
		templateDesc = "Enterprise Campus - Multi-building network"
	case "f":
		selectedTemplate = "service-provider"
		templateDesc = "Service Provider - ISP-style topology"
	case "g":
		selectedTemplate = "home-network"
		templateDesc = "Home Network - Residential gateway and devices"
	case "h":
		selectedTemplate = "test-lab"
		templateDesc = "Test Lab - Comprehensive protocol testing"
	}

	fmt.Println()
	color.Green("✓ Selected: %s", templateDesc)
	fmt.Println()

	// Get template
	tmpl, err := templates.Get(selectedTemplate)
	if err != nil {
		color.Red("Error loading template: %v", err)
		os.Exit(1)
	}

	// Show template details
	fmt.Println(color.YellowString("Template Details:"))
	fmt.Printf("  Name: %s\n", tmpl.Name)
	fmt.Printf("  Description: %s\n", tmpl.Description)
	fmt.Printf("  Use case: %s\n", tmpl.UseCase)
	fmt.Println()

	// Question 2: Output filename
	var outputFile string
	if len(args) > 0 {
		outputFile = args[0]
	} else {
		fmt.Print("2. Enter output filename [config.yaml]: ")
		filename, err := readLine(reader)
		if err != nil && !errors.Is(err, io.EOF) {
			handleInputError(err)
		}
		if filename == "" {
			outputFile = "config.yaml"
		} else {
			outputFile = filename
		}
	}

	// Check if file exists
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Println()
		color.Yellow("Warning: File %s already exists!", outputFile)
		overwrite := mustPromptYesNo(reader, "Overwrite? (y/n): ")
		if !overwrite {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	// Write template to file
	if err := os.WriteFile(outputFile, []byte(tmpl.Content), 0644); err != nil {
		color.Red("Error writing file: %v", err)
		os.Exit(1)
	}

	// Success message
	fmt.Println()
	color.Green("✓ Successfully created %s", outputFile)
	fmt.Println()

	// Next steps
	color.New(color.Bold).Println("Next Steps:")
	fmt.Println()
	fmt.Println("1. Validate your configuration:")
	fmt.Printf("   %s\n", color.CyanString("niac validate %s", outputFile))
	fmt.Println()
	fmt.Println("2. Edit the configuration (optional):")
	fmt.Printf("   %s\n", color.CyanString("vi %s", outputFile))
	fmt.Println()
	fmt.Println("3. Run the simulation:")
	fmt.Printf("   %s\n", color.CyanString("sudo niac interactive en0 %s", outputFile))
	fmt.Println()
	fmt.Println("4. Or use dry-run mode to test without running:")
	fmt.Printf("   %s\n", color.CyanString("niac --dry-run en0 %s", outputFile))
	fmt.Println()

	// Optional: Show device count
	fmt.Println(color.YellowString("Tip:") + " To see what devices are in this configuration:")
	fmt.Printf("     %s\n", color.CyanString("niac template apply %s", selectedTemplate))
	fmt.Println()
}

// promptChoice prompts for a choice from a list of valid options
func promptChoice(reader *bufio.Reader, prompt string, validChoices []string) (string, error) {
	for {
		fmt.Print(prompt)
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}
		input = strings.ToLower(strings.TrimSpace(input))

		for _, choice := range validChoices {
			if input == choice {
				return input, nil
			}
		}

		color.Red("Invalid choice. Please enter one of: %s", strings.Join(validChoices, ", "))
	}
}

// promptYesNo prompts for a yes/no answer
func promptYesNo(reader *bufio.Reader, prompt string) (bool, error) {
	for {
		fmt.Print(prompt)
		input, err := readLine(reader)
		if err != nil {
			return false, err
		}
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "yes" {
			return true, nil
		}
		if input == "n" || input == "no" {
			return false, nil
		}

		color.Red("Please enter 'y' or 'n'")
	}
}

// promptInt prompts for an integer within a range
func promptInt(reader *bufio.Reader, prompt string, min, max int) (int, error) {
	for {
		fmt.Print(prompt)
		input, err := readLine(reader)
		if err != nil {
			return 0, err
		}
		input = strings.TrimSpace(input)

		value, err := strconv.Atoi(input)
		if err != nil {
			color.Red("Please enter a valid number")
			continue
		}

		if value < min || value > max {
			color.Red("Please enter a number between %d and %d", min, max)
			continue
		}

		return value, nil
	}
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			line = strings.TrimSpace(line)
			if line == "" {
				return "", io.EOF
			}
			return line, nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func handleInputError(err error) {
	if errors.Is(err, io.EOF) {
		fmt.Println()
		color.Yellow("Input cancelled.")
		os.Exit(0)
	}
	color.Red("Error reading input: %v", err)
	os.Exit(1)
}

func mustPromptChoice(reader *bufio.Reader, prompt string, validChoices []string) string {
	choice, err := promptChoice(reader, prompt, validChoices)
	if err != nil {
		handleInputError(err)
	}
	return choice
}

func mustPromptYesNo(reader *bufio.Reader, prompt string) bool {
	value, err := promptYesNo(reader, prompt)
	if err != nil {
		handleInputError(err)
	}
	return value
}

func mustPromptInt(reader *bufio.Reader, prompt string, min, max int) int {
	value, err := promptInt(reader, prompt, min, max)
	if err != nil {
		handleInputError(err)
	}
	return value
}
