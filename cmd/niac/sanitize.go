package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// defaultDeviceType is the device type abbreviation for generic/unrecognized devices.
const defaultDeviceType = "dev"

// IPMapping tracks IP address transformations.
type IPMapping struct {
	Original  string `json:"original"`
	Sanitized string `json:"sanitized"`
}

// SanitizationMapping stores all transformations.
type SanitizationMapping struct {
	IPMappings map[string]string `json:"ip_mappings"`
	Hostnames  map[string]string `json:"hostnames"`
	Statistics struct {
		FilesProcessed       int `json:"files_processed"`
		IPsTransformed       int `json:"ips_transformed"`
		HostnamesTransformed int `json:"hostnames_transformed"`
	} `json:"statistics"`
	mu sync.RWMutex // Protect concurrent access to maps
}

type sanitizeOptions struct {
	mappingFile string
	domain      string
	location    string
	contact     string
	community   string
	batch       bool
	inputDir    string
	outputDir   string
}

func addSanitizeCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(sanitizeOptions)

	sanitizeCmd := &cobra.Command{
		Use:   "sanitize <input-walk> <output-walk>",
		Short: "Sanitize SNMP walk files with NiAC-Go branding",
		Long: `Sanitize SNMP walk files by replacing real network data with consistent
NiAC-Go branded data. IP addresses are mapped deterministically so the
same input IP always produces the same output IP.

What is KEPT (not sensitive):
  • Serial numbers
  • MAC addresses
  • Hardware models
  • Interface counts/types
  • VLAN IDs

What is TRANSFORMED (deterministic):
  • IP addresses → 10.0.0.0/8 (NiAC-Go network)
  • Hostnames → niac-<location>-<type>-<number>
  • DNS domains → niac-go.com / niac-go.local
  • Contact info → netadmin@niac-go.com
  • Location strings → NiAC-Go - DC-WEST
  • Community strings → public or niac-go-ro`,
		Example: `  # Sanitize a single walk file
  niac sanitize device.walk device-sanitized.walk

  # Batch mode - sanitize all walks in a directory
  niac sanitize --batch --input-dir walks/ --output-dir sanitized/

  # Use persistent mapping file
  niac sanitize --mapping-file ip-map.json device.walk output.walk`,
		Args: func(_ *cobra.Command, args []string) error {
			if options.batch {
				// Batch mode requires --input-dir and --output-dir
				if options.inputDir == "" || options.outputDir == "" {
					return errors.New("batch mode requires --input-dir and --output-dir")
				}
				return nil
			}
			// Single file mode requires exactly 2 args
			if len(args) != minArgsForConfig {
				return errors.New("requires <input-walk> and <output-walk> arguments")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runSanitize(args, options)
		},
	}

	sanitizeCmd.Flags().StringVar(&options.mappingFile, "mapping-file", "", "JSON file to load/save IP mappings")
	sanitizeCmd.Flags().StringVar(&options.domain, "domain", "niac-go.com", "Domain for hostnames and DNS")
	sanitizeCmd.Flags().StringVar(&options.location, "location", "DC-WEST", "Default location suffix")
	sanitizeCmd.Flags().StringVar(&options.contact, "contact", "netadmin@niac-go.com", "Contact email")
	sanitizeCmd.Flags().StringVar(&options.community, "community", "public", "SNMP community string")
	sanitizeCmd.Flags().BoolVar(&options.batch, "batch", false, "Batch process multiple files")
	sanitizeCmd.Flags().StringVar(&options.inputDir, "input-dir", "", "Input directory for batch mode")
	sanitizeCmd.Flags().StringVar(&options.outputDir, "output-dir", "", "Output directory for batch mode")

	root.AddCommand(sanitizeCmd)
}

// validateSingleFilePaths validates input and output file paths for single file mode.
func validateSingleFilePaths(inputFile, outputFile string) error {
	if pathErr := validateFilePath(inputFile, false); pathErr != nil {
		return fmt.Errorf("invalid input file: %w", pathErr)
	}

	if pathErr := validateFilePath(outputFile, true); pathErr != nil {
		return fmt.Errorf("invalid output file: %w", pathErr)
	}

	return nil
}

