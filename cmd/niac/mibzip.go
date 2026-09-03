package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosnmp/gosnmp"
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func addMibZipCommand(root *cobra.Command, _ *serviceOptions) {
	mibZipCmd := &cobra.Command{
		Use:   "mibzip",
		Short: "Convert SNMP walk files to and from MibZip format",
		Long: `Convert SNMP walk files to and from NIAC MibZip format.

MibZip is the compact binary SNMP walk format used by the legacy Java
implementation. Use this command to preserve legacy walk workflows while
keeping walk conversion available in the modern Go toolchain.`,
		Example: `  # Compress a text snmpwalk file
  niac mibzip compress cisco.walk cisco.mz

  # Expand a MibZip file back to text
  niac mibzip expand cisco.mz cisco-expanded.walk

  # Inspect whether a file is MibZip
  niac mibzip inspect cisco.mz`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	compressCmd := &cobra.Command{
		Use:   "compress <walk-file> <mibzip-file>",
		Short: "Compress a text SNMP walk file",
		Long: `Compress a text SNMP walk file into NIAC MibZip binary format.

The input file is parsed as a standard snmpwalk-style text file. The output
file is created with owner-only permissions.`,
		Example: `  niac mibzip compress walks/cisco-c9300.walk walks/cisco-c9300.mz`,
		Args:    cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runMibZipCompress(args[0], args[1])
		},
	}

	expandCmd := &cobra.Command{
		Use:   "expand <mibzip-file> <walk-file>",
		Short: "Expand a MibZip file to text",
		Long: `Expand a NIAC MibZip binary file back into a text SNMP walk file.

The expanded file is useful for validating compressed walks, reviewing legacy
assets, or converting MibZip content back into the shared walk catalog.`,
		Example: `  niac mibzip expand walks/cisco-c9300.mz walks/cisco-c9300-expanded.walk`,
		Args:    cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runMibZipExpand(args[0], args[1])
		},
	}

	inspectCmd := &cobra.Command{
		Use:   "inspect <file>",
		Short: "Inspect a file for MibZip format",
		Long: `Inspect a file and report whether it uses NIAC MibZip format.

For MibZip files, the command also reports the number of expanded walk entries
so operators can quickly verify the compressed asset is readable.`,
		Example: `  niac mibzip inspect walks/cisco-c9300.mz`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runMibZipInspect(args[0])
		},
	}

	mibZipCmd.AddCommand(compressCmd, expandCmd, inspectCmd)
	root.AddCommand(mibZipCmd)
}

func runMibZipCompress(inputFile, outputFile string) error {
	if err := snmp.CompressMibZipFile(inputFile, outputFile); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Compressed %s -> %s\n", inputFile, outputFile)
	return nil
}

func runMibZipExpand(inputFile, outputFile string) error {
	entries, err := snmp.ParseMibZipFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to expand mibzip file: %w", err)
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(filepath.Clean(outputFile)), 0o750); mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	content := formatWalkEntries(entries)
	if writeErr := os.WriteFile(filepath.Clean(outputFile), []byte(content), 0o600); writeErr != nil {
		return fmt.Errorf("failed to write expanded walk file: %w", writeErr)
	}

	fmt.Fprintf(os.Stdout, "Expanded %s -> %s (%d entries)\n", inputFile, outputFile, len(entries))
	return nil
}

func runMibZipInspect(filename string) error {
	isMibZip, err := snmp.IsMibZipFile(filename)
	if err != nil {
		return err
	}

	if !isMibZip {
		fmt.Fprintf(os.Stdout, "%s: text or unknown format\n", filename)
		return nil
	}

	entries, err := snmp.ParseMibZipFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read mibzip file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s: mibzip (%d entries)\n", filename, len(entries))
	return nil
}

func formatWalkEntries(entries []snmp.WalkEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, formatWalkEntry(entry))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatWalkEntry(entry snmp.WalkEntry) string {
	switch entry.Type {
	case gosnmp.OctetString:
		return fmt.Sprintf("%s = STRING: %q", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.Integer:
		return fmt.Sprintf("%s = INTEGER: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.Counter32:
		return fmt.Sprintf("%s = Counter32: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.Counter64:
		return fmt.Sprintf("%s = Counter64: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.Gauge32, gosnmp.Uinteger32:
		return fmt.Sprintf("%s = Gauge32: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.TimeTicks:
		return fmt.Sprintf("%s = Timeticks: (%s)", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.ObjectIdentifier:
		return fmt.Sprintf("%s = OID: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.IPAddress:
		return fmt.Sprintf("%s = IpAddress: %s", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	case gosnmp.Null:
		return fmt.Sprintf("%s = NULL: null", normalizeOID(entry.OID))
	// gosnmp gives UnknownType the same value as EndOfContents, so it cannot
	// be listed separately. The previous spelling was a bitwise OR of the two,
	// which matched only because both are 0x00.
	case gosnmp.EndOfContents,
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		return fmt.Sprintf("%s = STRING: %q", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	default:
		return fmt.Sprintf("%s = STRING: %q", normalizeOID(entry.OID), fmt.Sprint(entry.Value))
	}
}

func normalizeOID(oid string) string {
	if strings.HasPrefix(oid, ".") {
		return oid
	}
	return "." + oid
}
