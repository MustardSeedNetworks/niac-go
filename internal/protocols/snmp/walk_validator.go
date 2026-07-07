package snmp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidationIssue represents a single issue found in a walk file.
type ValidationIssue struct {
	Line       int    `json:"line"`
	Severity   string `json:"severity"` // "error", "warning", "info"
	Message    string `json:"message"`
	Original   string `json:"original"`
	Suggestion string `json:"suggestion,omitempty"`
	AutoFix    bool   `json:"auto_fix"` // Whether this can be auto-fixed
	OID        string `json:"oid,omitempty"`
}

// ValidationResult contains the results of validating a walk file.
type ValidationResult struct {
	Filename   string            `json:"filename"`
	Valid      bool              `json:"valid"`
	TotalLines int               `json:"total_lines"`
	ValidLines int               `json:"valid_lines"`
	Issues     []ValidationIssue `json:"issues"`
	FixedCount int               `json:"fixed_count,omitempty"`
	FixedLines []int             `json:"fixed_lines,omitempty"`
}

// ValidateWalkFile validates a walk file and returns detailed results.
func ValidateWalkFile(filename string) (*ValidationResult, error) {
	absPath, err := validateAndResolvePath(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToOpenWalkFile, err)
	}
	defer func() { _ = file.Close() }()

	return scanWalkFile(filename, file)
}

// writeValidatedFile writes data with 0600 perms to a path that has already been
// validated upstream. The defensive checks here are belt-and-suspenders so static
// analysers (gosec, CodeQL) can see the path is bounded.
func writeValidatedFile(path string, data []byte) error {
	safePath := filepath.Clean(path)
	if !filepath.IsAbs(safePath) || strings.Contains(safePath, "..") {
		return fmt.Errorf("walk output path failed safety check: %s", safePath)
	}
	return os.WriteFile(safePath, data, 0o600) //nolint:gosec // G703: path validated above and upstream
}

// validateAndResolvePath validates the path for security issues and returns the absolute path.
func validateAndResolvePath(filename string) (string, error) {
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return "", ErrDirectoryTraversal
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid walk file path: %w", err)
	}

	fileInfo, err := os.Lstat(absPath)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToAccessWalkFile, err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return "", ErrWalkFileSymlink
	}

	if fileInfo.IsDir() {
		return "", ErrWalkFileIsDirectory
	}

	return absPath, nil
}

// scanWalkFile reads and validates each line in the walk file.
func scanWalkFile(filename string, file *os.File) (*ValidationResult, error) {
	result := &ValidationResult{
		Filename: filename,
		Valid:    true,
		Issues:   []ValidationIssue{},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		result.TotalLines++

		processWalkLine(lineNum, line, result)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadingWalkFile, scanErr)
	}

	return result, nil
}

// processWalkLine validates a single line and updates the result.
func processWalkLine(lineNum int, line string, result *ValidationResult) {
	trimmedLine := strings.TrimSpace(line)

	if isSkippableLine(trimmedLine) {
		result.ValidLines++
		return
	}

	issues := validateWalkLine(lineNum, line)
	if len(issues) == 0 {
		result.ValidLines++
		return
	}

	result.Issues = append(result.Issues, issues...)
	if containsError(issues) {
		result.Valid = false
	}
}

// isSkippableLine returns true if the line should be skipped (empty or comment).
func isSkippableLine(trimmedLine string) bool {
	return trimmedLine == "" || strings.HasPrefix(trimmedLine, "#")
}