// validateBatchPaths validates input and output directory paths for batch mode.
func validateBatchPaths(inputDir, outputDir string) error {
	if dirErr := validateDirPath(inputDir, false); dirErr != nil {
		return fmt.Errorf("invalid input directory: %w", dirErr)
	}

	if dirErr := validateDirPath(outputDir, true); dirErr != nil {
		return fmt.Errorf("invalid output directory: %w", dirErr)
	}

	return nil
}

func runSanitize(args []string, options *sanitizeOptions) error {
	batch := options.batch
	mappingFile := options.mappingFile
	domain := options.domain
	location := options.location
	contact := options.contact
	community := options.community

	// Validate input paths (Fix #67 - Input validation)
	if batch {
		if pathErr := validateBatchPaths(options.inputDir, options.outputDir); pathErr != nil {
			return pathErr
		}
	} else {
		if pathErr := validateSingleFilePaths(args[0], args[1]); pathErr != nil {
			return pathErr
		}
	}

	// Validate mapping file path if provided
	if mappingFile != "" {
		pathErr := validateFilePath(mappingFile, true)
		if pathErr != nil {
			return fmt.Errorf("invalid mapping file path: %w", pathErr)
		}
	}

	// Load or create mapping
	mapping := new(SanitizationMapping)
	mapping.IPMappings = make(map[string]string)
	mapping.Hostnames = make(map[string]string)

	if mappingFile != "" {
		loadErr := loadMapping(mappingFile, mapping)
		if loadErr != nil && !os.IsNotExist(loadErr) {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not load mapping file: %v\n", loadErr)
		}
	}

	if batch {
		inputDir := options.inputDir
		outputDir := options.outputDir
		return sanitizeBatch(inputDir, outputDir, mapping, domain, location, contact, community, mappingFile)
	}

	// Single file mode
	inputFile := args[0]
	outputFile := args[1]

	sanitizeErr := sanitizeFile(inputFile, outputFile, mapping, domain, location, contact, community)
	if sanitizeErr != nil {
		return fmt.Errorf("sanitization failed: %w", sanitizeErr)
	}

	// Save mapping if file specified
	if mappingFile != "" {
		saveErr := saveMapping(mappingFile, mapping)
		if saveErr != nil {
			return fmt.Errorf("failed to save mapping: %w", saveErr)
		}
	}

	fmt.Fprintf(os.Stdout, "✅ Sanitized %s → %s\n", inputFile, outputFile)
	fmt.Fprintf(os.Stdout, "   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Fprintf(os.Stdout, "   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

func sanitizeBatch(
	inputDir, outputDir string,
	mapping *SanitizationMapping,
	domain, location, contact, community, mappingFile string,
) error {
	// Create output directory
	mkdirErr := os.MkdirAll(outputDir, 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	// Find all .walk files
	walkFiles, err := filepath.Glob(filepath.Join(inputDir, "*.walk"))
	if err != nil {
		return fmt.Errorf("failed to list walk files: %w", err)
	}

	if len(walkFiles) == 0 {
		return fmt.Errorf("no .walk files found in %s", inputDir)
	}

	fmt.Fprintf(os.Stdout, "🔍 Found %d walk files\n", len(walkFiles))

	for i, inputFile := range walkFiles {
		basename := filepath.Base(inputFile)
		outputFile := filepath.Join(outputDir, basename)

		fmt.Fprintf(os.Stdout, "[%d/%d] Sanitizing %s...\n", i+1, len(walkFiles), basename)

		sanitizeErr := sanitizeFile(inputFile, outputFile, mapping, domain, location, contact, community)
		if sanitizeErr != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  Error: %v\n", sanitizeErr)
			continue
		}

		mapping.Statistics.FilesProcessed++
	}

	// Save mapping
	if mappingFile != "" {
		saveErr := saveMapping(mappingFile, mapping)
		if saveErr != nil {
			return fmt.Errorf("failed to save mapping: %w", saveErr)
		}
		fmt.Fprintf(os.Stdout, "\n💾 Saved mapping to %s\n", mappingFile)
	}

	fmt.Fprintf(os.Stdout, "\n✅ Batch sanitization complete!\n")
	fmt.Fprintf(os.Stdout, "   Files processed: %d\n", mapping.Statistics.FilesProcessed)
	fmt.Fprintf(os.Stdout, "   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Fprintf(os.Stdout, "   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

func sanitizeFile(
	inputFile, outputFile string,
	mapping *SanitizationMapping,
	domain, location, contact, community string,
) error {
	initialIPs, initialHostnames := getInitialMappingCounts(mapping)

	tempFile := outputFile + ".tmp"
	err := processSanitizeFile(inputFile, tempFile, mapping, domain, location, contact, community)
	if err != nil {
		return err
	}

	if renameErr := os.Rename(tempFile, outputFile); renameErr != nil {
		return fmt.Errorf("failed to rename temp file: %w", renameErr)
	}

	updateMappingStatistics(mapping, initialIPs, initialHostnames)
	return nil
}

// getInitialMappingCounts retrieves current mapping counts for tracking new transformations.
func getInitialMappingCounts(mapping *SanitizationMapping) (int, int) {
	mapping.mu.RLock()
	defer mapping.mu.RUnlock()
	return len(mapping.IPMappings), len(mapping.Hostnames)
}

// processSanitizeFile handles the file I/O and line-by-line sanitization.
func processSanitizeFile(
	inputFile, tempFile string,
	mapping *SanitizationMapping,
	domain, location, contact, community string,
) error {
	input, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer input.Close()

	output, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	writeErr := writeAndFinalize(input, output, mapping, domain, location, contact, community)
	if writeErr != nil {
		_ = output.Close() // Best effort cleanup; file will be removed
		_ = os.Remove(tempFile)
		return writeErr
	}

	return nil
}

// maxWalkLineBytes bounds a single scanned walk line. Hex-string values can be
// large, so this is well above bufio's 64KiB default while still capping memory.
const maxWalkLineBytes = 8 << 20 // 8 MiB

// writeAndFinalize processes all lines and finalizes the output file. It is a
// two-pass operation: pass one collects identity strings (hostname, contact)
// that vendor/entity OIDs echo outside the system group; pass two applies the
// per-line rules and scrubs those echoes globally.
func writeAndFinalize(
	input *os.File,
	output *os.File,
	mapping *SanitizationMapping,
	domain, location, contact, community string,
) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxWalkLineBytes)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	subs := collectIdentitySubs(lines, mapping, contact)

	writer := bufio.NewWriter(output)
	for _, line := range lines {
		var out string
		if isIdentityScalar(line) {
			// The scalar rules already replace this line's value; scrubbing it
			// again would double-sanitize the just-written hostname.
			out = sanitizeLine(line, mapping, domain, location, contact, community)
		} else {
			// Scrub echoed identity before the per-line DNS rewrite so a full
			// FQDN still matches, then apply IP/DNS rules.
			out = sanitizeLine(applyIdentitySubs(line, subs), mapping, domain, location, contact, community)
		}
		if _, err := fmt.Fprintln(writer, out); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	if err := output.Sync(); err != nil {
		return fmt.Errorf("failed to sync output: %w", err)
	}

	if err := output.Close(); err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	return nil
}

// updateMappingStatistics updates the transformation statistics.
func updateMappingStatistics(mapping *SanitizationMapping, initialIPs, initialHostnames int) {
	mapping.mu.Lock()
	defer mapping.mu.Unlock()
	mapping.Statistics.IPsTransformed += len(mapping.IPMappings) - initialIPs
	mapping.Statistics.HostnamesTransformed += len(mapping.Hostnames) - initialHostnames
}

// Precompiled patterns for sanitizeLine. Compiling once (rather than per line)
// matters: walks run to hundreds of thousands of lines each.
var (
	stringValueRe   = regexp.MustCompile(`= STRING:.*`)
	stringCaptureRe = regexp.MustCompile(`= STRING: (.+)`)
	ipValueRe       = regexp.MustCompile(`IpAddress: (\d+\.\d+\.\d+\.\d+)`)
	dnsLocalRe      = regexp.MustCompile(`\.local\b`)
	dnsTLDRe        = regexp.MustCompile(`\.(com|net|org)\b`)

	// ipIndexedOIDRe matches the standard IPv4 table columns whose row index ends
	// in a dotted-quad IP address. Only OIDs matching this have their trailing
	// four arcs rewritten as an IP; applying that to an arbitrary OID whose last
	// arcs merely look like octets corrupts the MIB structure.
	//   .4.20.1 ipAddrTable      · .4.21.1 ipRouteTable
	//   .4.22.1 ipNetToMediaTable · .3.1.1 atTable (legacy)
	ipIndexedOIDRe = regexp.MustCompile(`^\.1\.3\.6\.1\.2\.1\.(?:4\.2[012]\.1|3\.1\.1)\.`)
)

// sysGroupPrefix is the numeric OID root of the SNMPv2-MIB system group.
const sysGroupPrefix = ".1.3.6.1.2.1.1."

// ipv4OctetCount is the number of trailing OID arcs that form a dotted-quad index.
const ipv4OctetCount = 4

func sanitizeLine(line string, mapping *SanitizationMapping, domain, location, contact, community string) string {
	// 1-3. System-group identity scalars. Match both numeric walks
	// (.1.3.6.1.2.1.1.<n>.0, i.e. snmpwalk -On) and symbolic walks (MIB name).
	isContact := isSystemScalar(line, "4", "sysContact")
	switch {
	case isContact:
		line = stringValueRe.ReplaceAllString(line, "= STRING: "+contact)
	case isSystemScalar(line, "6", "sysLocation"):
		line = stringValueRe.ReplaceAllString(line, "= STRING: NiAC-Go - "+location+" - Network Operations")
	case isSystemScalar(line, "5", "sysName"):
		// Use the unquoted value so the scalar and the global echo scrub
		// (collectIdentitySubs) sanitize the identical hostname string.
		if v, ok := scalarStringValue(line); ok {
			line = stringValueRe.ReplaceAllString(line, "= STRING: "+sanitizeHostname(v, mapping))
		}
	}

	// 4. SNMP community strings (symbolic MIB objects only; numeric community
	// columns are enterprise-specific and not represented by a fixed OID).
	if strings.Contains(line, "snmpCommunity") || strings.Contains(line, "community") {
		line = stringValueRe.ReplaceAllString(line, "= STRING: "+community)
	}

	// 5. IP addresses in IpAddress values.
	line = ipValueRe.ReplaceAllStringFunc(line, func(match string) string {
		ip := ipValueRe.FindStringSubmatch(match)[1]
		if isSpecialIP(ip) {
			return match
		}
		return "IpAddress: " + sanitizeIP(ip, mapping)
	})

	// 6. IP addresses embedded as the row index of a standard IPv4 table column.
	line = rewriteOIDIndexIP(line, mapping)

	// 7. DNS domains (but skip email addresses in contact strings).
	if domain != "" && !isContact {
		line = dnsLocalRe.ReplaceAllString(line, ".niac-go.local")
		line = dnsTLDRe.ReplaceAllString(line, "."+domain)
	}

	return line
}

// isSystemScalar reports whether line is the given system-group scalar, in either
// numeric (.1.3.6.1.2.1.1.<arc>.0) or symbolic (MIB object name) walk format.
func isSystemScalar(line, arc, symbolic string) bool {
	return strings.HasPrefix(line, sysGroupPrefix+arc+".0 ") || strings.Contains(line, symbolic)
}

// rewriteOIDIndexIP rewrites the trailing dotted-quad IP index of a standard
// IPv4-indexed table column and leaves every other OID untouched, so structural
// arcs that merely look like octets are never disturbed. Only the OID field is
// considered; the value field is handled by the IpAddress pass.
func rewriteOIDIndexIP(line string, mapping *SanitizationMapping) string {
	sep := strings.Index(line, " = ")
	if sep < 0 {
		return line
	}
	oid := line[:sep]
	if !hasIPIndexedPrefix(oid) {
		return line
	}

	arcs := strings.Split(oid, ".")
	if len(arcs) < ipv4OctetCount {
		return line
	}
	index := arcs[len(arcs)-ipv4OctetCount:]
	for _, arc := range index {
		if !looksLikeIPOctet(arc) {
			return line
		}
	}

	ip := strings.Join(index, ".")
	if isSpecialIP(ip) {
		return line
	}

	copy(arcs[len(arcs)-ipv4OctetCount:], strings.Split(sanitizeIP(ip, mapping), "."))
	return strings.Join(arcs, ".") + line[sep:]
}

// hasIPIndexedPrefix reports whether oid sits under a known IPv4-indexed table column.
func hasIPIndexedPrefix(oid string) bool {
	return ipIndexedOIDRe.MatchString(oid)
}

// minIdentityScrubLen is the shortest identity value eligible for global scrubbing.
// Shorter values risk matching legitimate substrings elsewhere in the walk.
const minIdentityScrubLen = 5

// identitySub is a literal find/replace for an echoed identity string.
type identitySub struct {
	from string
	to   string
}

// isIdentityScalar reports whether line is one of the system-group identity
// scalars whose value the per-line rules already replace.
func isIdentityScalar(line string) bool {
	return isSystemScalar(line, "4", "sysContact") ||
		isSystemScalar(line, "5", "sysName") ||
		isSystemScalar(line, "6", "sysLocation")
}

// collectIdentitySubs scans a walk for the sysName and sysContact scalar values
// and returns literal substitutions (original -> sanitized) for the distinctive
// ones. Real devices echo these strings in vendor OIDs, entPhysicalName, and
// CDP/LLDP neighbor tables, which the per-line scalar rules never reach. The
// substitutions are ordered longest-first so overlapping values replace fully.
func collectIdentitySubs(lines []string, mapping *SanitizationMapping, contact string) []identitySub {
	seen := make(map[string]struct{})
	var subs []identitySub

	add := func(orig, repl string) {
		if !isDistinctiveIdentifier(orig) || orig == repl {
			return
		}
		if _, dup := seen[orig]; dup {
			return
		}
		seen[orig] = struct{}{}
		subs = append(subs, identitySub{from: orig, to: repl})
	}

	for _, line := range lines {
		switch {
		case isSystemScalar(line, "5", "sysName"):
			if v, ok := scalarStringValue(line); ok {
				add(v, sanitizeHostname(v, mapping))
			}
		case isSystemScalar(line, "4", "sysContact"):
			if v, ok := scalarStringValue(line); ok {
				add(v, contact)
			}
		}
	}

	sort.SliceStable(subs, func(i, j int) bool { return len(subs[i].from) > len(subs[j].from) })
	return subs
}

// applyIdentitySubs replaces every occurrence of each collected identity string.
func applyIdentitySubs(line string, subs []identitySub) string {
	for _, sub := range subs {
		if strings.Contains(line, sub.from) {
			line = strings.ReplaceAll(line, sub.from, sub.to)
		}
	}
	return line
}

// scalarStringValue extracts and unquotes the STRING value of a walk line.
func scalarStringValue(line string) (string, bool) {
	matches := stringCaptureRe.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return "", false
	}
	value := strings.TrimSpace(matches[1])
	value = strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	return value, value != ""
}

// isDistinctiveIdentifier reports whether s is specific enough to blanket-replace:
// long enough and containing a character typical of a hostname or contact
// (digit, dot, hyphen, underscore, or @), which excludes plain dictionary words.
func isDistinctiveIdentifier(s string) bool {
	if len(s) < minIdentityScrubLen {
		return false
	}
	return strings.ContainsAny(s, "0123456789.-_@")
}

func sanitizeIP(ip string, mapping *SanitizationMapping) string {
	// Fix #60: Check if already mapped with read lock
	mapping.mu.RLock()
	if sanitized, exists := mapping.IPMappings[ip]; exists {
		mapping.mu.RUnlock()
		return sanitized
	}
	mapping.mu.RUnlock()

	// Deterministic mapping using hash
	hash := sha256.Sum256([]byte(ip))
	hashInt := binary.BigEndian.Uint32(hash[:4])

	// Map to 10.0.0.0/8 network
	// Spread across different /16s based on original network
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip
	}

	ipBytes := parsedIP.To4()
	if ipBytes == nil {
		return ip // IPv6 not supported yet
	}

	// Determine subnet based on original first octet
	var subnet byte
	switch {
	case ipBytes[0] == privateIPClassA:
		subnet = 0 // 10.0.0.0/16 - Data Center West
	case ipBytes[0] == privateIPClassB:
		subnet = 1 // 10.1.0.0/16 - Data Center East
	case ipBytes[0] == privateIPClassC:
		subnet = 2 // 10.2.0.0/16 - Corporate Campus
	case ipBytes[0] == 63 || ipBytes[0] < privateIPClassA:
		subnet = maxPercentage // 10.100.0.0/16 - Management
	default:
		subnet = 3 // 10.3.0.0/16 - Remote Offices
	}

	// Use hash for host portion
	octet3 := safeconv.ByteFromUint32(hashInt >> bitShiftOctet)
	octet4 := safeconv.ByteFromUint32(hashInt)

	sanitized := fmt.Sprintf("10.%d.%d.%d", subnet, octet3, octet4)

	// Fix #60: Store mapping with write lock
	mapping.mu.Lock()
	// Double-check in case another goroutine added it while we waited
	if existing, exists := mapping.IPMappings[ip]; exists {
		mapping.mu.Unlock()
		return existing
	}
	mapping.IPMappings[ip] = sanitized
	mapping.mu.Unlock()

	return sanitized
}

func sanitizeHostname(hostname string, mapping *SanitizationMapping) string {
	// Fix #60: Check if already mapped with read lock
	mapping.mu.RLock()
	if sanitized, exists := mapping.Hostnames[hostname]; exists {
		mapping.mu.RUnlock()
		return sanitized
	}
	mapping.mu.RUnlock()

	// Determine device type from hostname patterns
	var deviceType string
	lower := strings.ToLower(hostname)
	switch {
	case strings.Contains(lower, "sw") || strings.Contains(lower, "switch"):
		deviceType = "sw"
	case strings.Contains(lower, "rtr") || strings.Contains(lower, "router"):
		deviceType = "rtr"
	case strings.Contains(lower, "ap") || strings.Contains(lower, "access"):
		deviceType = "ap"
	case strings.Contains(lower, "srv") || strings.Contains(lower, "server"):
		deviceType = "srv"
	case strings.Contains(lower, "fw") || strings.Contains(lower, "firewall"):
		deviceType = "fw"
	default:
		deviceType = defaultDeviceType
	}

	// Generate deterministic number from hash
	hash := sha256.Sum256([]byte(hostname))
	num := binary.BigEndian.Uint16(hash[:2]) % maxPercentage

	sanitized := fmt.Sprintf("niac-core-%s-%02d", deviceType, num)

	// Fix #60: Store mapping with write lock
	mapping.mu.Lock()
	// Double-check in case another goroutine added it while we waited
	if existing, exists := mapping.Hostnames[hostname]; exists {
		mapping.mu.Unlock()
		return existing
	}
	mapping.Hostnames[hostname] = sanitized
	mapping.mu.Unlock()

	return sanitized
}

func isSpecialIP(ip string) bool {
	// Don't transform special IPs
	specials := []string{
		"0.0.0.0", "255.255.255.255",
		"127.0.0.1", "127.0.0.0",
		"224.0.0.", "239.255.255.250", // Multicast
	}

	for _, special := range specials {
		if strings.HasPrefix(ip, special) || ip == special {
			return true
		}
	}

	return false
}

func looksLikeIPOctet(s string) bool {
	// Must be 1-3 digits and <= 255
	if len(s) < 1 || len(s) > 3 {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	// Parse and check range (digits already validated above)
	val, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return val >= 0 && val <= 255
}

func loadMapping(filename string, mapping *SanitizationMapping) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	unmarshalErr := json.Unmarshal(data, mapping)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal mapping: %w", unmarshalErr)
	}
	return nil
}

