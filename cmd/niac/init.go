package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

const defaultInitOutputFile = "config.yaml"

func addInitCommand(root *cobra.Command, _ *serviceOptions) {
	initCmd := &cobra.Command{
		Use:   "init [output-file]",
		Short: "Interactive template wizard for quick configuration setup",
		Long: `Interactive wizard that helps you choose the right template and create
a configuration file for your network simulation needs.

The wizard will ask about your network type, size, and requirements,
then suggest the most appropriate template.`,
		Example: fmt.Sprintf(
			"  # Start interactive wizard\n  niac init\n\n  # Start wizard with specific output file\n  niac init my-network.yaml\n\n  # Quick workflow\n  niac init && niac validate %s",
			defaultInitOutputFile,
		),
		RunE: func(_ *cobra.Command, args []string) error {
			return runInit(args)
		},
	}

	root.AddCommand(initCmd)
}

func runInit(args []string) error {
	reader := bufio.NewReader(os.Stdin)

	printInitHeader()

	networkType, err := promptNetworkType(reader)
	if err != nil {
		return stopIfCancelled(err)
	}
	selectedTemplate, templateDesc := mapNetworkTypeToTemplate(networkType)

	fmt.Fprintln(os.Stdout)
	color.Green("Selected: %s", templateDesc)
	fmt.Fprintln(os.Stdout)

	tmpl, err := templates.Get(selectedTemplate)
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	printTemplateDetails(tmpl)

	outputFile, err := promptOutputFile(reader, args)
	if err != nil {
		return stopIfCancelled(err)
	}

	overwrite, err := confirmOverwriteIfExists(reader, outputFile)
	if err != nil {
		return stopIfCancelled(err)
	}
	if !overwrite {
		fmt.Fprintln(os.Stdout, "Aborted.")
		return nil
	}

	if writeErr := writeSafeFile(outputFile, []byte(tmpl.Content)); writeErr != nil {
		return fmt.Errorf("writing file: %w", writeErr)
	}

	printInitSuccess(outputFile, selectedTemplate)

	return nil
}

// printInitHeader displays the wizard header banner.
func printInitHeader() {
	_, _ = color.New(color.Bold, color.FgCyan).
		Println("\n+============================================================+")
	_, _ = color.New(color.Bold, color.FgCyan).
		Println("|         NIAC Configuration Template Wizard                |")
	_, _ = color.New(color.Bold, color.FgCyan).
		Print("+============================================================+\n")

	fmt.Fprintln(os.Stdout, "This wizard will help you choose the right template for your")
	fmt.Fprint(os.Stdout, "network simulation.\n")
}

// promptNetworkType displays network type menu and prompts for selection.
func promptNetworkType(reader *bufio.Reader) (string, error) {
	_, _ = fmt.Fprintln(
		os.Stdout,
		color.CyanString("1. What type of network are you simulating?"),
	)
	fmt.Fprintln(os.Stdout, "   a) Basic network (router + switch)")
	fmt.Fprintln(os.Stdout, "   b) Small office network")
	fmt.Fprintln(os.Stdout, "   c) Data center / enterprise core")
	fmt.Fprintln(os.Stdout, "   d) IoT / sensor network")
	fmt.Fprintln(os.Stdout, "   e) Enterprise campus (multi-building)")
	fmt.Fprintln(os.Stdout, "   f) Service provider / ISP")
	fmt.Fprintln(os.Stdout, "   g) Home network")
	fmt.Fprintln(os.Stdout, "   h) Test lab / protocol testing")
	fmt.Fprintln(os.Stdout)

	return mustPromptChoice(
		reader,
		"Enter your choice (a-h): ",
		[]string{"a", "b", "c", "d", "e", "f", "g", "h"},
	)
}

// mapNetworkTypeToTemplate maps user selection to template name and description.
func mapNetworkTypeToTemplate(networkType string) (string, string) {
	templateMap := map[string][2]string{
		"a": {"basic-network", "Basic Network - Simple router and switch setup"},
		"b": {"small-office", "Small Office - Router, switch, AP, and services"},
		"c": {"data-center", "Data Center - Multiple routers, switches, and servers"},
		"d": {"iot-network", "IoT Network - Sensor devices with lightweight protocols"},
		"e": {"enterprise-campus", "Enterprise Campus - Multi-building network"},
		"f": {"service-provider", "Service Provider - ISP-style topology"},
		"g": {"home-network", "Home Network - Residential gateway and devices"},
		"h": {"test-lab", "Test Lab - Comprehensive protocol testing"},
	}

	if entry, ok := templateMap[networkType]; ok {
		return entry[0], entry[1]
	}
	return "basic-network", "Basic Network - Simple router and switch setup"
}

// printTemplateDetails displays information about the selected template.
func printTemplateDetails(tmpl *templates.Template) {
	_, _ = fmt.Fprintln(os.Stdout, color.YellowString("Template Details:"))
	fmt.Fprintf(os.Stdout, "  Name: %s\n", tmpl.Name)
	fmt.Fprintf(os.Stdout, "  Description: %s\n", tmpl.Description)
	fmt.Fprintf(os.Stdout, "  Use case: %s\n", tmpl.UseCase)
	fmt.Fprintln(os.Stdout)
}