// containsError checks if any issue in the slice has error severity.
func containsError(issues []ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// validateWalkLine validates a single line and returns any issues found, each
// stamped with the OID parsed from the line (via the shared extractLineOID
// parser used by ParseWalkFile). Structural issues where no OID could be
// parsed (e.g. a missing "=" separator) are left with an empty OID.
func validateWalkLine(lineNum int, line string) []ValidationIssue {
	issues := collectWalkLineIssues(lineNum, line)

	lineOID := extractLineOID(line)
	for i := range issues {
		issues[i].OID = lineOID
	}

	return issues
}

// collectWalkLineIssues performs the actual per-line validation checks.
func collectWalkLineIssues(lineNum int, line string) []ValidationIssue {
	var issues []ValidationIssue

	trimmedLine := strings.TrimSpace(line)

	// Check for leading/trailing whitespace
	if line != trimmedLine {
		issues = append(issues, ValidationIssue{
			Line:       lineNum,
			Severity:   "info",
			Message:    "Line has leading or trailing whitespace",
			Original:   line,
			Suggestion: trimmedLine,
			AutoFix:    true,
		})
	}

	// Check for = separator
	if !strings.Contains(trimmedLine, "=") {
		issues = append(issues, ValidationIssue{
			Line:     lineNum,
			Severity: "error",
			Message:  "Missing '=' separator between OID and value",
			Original: trimmedLine,
			AutoFix:  false,
		})

		return issues
	}

	parts := strings.SplitN(trimmedLine, "=", OIDPartsMinPDU)
	oid := strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	// Validate OID format
	oidIssues := validateOID(lineNum, oid, trimmedLine)
	issues = append(issues, oidIssues...)

	// Check for : separator in type/value
	if !strings.Contains(rest, ":") {
		issues = append(issues, ValidationIssue{
			Line:     lineNum,
			Severity: "error",
			Message:  "Missing ':' separator between type and value",
			Original: trimmedLine,
			AutoFix:  false,
		})

		return issues
	}

	typeParts := strings.SplitN(rest, ":", OIDPartsMinPDU)
	typeStr := strings.TrimSpace(typeParts[0])
	valueStr := strings.TrimSpace(typeParts[1])

	// Validate type
	typeIssues := validateType(lineNum, typeStr, valueStr, trimmedLine)
	issues = append(issues, typeIssues...)

	// Validate value based on type
	valueIssues := validateValue(lineNum, typeStr, valueStr, trimmedLine)
	issues = append(issues, valueIssues...)

	return issues
}

// validateOID checks if the OID is valid.
func validateOID(lineNum int, oid, originalLine string) []ValidationIssue {
	var issues []ValidationIssue

	// Check if OID starts with a dot (numeric OID) or is a named OID
	if oid == "" {
		issues = append(issues, ValidationIssue{
			Line:     lineNum,
			Severity: "error",
			Message:  "Empty OID",
			Original: originalLine,
			AutoFix:  false,
		})

		return issues
	}

	// Numeric OID validation
	switch {
	case strings.HasPrefix(oid, "."):
		// Check for valid numeric OID format
		oidPattern := regexp.MustCompile(`^\.(\d+\.)*\d+$`)
		if !oidPattern.MatchString(oid) {
			// Check for common issues
			switch {
			case strings.Contains(oid, ".."):
				issues = append(issues, ValidationIssue{
					Line:       lineNum,
					Severity:   "error",
					Message:    "OID contains double dots '..'",
					Original:   originalLine,
					Suggestion: strings.ReplaceAll(oid, "..", "."),
					AutoFix:    true,
				})
			default:
				if trimmedOID, found := strings.CutSuffix(oid, "."); found {
					issues = append(issues, ValidationIssue{
						Line:       lineNum,
						Severity:   "warning",
						Message:    "OID ends with a trailing dot",
						Original:   originalLine,
						Suggestion: trimmedOID,
						AutoFix:    true,
					})
				}
			}
		}
	case strings.Contains(oid, "::"):
		// Named OID like SNMPv2-MIB::sysDescr.0 - valid format, no action needed.
		return issues
	case !strings.Contains(oid, "."):
		// Check if it looks like a malformed OID
		issues = append(issues, ValidationIssue{
			Line:     lineNum,
			Severity: "warning",
			Message:  "OID format is unusual - expected numeric (.1.3.6...) or named (MIB::name) format",
			Original: originalLine,
			AutoFix:  false,
		})
	}

	return issues
}

// validateType checks if the type is valid and properly formatted.
func validateType(lineNum int, typeStr, _ /*valueStr*/, originalLine string) []ValidationIssue {
	upperType := strings.ToUpper(typeStr)

	if isValidSNMPType(upperType) {
		return nil
	}

	// Check for case mismatch first (original is different case but valid when uppercased)
	if issue := checkCaseMismatch(lineNum, typeStr, upperType, originalLine); issue != nil {
		return []ValidationIssue{*issue}
	}

	// Check for common misspellings
	if issue := checkMisspelling(lineNum, typeStr, upperType, originalLine); issue != nil {
		return []ValidationIssue{*issue}
	}

	// Unknown type - will be treated as STRING
	return []ValidationIssue{{
		Line:     lineNum,
		Severity: "warning",
		Message:  fmt.Sprintf("Unknown type '%s' - will be treated as STRING", typeStr),
		Original: originalLine,
		AutoFix:  false,
	}}
}

// isValidSNMPType checks if the given uppercase type string is a valid SNMP type.
func isValidSNMPType(upperType string) bool {
	validTypes := map[string]bool{
		"STRING": true, "OCTET STRING": true,
		"INTEGER": true, "INT": true,
		"GAUGE": true, "GAUGE32": true,
		"COUNTER": true, "COUNTER32": true,
		"COUNTER64": true,
		"TIMETICKS": true,
		"OID":       true, "OBJECT IDENTIFIER": true,
		"IPADDRESS": true, "IP ADDRESS": true, "IPADDR": true,
		"BITS":       true,
		"HEX-STRING": true, "HEX": true,
		"OPAQUE": true,
		"NULL":   true,
		// Also allow "No Such Object", "No Such Instance", etc.
		"NO SUCH OBJECT AT THIS OID": true,
		"NO SUCH INSTANCE EXISTS":    true,
		"NO SUCH OBJECT":             true,
		"NO SUCH INSTANCE":           true,
		"ENDOFMIBVIEW":               true,
	}
	return validTypes[upperType]
}

// checkCaseMismatch checks if the type is valid but has case issues.
func checkCaseMismatch(lineNum int, typeStr, upperType, originalLine string) *ValidationIssue {
	// If the original differs from uppercase AND uppercase is valid, it's a case issue
	if typeStr != upperType && isValidSNMPType(upperType) {
		return &ValidationIssue{
			Line:       lineNum,
			Severity:   "warning",
			Message:    fmt.Sprintf("Type '%s' should be uppercase '%s'", typeStr, upperType),
			Original:   originalLine,
			Suggestion: strings.Replace(originalLine, typeStr+":", upperType+":", 1),
			AutoFix:    true,
		}
	}
	return nil
}

// checkMisspelling checks for common type misspellings.
func checkMisspelling(lineNum int, typeStr, upperType, originalLine string) *ValidationIssue {
	corrections := map[string]string{
		"STRNG":    "STRING",
		"STRIGN":   "STRING",
		"INTGER":   "INTEGER",
		"INTEGR":   "INTEGER",
		"GUAGE32":  "GAUGE32",
		"CONTER":   "COUNTER",
		"CONTER32": "COUNTER32",
		"TIMTICKS": "TIMETICKS",
		"TIMETICK": "TIMETICKS",
		"IPADRESS": "IPADDRESS",
		"IPADDRES": "IPADDRESS",
	}
	if correction, ok := corrections[upperType]; ok {
		return &ValidationIssue{
			Line:     lineNum,
			Severity: "warning",
			Message: fmt.Sprintf(
				"Type '%s' appears to be misspelled, did you mean '%s'?",
				typeStr,
				correction,
			),
			Original:   originalLine,
			Suggestion: strings.Replace(originalLine, typeStr+":", correction+":", 1),
			AutoFix:    true,
		}
	}
	return nil
}

// validateValue checks if the value is valid for the given type.
func validateValue(lineNum int, typeStr, valueStr, originalLine string) []ValidationIssue {
	upperType := strings.ToUpper(typeStr)

	switch upperType {
	case "STRING", "OCTET STRING":
		return validateStringValue(lineNum, valueStr, originalLine)
	case "INTEGER", "INT":
		return validateIntegerValue(lineNum, valueStr, originalLine)
	case "GAUGE", "GAUGE32", "COUNTER", "COUNTER32":
		return validateUnsigned32Value(lineNum, upperType, valueStr, originalLine)
	case "COUNTER64":
		return validateCounter64Value(lineNum, valueStr, originalLine)
	case "TIMETICKS":
		return validateTimeticksValue(lineNum, valueStr, originalLine)
	case "IPADDRESS", "IP ADDRESS", "IPADDR":
		return validateIPAddressValue(lineNum, valueStr, originalLine)
	case "OID", "OBJECT IDENTIFIER":
		return validateOIDValue(lineNum, valueStr, originalLine)
	default:
		return nil
	}
}

// validateStringValue validates string/octet string values.
func validateStringValue(lineNum int, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" || strings.HasPrefix(valueStr, "\"") {
		return nil
	}

	if strings.HasPrefix(valueStr, "0x") || strings.HasPrefix(valueStr, "varimib") ||
		strings.HasPrefix(valueStr, "fixed") {
		return nil
	}

	return []ValidationIssue{{
		Line:       lineNum,
		Severity:   "info",
		Message:    "String value is not quoted (recommended for clarity)",
		Original:   originalLine,
		Suggestion: strings.Replace(originalLine, valueStr, fmt.Sprintf("\"%s\"", valueStr), 1),
		AutoFix:    true,
	}}
}

