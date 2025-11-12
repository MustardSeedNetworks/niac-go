package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// IPMapping tracks IP address transformations
type IPMapping struct {
	Original  string `json:"original"`
	Sanitized string `json:"sanitized"`
}

// SanitizationMapping stores all transformations
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

var sanitizeCmd = &cobra.Command{
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
	Args: func(cmd *cobra.Command, args []string) error {
		batch, _ := cmd.Flags().GetBool("batch")
		if batch {
			// Batch mode requires --input-dir and --output-dir
			inputDir, _ := cmd.Flags().GetString("input-dir")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			if inputDir == "" || outputDir == "" {
				return fmt.Errorf("batch mode requires --input-dir and --output-dir")
			}
			return nil
		}
		// Single file mode requires exactly 2 args
		if len(args) != 2 {
			return fmt.Errorf("requires <input-walk> and <output-walk> arguments")
		}
		return nil
	},
	RunE: runSanitize,
}

func init() {
	rootCmd.AddCommand(sanitizeCmd)

	sanitizeCmd.Flags().String("mapping-file", "", "JSON file to load/save IP mappings")
	sanitizeCmd.Flags().String("domain", "niac-go.com", "Domain for hostnames and DNS")
	sanitizeCmd.Flags().String("location", "DC-WEST", "Default location suffix")
	sanitizeCmd.Flags().String("contact", "netadmin@niac-go.com", "Contact email")
	sanitizeCmd.Flags().String("community", "public", "SNMP community string")
	sanitizeCmd.Flags().Bool("batch", false, "Batch process multiple files")
	sanitizeCmd.Flags().String("input-dir", "", "Input directory for batch mode")
	sanitizeCmd.Flags().String("output-dir", "", "Output directory for batch mode")
}

func runSanitize(cmd *cobra.Command, args []string) error {
	batch, _ := cmd.Flags().GetBool("batch")
	mappingFile, _ := cmd.Flags().GetString("mapping-file")
	domain, _ := cmd.Flags().GetString("domain")
	location, _ := cmd.Flags().GetString("location")
	contact, _ := cmd.Flags().GetString("contact")
	community, _ := cmd.Flags().GetString("community")

	// Validate input paths (Fix #67 - Input validation)
	if !batch {
		if err := validateFilePath(args[0], false); err != nil {
			return fmt.Errorf("invalid input file: %w", err)
		}
		if err := validateFilePath(args[1], true); err != nil {
			return fmt.Errorf("invalid output file: %w", err)
		}
	} else {
		inputDir, _ := cmd.Flags().GetString("input-dir")
		outputDir, _ := cmd.Flags().GetString("output-dir")
		if err := validateDirPath(inputDir, false); err != nil {
			return fmt.Errorf("invalid input directory: %w", err)
		}
		if err := validateDirPath(outputDir, true); err != nil {
			return fmt.Errorf("invalid output directory: %w", err)
		}
	}

	// Validate mapping file path if provided
	if mappingFile != "" {
		if err := validateFilePath(mappingFile, true); err != nil {
			return fmt.Errorf("invalid mapping file path: %w", err)
		}
	}

	// Load or create mapping
	mapping := &SanitizationMapping{
		IPMappings: make(map[string]string),
		Hostnames:  make(map[string]string),
	}

	if mappingFile != "" {
		if err := loadMapping(mappingFile, mapping); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not load mapping file: %v\n", err)
		}
	}

	if batch {
		inputDir, _ := cmd.Flags().GetString("input-dir")
		outputDir, _ := cmd.Flags().GetString("output-dir")
		return sanitizeBatch(inputDir, outputDir, mapping, domain, location, contact, community, mappingFile)
	}

	// Single file mode
	inputFile := args[0]
	outputFile := args[1]

	if err := sanitizeFile(inputFile, outputFile, mapping, domain, location, contact, community); err != nil {
		return fmt.Errorf("sanitization failed: %w", err)
	}

	// Save mapping if file specified
	if mappingFile != "" {
		if err := saveMapping(mappingFile, mapping); err != nil {
			return fmt.Errorf("failed to save mapping: %w", err)
		}
	}

	fmt.Printf("✅ Sanitized %s → %s\n", inputFile, outputFile)
	fmt.Printf("   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Printf("   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

func sanitizeBatch(inputDir, outputDir string, mapping *SanitizationMapping, domain, location, contact, community, mappingFile string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all .walk files
	walkFiles, err := filepath.Glob(filepath.Join(inputDir, "*.walk"))
	if err != nil {
		return fmt.Errorf("failed to list walk files: %w", err)
	}

	if len(walkFiles) == 0 {
		return fmt.Errorf("no .walk files found in %s", inputDir)
	}

	fmt.Printf("🔍 Found %d walk files\n", len(walkFiles))

	for i, inputFile := range walkFiles {
		basename := filepath.Base(inputFile)
		outputFile := filepath.Join(outputDir, basename)

		fmt.Printf("[%d/%d] Sanitizing %s...\n", i+1, len(walkFiles), basename)

		if err := sanitizeFile(inputFile, outputFile, mapping, domain, location, contact, community); err != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  Error: %v\n", err)
			continue
		}

		mapping.Statistics.FilesProcessed++
	}

	// Save mapping
	if mappingFile != "" {
		if err := saveMapping(mappingFile, mapping); err != nil {
			return fmt.Errorf("failed to save mapping: %w", err)
		}
		fmt.Printf("\n💾 Saved mapping to %s\n", mappingFile)
	}

	fmt.Printf("\n✅ Batch sanitization complete!\n")
	fmt.Printf("   Files processed: %d\n", mapping.Statistics.FilesProcessed)
	fmt.Printf("   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Printf("   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

func sanitizeFile(inputFile, outputFile string, mapping *SanitizationMapping, domain, location, contact, community string) error {
	// Fix #59: Ensure proper file handle cleanup with explicit error handling
	input, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		if cerr := input.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close input file: %w", cerr)
		}
	}()

	// Fix #68: Atomic write pattern to avoid TOCTOU
	// Write to temporary file first, then atomic rename
	tempFile := outputFile + ".tmp"
	output, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if cerr := output.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", cerr)
		}
		// Clean up temp file if operation failed
		if err != nil {
			os.Remove(tempFile)
		}
	}()

	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)

	// Track transformations for this file
	mapping.mu.RLock()
	initialIPs := len(mapping.IPMappings)
	initialHostnames := len(mapping.Hostnames)
	mapping.mu.RUnlock()

	for scanner.Scan() {
		line := scanner.Text()
		sanitized := sanitizeLine(line, mapping, domain, location, contact, community)
		if _, err := fmt.Fprintln(writer, sanitized); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	// Sync to disk before closing
	if err := output.Sync(); err != nil {
		return fmt.Errorf("failed to sync output: %w", err)
	}

	// Close output file explicitly before rename
	if err := output.Close(); err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	// Fix #68: Atomic rename to prevent TOCTOU
	if err := os.Rename(tempFile, outputFile); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Update statistics with lock
	mapping.mu.Lock()
	newIPs := len(mapping.IPMappings) - initialIPs
	newHostnames := len(mapping.Hostnames) - initialHostnames
	mapping.Statistics.IPsTransformed += newIPs
	mapping.Statistics.HostnamesTransformed += newHostnames
	mapping.mu.Unlock()

	return nil
}

