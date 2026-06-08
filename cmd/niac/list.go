package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/content"
	"github.com/krisarmstrong/niac-go/internal/library"
	"github.com/krisarmstrong/niac-go/internal/templates"
)

type listOptions struct {
	root string
	all  bool
}

func addListCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(listOptions)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List interfaces and demo content",
		Long: `List the same operator-facing resources the legacy Java demo
wrapper exposed: network interfaces, runnable scenarios, SNMP walks, and
packet captures.

Scenario output includes built-in templates and installed library networks.
Walk and capture output reads the on-disk content library.`,
		Example: `  # List usable network interfaces
  niac list interfaces

  # List runnable scenarios
  niac list scenarios

  # List SNMP walks, optionally scoped by vendor/path prefix
  niac list walks
  niac list walks cisco

  # List packet captures
  niac list captures`,
	}

	listCmd.PersistentFlags().
		StringVar(&options.root, "root", "", "Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")

	listCmd.AddCommand(newListInterfacesCmd(options))
	listCmd.AddCommand(newListScenariosCmd(options))
	listCmd.AddCommand(newListWalksCmd(options))
	listCmd.AddCommand(newListCapturesCmd(options))
	root.AddCommand(listCmd)
}

func newListInterfacesCmd(options *listOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interfaces",
		Short: "List available network interfaces",
		Long: `List network interfaces available to NIAC. By default this shows
usable interfaces only (ethernet, Wi-Fi, and loopback). Use --all to include
every interface returned by libpcap.`,
		Example: `  # List usable interfaces
  niac list interfaces

  # Include every libpcap-visible interface
  niac list interfaces --all`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListInterfaces(options)
		},
	}
	cmd.Flags().BoolVar(&options.all, "all", false, "Show all interfaces instead of only usable ones")
	return cmd
}

func runListInterfaces(options *listOptions) error {
	var (
		devices []capture.PcapInterface
		err     error
	)
	if options.all {
		devices, err = capture.GetAllInterfaces()
	} else {
		devices, err = capture.GetUsableInterfaces()
	}
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		fmt.Fprintln(os.Stdout, "No interfaces found")
		return nil
	}

	fmt.Fprintf(os.Stdout, "%-24s %s\n", "Interface", "Description")
	for _, device := range devices {
		desc := device.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(os.Stdout, "%-24s %s\n", device.Name, desc)
		for _, addr := range device.Addresses {
			fmt.Fprintf(os.Stdout, "  IP: %s\n", addr.IP)
		}
	}
	return nil
}

func newListScenariosCmd(options *listOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "scenarios",
		Short: "List runnable scenarios",
		Long: `List runnable scenario sources. Built-in templates are always
available. Installed library networks are shown when the content library can
be opened.`,
		Example: `  # List built-in and installed scenarios
  niac list scenarios

  # Inspect a non-default library
  niac list scenarios --root /var/lib/niac/library`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListScenarios(options)
		},
	}
}

func runListScenarios(options *listOptions) error {
	fmt.Fprintln(os.Stdout, "Built-in templates:")
	for _, tmpl := range templates.List() {
		fmt.Fprintf(os.Stdout, "  %-24s %s\n", tmpl.Name, tmpl.Description)
	}

	lib, err := openListLibrary(options.root)
	if err != nil {
		fmt.Fprintf(os.Stdout, "\nLibrary networks: unavailable (%v)\n", err)
		return nil
	}

	networks, err := lib.ListNetworks()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "\nLibrary networks: %s\n", lib.Root())
	if len(networks) == 0 {
		fmt.Fprintln(os.Stdout, "  none")
		return nil
	}
	for _, network := range networks {
		status := "ok"
		if !network.Valid {
			status = "invalid"
		}
		fmt.Fprintf(os.Stdout, "  %-24s devices=%d source=%s status=%s",
			network.Name, network.DeviceCount, network.Source, status)
		if network.Description != "" {
			fmt.Fprintf(os.Stdout, " - %s", network.Description)
		}
		fmt.Fprintln(os.Stdout)
	}
	return nil
}

func newListWalksCmd(options *listOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "walks [vendor-or-prefix]",
		Short: "List SNMP walk files",
		Long: `List SNMP walk files from the content library. If a prefix is
provided, only matching walk paths are shown. This mirrors the Java demo
wrapper's vendor browsing flow while preserving Go's library layout.`,
		Example: `  # List all SNMP walks
  niac list walks

  # List Cisco walks
  niac list walks cisco`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prefix := ""
			if len(args) == 1 {
				prefix = args[0]
			}
			return runListFiles(options, library.KindWalks, prefix)
		},
	}
}

func newListCapturesCmd(options *listOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "captures",
		Short: "List packet captures",
		Long: `List packet captures from the content library. Captures are the
PCAP/PCAPNG/CAP files used by replay, packet inspection, and offline analysis
workflows.`,
		Example: `  # List installed captures
  niac list captures`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListFiles(options, library.KindPcaps, "")
		},
	}
}

func runListFiles(options *listOptions, kind library.Kind, prefix string) error {
	lib, err := openListLibrary(options.root)
	if err != nil {
		return err
	}

	files, err := lib.ListFiles(kind)
	if err != nil {
		return err
	}

	prefix = strings.Trim(filepath.ToSlash(prefix), "/")
	title := string(kind)
	if prefix != "" {
		title += " matching " + prefix
	}
	fmt.Fprintf(os.Stdout, "%s: %s\n", title, lib.SubDir(kind))

	count := 0
	for _, file := range files {
		if prefix != "" && !strings.HasPrefix(file.Name, prefix) {
			continue
		}
		count++
		fmt.Fprintf(os.Stdout, "  %-48s %10s source=%s\n",
			file.Name, content.HumanBytes(file.SizeBytes), file.Source)
	}
	if count == 0 {
		fmt.Fprintln(os.Stdout, "  none")
	}
	return nil
}

func openListLibrary(root string) (*library.Library, error) {
	libRoot := root
	if libRoot == "" {
		libRoot = library.DefaultRoot()
	}
	return library.Open(libRoot)
}