// promptOutputFile prompts for output filename or uses provided argument.
func promptOutputFile(reader *bufio.Reader, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	fmt.Fprintf(os.Stdout, "2. Enter output filename [%s]: ", defaultInitOutputFile)
	filename, readErr := readLine(reader)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", inputError(readErr)
	}
	if filename == "" {
		return defaultInitOutputFile, nil
	}
	return filename, nil
}

// confirmOverwriteIfExists checks if file exists and prompts for overwrite confirmation.
// Returns true if file doesn't exist or user confirms overwrite.
func confirmOverwriteIfExists(reader *bufio.Reader, outputFile string) (bool, error) {
	if _, statErr := statSafeFile(outputFile); statErr == nil {
		fmt.Fprintln(os.Stdout)
		color.Yellow("Warning: File %s already exists!", outputFile)
		return mustPromptYesNo(reader, "Overwrite? (y/n): ")
	}
	return true, nil
}

// printInitSuccess displays success message and next steps.
func printInitSuccess(outputFile, selectedTemplate string) {
	fmt.Fprintln(os.Stdout)
	color.Green("Successfully created %s", outputFile)
	fmt.Fprintln(os.Stdout)

	_, _ = color.New(color.Bold).Println("Next Steps:")
	fmt.Fprintln(os.Stdout)

	printNextStep("1", "Validate your configuration:",
		color.CyanString("niac validate %s", outputFile))
	printNextStep("2", "Edit the configuration (optional):",
		color.CyanString("vi %s", outputFile))
	printNextStep("3", "Run the simulation:",
		color.CyanString("sudo niac interactive en0 %s", outputFile))
	printNextStep("4", "Or use dry-run mode to test without running:",
		color.CyanString("niac --dry-run en0 %s", outputFile))

	_, _ = fmt.Fprintln(os.Stdout,
		color.YellowString("Tip:")+" To see what devices are in this configuration:",
	)
	_, _ = fmt.Fprintf(os.Stdout,
		"     %s\n",
		color.CyanString("niac template apply %s", selectedTemplate),
	)
	fmt.Fprintln(os.Stdout)
}

// printNextStep prints a numbered step with description and command.
func printNextStep(num, description, command string) {
	fmt.Fprintf(os.Stdout, "%s. %s\n", num, description)
	_, _ = fmt.Fprintf(os.Stdout, "   %s\n", command)
	fmt.Fprintln(os.Stdout)
}

// promptChoice prompts for a choice from a list of valid options.
func promptChoice(reader *bufio.Reader, prompt string, validChoices []string) (string, error) {
	for {
		fmt.Fprint(os.Stdout, prompt)
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}
		input = strings.ToLower(strings.TrimSpace(input))

		if slices.Contains(validChoices, input) {
			return input, nil
		}

		color.Red("Invalid choice. Please enter one of: %s", strings.Join(validChoices, ", "))
	}
}

// promptYesNo prompts for a yes/no answer.
func promptYesNo(reader *bufio.Reader, prompt string) (bool, error) {
	for {
		fmt.Fprint(os.Stdout, prompt)
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

// promptInt prompts for an integer within a range.
func promptInt(reader *bufio.Reader, prompt string, minValue, maxValue int) (int, error) {
	for {
		fmt.Fprint(os.Stdout, prompt)
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

		if value < minValue || value > maxValue {
			color.Red("Please enter a number between %d and %d", minValue, maxValue)
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
		return "", fmt.Errorf("failed to read line: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// errInputCancelled marks operator cancellation of a prompt (e.g. Ctrl-D) so
// callers can stop cleanly instead of treating it as a failure.
var errInputCancelled = errors.New("input cancelled")

// stopIfCancelled converts a cancelled-input sentinel into a clean stop
// (nil); any other error propagates unchanged.
func stopIfCancelled(err error) error {
	if errors.Is(err, errInputCancelled) {
		return nil
	}
	return err
}

// inputError converts a prompt read failure into a returned error. EOF means
// the operator cancelled (e.g. Ctrl-D), which is reported as
// errInputCancelled so callers can stop without treating it as a failure.
func inputError(err error) error {
	if errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stdout)
		color.Yellow("Input cancelled.")
		return errInputCancelled
	}
	return fmt.Errorf("reading input: %w", err)
}

func mustPromptChoice(reader *bufio.Reader, prompt string, validChoices []string) (string, error) {
	choice, err := promptChoice(reader, prompt, validChoices)
	if err != nil {
		return "", inputError(err)
	}
	return choice, nil
}

func mustPromptYesNo(reader *bufio.Reader, prompt string) (bool, error) {
	value, err := promptYesNo(reader, prompt)
	if err != nil {
		return false, inputError(err)
	}
	return value, nil
}

func mustPromptInt(reader *bufio.Reader, prompt string, minValue, maxValue int) (int, error) {
	value, err := promptInt(reader, prompt, minValue, maxValue)
	if err != nil {
		return 0, inputError(err)
	}
	return value, nil
}