func saveMapping(filename string, mapping *SanitizationMapping) error {
	// Use lock when marshaling to avoid concurrent modifications
	mapping.mu.RLock()
	data, err := json.MarshalIndent(mapping, "", "  ")
	mapping.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Fix #68: Atomic write for mapping file
	tempFile := filename + ".tmp"
	writeErr := os.WriteFile(tempFile, data, 0o600)
	if writeErr != nil {
		return fmt.Errorf("failed to write temp mapping file: %w", writeErr)
	}

	renameErr := os.Rename(tempFile, filename)
	if renameErr != nil {
		return fmt.Errorf("failed to rename mapping file: %w", renameErr)
	}
	return nil
}

// validateFilePath validates file paths to prevent path traversal attacks
// Fix #67: Input validation.
func validateFilePath(path string, allowCreate bool) error {
	if path == "" {
		return errors.New("empty path")
	}

	cleanPath := filepath.Clean(path)

	if err := checkPathTraversal(cleanPath, path); err != nil {
		return err
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err = validatePathWithinAllowedDirs(absPath, path); err != nil {
		return err
	}

	if !allowCreate {
		return validateExistingFile(absPath, path)
	}

	return nil
}

// checkPathTraversal detects path traversal attempts in the cleaned path.
func checkPathTraversal(cleanPath, originalPath string) error {
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", originalPath)
	}
	return nil
}