func sanitizeLine(line string, mapping *SanitizationMapping, domain, location, contact, community string) string {
	// 1. System contact
	if strings.Contains(line, "sysContact") {
		line = regexp.MustCompile(`= STRING:.*`).ReplaceAllString(line, fmt.Sprintf("= STRING: %s", contact))
	}

	// 2. System location
	if strings.Contains(line, "sysLocation") {
		line = regexp.MustCompile(`= STRING:.*`).ReplaceAllString(line, fmt.Sprintf("= STRING: NiAC-Go - %s - Network Operations", location))
	}

	// 3. System name (hostname)
	if strings.Contains(line, "sysName") {
		// Extract original hostname
		re := regexp.MustCompile(`= STRING: (.+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			original := strings.TrimSpace(matches[1])
			sanitized := sanitizeHostname(original, mapping)
			line = re.ReplaceAllString(line, fmt.Sprintf("= STRING: %s", sanitized))
		}
	}

	// 4. SNMP community strings
	if strings.Contains(line, "snmpCommunity") || strings.Contains(line, "community") {
		line = regexp.MustCompile(`= STRING:.*`).ReplaceAllString(line, fmt.Sprintf("= STRING: %s", community))
	}

	// 5. IP addresses in IpAddress values
	ipValueRe := regexp.MustCompile(`IpAddress: (\d+\.\d+\.\d+\.\d+)`)
	line = ipValueRe.ReplaceAllStringFunc(line, func(match string) string {
		ip := ipValueRe.FindStringSubmatch(match)[1]
		// Skip special IPs
		if isSpecialIP(ip) {
			return match
		}
		sanitized := sanitizeIP(ip, mapping)
		return fmt.Sprintf("IpAddress: %s", sanitized)
	})

	// 6. IP addresses in OIDs (e.g., .1.3.6.1.2.1.3.1.1.3.2.1.10.250.0.45)
	// This is trickier - need to identify IP octets in OID
	oidIPRe := regexp.MustCompile(`\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})(?:\s|=|$)`)
	line = oidIPRe.ReplaceAllStringFunc(line, func(match string) string {
		parts := oidIPRe.FindStringSubmatch(match)
		if len(parts) < 5 {
			return match
		}

		// Check if these look like IP octets
		o1, o2, o3, o4 := parts[1], parts[2], parts[3], parts[4]
		if !looksLikeIPOctet(o1) || !looksLikeIPOctet(o2) || !looksLikeIPOctet(o3) || !looksLikeIPOctet(o4) {
			return match
		}

		ip := fmt.Sprintf("%s.%s.%s.%s", o1, o2, o3, o4)
		if isSpecialIP(ip) {
			return match
		}

		sanitized := sanitizeIP(ip, mapping)
		octets := strings.Split(sanitized, ".")
		suffix := match[len(match)-1:] // Preserve trailing character
		return fmt.Sprintf(".%s.%s.%s.%s%s", octets[0], octets[1], octets[2], octets[3], suffix)
	})

	// 7. DNS domains (but skip email addresses in contact strings)
	if domain != "" && !strings.Contains(line, "sysContact") {
		// Replace common domain patterns
		line = regexp.MustCompile(`\.local\b`).ReplaceAllString(line, ".niac-go.local")
		line = regexp.MustCompile(`\.(com|net|org)\b`).ReplaceAllString(line, "."+domain)
	}

	return line
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
	case ipBytes[0] == 10:
		subnet = 0 // 10.0.0.0/16 - Data Center West
	case ipBytes[0] == 172:
		subnet = 1 // 10.1.0.0/16 - Data Center East
	case ipBytes[0] == 192:
		subnet = 2 // 10.2.0.0/16 - Corporate Campus
	case ipBytes[0] == 63 || ipBytes[0] < 10:
		subnet = 100 // 10.100.0.0/16 - Management
	default:
		subnet = 3 // 10.3.0.0/16 - Remote Offices
	}

	// Use hash for host portion
	octet3 := byte(hashInt >> 8)
	octet4 := byte(hashInt)

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
		deviceType = "dev"
	}

	// Generate deterministic number from hash
	hash := sha256.Sum256([]byte(hostname))
	num := binary.BigEndian.Uint16(hash[:2]) % 100

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

	// Parse and check range
	var val int
	fmt.Sscanf(s, "%d", &val)
	return val >= 0 && val <= 255
}

func loadMapping(filename string, mapping *SanitizationMapping) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, mapping)
}

func saveMapping(filename string, mapping *SanitizationMapping) error {
	// Use lock when marshaling to avoid concurrent modifications
	mapping.mu.RLock()
	data, err := json.MarshalIndent(mapping, "", "  ")
	mapping.mu.RUnlock()

	if err != nil {
		return err
	}

	// Fix #68: Atomic write for mapping file
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tempFile, filename)
}

// validateFilePath validates file paths to prevent path traversal attacks
// Fix #67: Input validation
func validateFilePath(path string, allowCreate bool) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within current working directory or subdirectories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Allow paths within CWD, /tmp, or user's home directory
	validPrefixes := []string{cwd, os.TempDir()}
	if homeDir, err := os.UserHomeDir(); err == nil {
		validPrefixes = append(validPrefixes, homeDir)
	}

	isValid := false
	for _, prefix := range validPrefixes {
		absPrefix, err := filepath.Abs(prefix)
		if err != nil {
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

	// For input files, ensure they exist and are regular files
	if !allowCreate {
		info, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", path)
			}
			return fmt.Errorf("cannot access file: %w", err)
		}

		// Reject symlinks for security
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks not allowed: %s", path)
		}

		// Ensure it's a regular file
		if !info.Mode().IsRegular() {
			return fmt.Errorf("not a regular file: %s", path)
		}
	}

	return nil
}

// validateDirPath validates directory paths
// Fix #67: Input validation
func validateDirPath(path string, allowCreate bool) error {
	if path == "" {
		return fmt.Errorf("empty path")
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
	if homeDir, err := os.UserHomeDir(); err == nil {
		validPrefixes = append(validPrefixes, homeDir)
	}

	isValid := false
	for _, prefix := range validPrefixes {
		absPrefix, err := filepath.Abs(prefix)
		if err != nil {
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
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("directory does not exist: %s", path)
			}
			return fmt.Errorf("cannot access directory: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", path)
		}
	}

	return nil
}
