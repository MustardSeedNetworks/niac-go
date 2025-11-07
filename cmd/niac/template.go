package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

//go:embed templates
var templatesFS embed.FS

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage configuration templates",
	Long:  `List, show, and use pre-built configuration templates for common scenarios.`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Run:   runTemplateList,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Show template contents",
	Args:  cobra.ExactArgs(1),
	Run:   runTemplateShow,
}

var templateUseCmd = &cobra.Command{
	Use:   "use <template-name> <output-file>",
	Short: "Copy template to a new file",
	Args:  cobra.ExactArgs(2),
	Run:   runTemplateUse,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateUseCmd)
}

func runTemplateList(cmd *cobra.Command, args []string) {
	templates := []struct {
		name string
		desc string
	}{
		{"minimal", "Single device with basic protocols"},
		{"router", "Enterprise router with full protocol support"},
		{"switch", "Layer 2/3 switch with STP and VLAN support"},
		{"ap", "Enterprise Wi-Fi access point"},
		{"server", "Multi-service server (DHCP, DNS, HTTP)"},
		{"iot", "Lightweight IoT sensor device"},
		{"complete", "Multi-device network simulation"},
	}

	color.New(color.Bold).Println("Available Templates:")
	fmt.Println()

	for _, t := range templates {
		color.New(color.FgCyan).Printf("  %-12s", t.name)
		fmt.Printf(" - %s\n", t.desc)
	}

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  niac template show <template-name>")
	fmt.Println("  niac template use <template-name> <output-file>")
}

func runTemplateShow(cmd *cobra.Command, args []string) {
	templateName := args[0]
	content, err := getTemplate(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
	fmt.Print(content)
}

func runTemplateUse(cmd *cobra.Command, args []string) {
	templateName := args[0]
	outputFile := args[1]

	// Check if output file exists
	if _, err := os.Stat(outputFile); err == nil {
		color.Red("Error: file already exists: %s", outputFile)
		os.Exit(1)
	}

	// Get template content
	content, err := getTemplate(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		color.Red("Error writing file: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Created %s from %s template", outputFile, templateName)
}

func getTemplate(name string) (string, error) {
	// Try with and without .yaml extension
	paths := []string{
		filepath.Join("templates", name+".yaml"),
		filepath.Join("templates", name),
	}

	var content []byte
	var err error

	for _, path := range paths {
		file, openErr := templatesFS.Open(path)
		if openErr == nil {
			content, err = io.ReadAll(file)
			file.Close()
			if err == nil {
				return string(content), nil
			}
		}
	}

	// List available templates for better error message
	entries, _ := templatesFS.ReadDir("templates")
	available := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			available = append(available, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}

	return "", fmt.Errorf("template not found: %s\nAvailable templates: %s",
		name, strings.Join(available, ", "))
}