// validateIntegerValue validates INTEGER values.
func validateIntegerValue(lineNum int, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  "INTEGER value is empty",
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	if _, err := strconv.ParseInt(valueStr, 10, 32); err != nil {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  fmt.Sprintf("Invalid INTEGER value '%s': %v", valueStr, err),
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	return nil
}

// validateUnsigned32Value validates GAUGE32/COUNTER32 values.
func validateUnsigned32Value(lineNum int, typeName, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  typeName + " value is empty",
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	if _, err := strconv.ParseUint(valueStr, 10, 32); err != nil {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  fmt.Sprintf("Invalid %s value '%s': %v", typeName, valueStr, err),
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	return nil
}

// validateCounter64Value validates COUNTER64 values.
func validateCounter64Value(lineNum int, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  "COUNTER64 value is empty",
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	if _, err := strconv.ParseUint(valueStr, 10, 64); err != nil {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  fmt.Sprintf("Invalid COUNTER64 value '%s': %v", valueStr, err),
			Original: originalLine,
			AutoFix:  false,
		}}
	}

	return nil
}

// validateTimeticksValue validates TIMETICKS values.
func validateTimeticksValue(lineNum int, valueStr, originalLine string) []ValidationIssue {
	re := regexp.MustCompile(`^\(?\d+\)?`)
	if !re.MatchString(valueStr) {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "error",
			Message:  fmt.Sprintf("Invalid Timeticks format '%s' - expected (number) or number", valueStr),
			Original: originalLine,
			AutoFix:  false,
		}}
	}
	return nil
}

