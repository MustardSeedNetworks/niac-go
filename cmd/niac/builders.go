package main

import "github.com/spf13/cobra"

// commandBuilders returns every builder that populates the root command, in
// the order they are registered. main, the docs generator and the help
// completeness test all build the tree from this one list, so a command added
// here is documented and checked without a second edit.
func commandBuilders(info versionInfo) []func(*cobra.Command, *serviceOptions) {
	return []func(*cobra.Command, *serviceOptions){
		func(root *cobra.Command, services *serviceOptions) { addRunCommand(root, services, info) },
		func(root *cobra.Command, _ *serviceOptions) { addCompletionCommand(root) },
		addAnalyzeCommand,
		addAnalyzePcapCommand,
		addConfigCommand,
		addContentCommand,
		func(root *cobra.Command, _ *serviceOptions) { addDaemonCommand(root, info) },
		addDumpCommand,
		addInitCommand,
		func(root *cobra.Command, _ *serviceOptions) { addInstallCACommand(root) },
		addListCommand,
		addLogsCommand,
		func(root *cobra.Command, _ *serviceOptions) { addManCommand(root, info) },
		func(root *cobra.Command, _ *serviceOptions) { addDocsCommand(root) },
		addMibZipCommand,
		addMonitorCommand,
		addNeighborsCommand,
		addSanitizeCommand,
		addServiceCommand,
		addStatusCommand,
		addTemplateCommand,
		addTopologyCommand,
		addValidateCommand,
		func(root *cobra.Command, _ *serviceOptions) { addVersionCommand(root, info) },
	}
}