// validatePathWithinAllowedDirs checks if path is within CWD, temp, or home directories.
func validatePathWithinAllowedDirs(absPath, originalPath string) error {
	validPrefixes := buildAllowedPrefixes()

	for _, prefix := range validPrefixes {
		if isPathUnderPrefix(absPath, prefix) {
			return nil
		}
	}

	return fmt.Errorf("path outside allowed directories: %s", originalPath)
}

// buildAllowedPrefixes constructs the list of allowed directory prefixes.
func buildAllowedPrefixes() []string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	prefixes := []string{cwd, os.TempDir()}

	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		prefixes = append(prefixes, homeDir)
	}

	return prefixes
}

// isPathUnderPrefix checks if absPath is under the given prefix directory.
func isPathUnderPrefix(absPath, prefix string) bool {
	absPrefix, err := filepath.Abs(prefix)
	if err != nil {
		return false
	}

	pathWithSep := absPath + string(filepath.Separator)
	prefixWithSep := absPrefix + string(filepath.Separator)

	return strings.HasPrefix(pathWithSep, prefixWithSep)
}

// validateExistingFile checks that a file exists and is a regular file (not symlink).
func validateExistingFile(absPath, originalPath string) error {
	info, err := os.Lstat(absPath)
	if err != nil {
		return handleStatError(err, originalPath)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks not allowed: %s", originalPath)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", originalPath)
	}

	return nil
}

// handleStatError converts stat errors to user-friendly messages.
func handleStatError(err error, path string) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	return fmt.Errorf("cannot access file: %w", err)
}

// validateDirPath validates directory paths
// Fix #67: Input validation.
func validateDirPath(path string, allowCreate bool) error {
	if path == "" {
		return errors.New("empty path")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed directories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	validPrefixes := []string{cwd, os.TempDir()}
	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		validPrefixes = append(validPrefixes, homeDir)
	}

	isValid := false
	for _, prefix := range validPrefixes {
		absPrefix, absErr := filepath.Abs(prefix)
		if absErr != nil {
			continue
		}
		if strings.HasPrefix(absPath+string(filepath.Separator), absPrefix+string(filepath.Separator)) {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("path outside allowed directories: %s", path)
	}

	// For input dirs, ensure they exist
	if !allowCreate {
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return fmt.Errorf("directory does not exist: %s", path)
			}
			return fmt.Errorf("cannot access directory: %w", statErr)
		}

		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", path)
		}
	}

	return nil
}