// validateIPAddressValue validates IP address values.
func validateIPAddressValue(lineNum int, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" {
		return nil
	}

	ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	if !ipPattern.MatchString(valueStr) {
		return []ValidationIssue{{
			Line:     lineNum,
			Severity: "warning",
			Message:  fmt.Sprintf("IP address '%s' may be malformed", valueStr),
			Original: originalLine,
			AutoFix:  false,
		}}
	}
	return nil
}

// validateOIDValue validates OID values.
func validateOIDValue(lineNum int, valueStr, originalLine string) []ValidationIssue {
	if valueStr == "" || strings.HasPrefix(valueStr, ".") || strings.Contains(valueStr, "::") {
		return nil
	}

	return []ValidationIssue{{
		Line:       lineNum,
		Severity:   "warning",
		Message:    "OID value should start with '.'",
		Original:   originalLine,
		Suggestion: strings.Replace(originalLine, ": "+valueStr, ": ."+valueStr, 1),
		AutoFix:    true,
	}}
}

// AutoFixWalkFile attempts to fix common issues in a walk file
// Returns the path to the fixed file and a list of fixes applied.
func AutoFixWalkFile(filename string, outputPath string) (*ValidationResult, error) {
	// First validate to get list of issues
	validation, err := ValidateWalkFile(filename)
	if err != nil {
		return nil, err
	}

	// Reuse the same validator that produced the absolute path during ValidateWalkFile
	// so static analysers see a single bounded source for the file read below.
	absPath, err := validateAndResolvePath(filename)
	if err != nil {
		return nil, err
	}

	// Inline barrier on the read sink so CodeQL sees the bound at this call.
	if !filepath.IsAbs(absPath) || strings.Contains(absPath, "..") {
		return nil, fmt.Errorf("walk file path failed safety check: %s", absPath)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToReadWalkFile, err)
	}

	lines := strings.Split(string(content), "\n")
	fixedLines := make(map[int]bool)

	// Build map of line -> suggested fix
	lineFixes := make(map[int]string)

	for _, issue := range validation.Issues {
		if issue.AutoFix && issue.Suggestion != "" {
			lineFixes[issue.Line] = issue.Suggestion
		}
	}

	// Apply fixes
	var fixedContent []string

	for i, line := range lines {
		lineNum := i + 1
		if fix, ok := lineFixes[lineNum]; ok {
			fixedContent = append(fixedContent, fix)
			fixedLines[lineNum] = true
		} else {
			fixedContent = append(fixedContent, line)
		}
	}

	// Determine output path
	if outputPath == "" {
		// Create backup and overwrite original
		backupPath := absPath + ".bak"
		if writeErr := writeValidatedFile(backupPath, content); writeErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToCreateBackup, writeErr)
		}

		outputPath = absPath
	}

	// Write fixed content
	if writeErr := writeValidatedFile(outputPath, []byte(strings.Join(fixedContent, "\n"))); writeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToWriteFixedFile, writeErr)
	}

	// Build list of fixed line numbers
	var fixedLineNumbers []int
	for lineNum := range fixedLines {
		fixedLineNumbers = append(fixedLineNumbers, lineNum)
	}

	validation.FixedCount = len(fixedLines)
	validation.FixedLines = fixedLineNumbers

	return validation, nil
}

// ValidateAndSummarize provides a summary of walk file validation.
func ValidateAndSummarize(filename string) (string, error) {
	result, err := ValidateWalkFile(filename)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "Walk File Validation: %s\n", filename)
	sb.WriteString(strings.Repeat("=", SecondsPerMinute) + "\n")
	fmt.Fprintf(&sb, "Total Lines: %d\n", result.TotalLines)
	fmt.Fprintf(&sb, "Valid Lines: %d\n", result.ValidLines)
	fmt.Fprintf(&sb, "Issues Found: %d\n", len(result.Issues))

	if result.Valid {
		sb.WriteString("Status: VALID ✓\n")
	} else {
		sb.WriteString("Status: INVALID ✗\n")
	}

	if len(result.Issues) > 0 {
		sb.WriteString("\nIssues:\n")
		sb.WriteString(strings.Repeat("-", SecondsPerMinute) + "\n")

		errorCount := 0
		warningCount := 0
		infoCount := 0
		autoFixable := 0

		for _, issue := range result.Issues {
			switch issue.Severity {
			case "error":
				errorCount++
			case "warning":
				warningCount++
			case "info":
				infoCount++
			}

			if issue.AutoFix {
				autoFixable++
			}

			fmt.Fprintf(&sb, "Line %d [%s]: %s\n", issue.Line, strings.ToUpper(issue.Severity), issue.Message)

			if issue.Suggestion != "" {
				fmt.Fprintf(&sb, "  Suggestion: %s\n", issue.Suggestion)
			}
		}

		sb.WriteString(strings.Repeat("-", SecondsPerMinute) + "\n")
		fmt.Fprintf(&sb, "Summary: %d errors, %d warnings, %d info\n", errorCount, warningCount, infoCount)
		fmt.Fprintf(&sb, "Auto-fixable: %d issues\n", autoFixable)
	}

	return sb.String(), nil
}
