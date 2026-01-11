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

// ValidationIssue represents a single issue found in a walk file
type ValidationIssue struct {
	Line       int    `json:"line"`
	Severity   string `json:"severity"` // "error", "warning", "info"
	Message    string `json:"message"`
	Original   string `json:"original"`
	Suggestion string `json:"suggestion,omitempty"`
	AutoFix    bool   `json:"auto_fix"` // Whether this can be auto-fixed
}

// ValidationResult contains the results of validating a walk file
type ValidationResult struct {
	Filename    string            `json:"filename"`
	Valid       bool              `json:"valid"`
	TotalLines  int               `json:"total_lines"`
	ValidLines  int               `json:"valid_lines"`
	Issues      []ValidationIssue `json:"issues"`
	FixedCount  int               `json:"fixed_count,omitempty"`
	FixedLines  []int             `json:"fixed_lines,omitempty"`
}

// ValidateWalkFile validates a walk file and returns detailed results
func ValidateWalkFile(filename string) (*ValidationResult, error) {
	// Security: validate path
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid walk file path: directory traversal not allowed")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid walk file path: %w", err)
	}

	fileInfo, err := os.Lstat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access walk file: %v", err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("walk file cannot be a symbolic link")
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("walk file path is a directory, not a file")
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open walk file: %v", err)
	}
	defer file.Close()

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
		trimmedLine := strings.TrimSpace(line)

		result.TotalLines++

		// Skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			result.ValidLines++
			continue
		}

		// Validate the line
		issues := validateWalkLine(lineNum, line)
		if len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			// Check if any issue is an error
			for _, issue := range issues {
				if issue.Severity == "error" {
					result.Valid = false
				}
			}
		} else {
			result.ValidLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading walk file: %v", err)
	}

	return result, nil
}

// validateWalkLine validates a single line and returns any issues found
func validateWalkLine(lineNum int, line string) []ValidationIssue {
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

	parts := strings.SplitN(trimmedLine, "=", 2)
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

	typeParts := strings.SplitN(rest, ":", 2)
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

// validateOID checks if the OID is valid
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
	if strings.HasPrefix(oid, ".") {
		// Check for valid numeric OID format
		oidPattern := regexp.MustCompile(`^\.(\d+\.)*\d+$`)
		if !oidPattern.MatchString(oid) {
			// Check for common issues
			if strings.Contains(oid, "..") {
				issues = append(issues, ValidationIssue{
					Line:       lineNum,
					Severity:   "error",
					Message:    "OID contains double dots '..'",
					Original:   originalLine,
					Suggestion: strings.ReplaceAll(oid, "..", "."),
					AutoFix:    true,
				})
			} else if trimmedOID, found := strings.CutSuffix(oid, "."); found {
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
	} else if strings.Contains(oid, "::") {
		// Named OID like SNMPv2-MIB::sysDescr.0
		// This is valid format
	} else {
		// Check if it looks like a malformed OID
		if !strings.Contains(oid, ".") && !strings.Contains(oid, "::") {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "warning",
				Message:  "OID format is unusual - expected numeric (.1.3.6...) or named (MIB::name) format",
				Original: originalLine,
				AutoFix:  false,
			})
		}
	}

	return issues
}

// validateType checks if the type is valid and properly formatted
func validateType(lineNum int, typeStr, _ /*valueStr*/, originalLine string) []ValidationIssue {
	var issues []ValidationIssue

	upperType := strings.ToUpper(typeStr)

	// Valid types
	validTypes := map[string]bool{
		"STRING": true, "OCTET STRING": true,
		"INTEGER": true, "INT": true,
		"GAUGE": true, "GAUGE32": true,
		"COUNTER": true, "COUNTER32": true,
		"COUNTER64": true,
		"TIMETICKS": true,
		"OID": true, "OBJECT IDENTIFIER": true,
		"IPADDRESS": true, "IP ADDRESS": true, "IPADDR": true,
		"BITS":       true,
		"HEX-STRING": true, "HEX": true,
		"OPAQUE": true,
		"NULL":   true,
		// Also allow "No Such Object", "No Such Instance", etc.
		"NO SUCH OBJECT AT THIS OID":   true,
		"NO SUCH INSTANCE EXISTS":      true,
		"NO SUCH OBJECT":               true,
		"NO SUCH INSTANCE":             true,
		"ENDOFMIBVIEW":                 true,
	}

	if !validTypes[upperType] {
		// Check for common misspellings or case issues
		if typeStr != upperType && validTypes[upperType] {
			issues = append(issues, ValidationIssue{
				Line:       lineNum,
				Severity:   "warning",
				Message:    fmt.Sprintf("Type '%s' should be uppercase '%s'", typeStr, upperType),
				Original:   originalLine,
				Suggestion: strings.Replace(originalLine, typeStr+":", upperType+":", 1),
				AutoFix:    true,
			})
		} else {
			// Check for common misspellings
			corrections := map[string]string{
				"STRNG":     "STRING",
				"STRIGN":    "STRING",
				"INTGER":    "INTEGER",
				"INTEGR":    "INTEGER",
				"GUAGE":     "GAUGE",
				"GUAGE32":   "GAUGE32",
				"CONTER":    "COUNTER",
				"CONTER32":  "COUNTER32",
				"TIMTICKS":  "TIMETICKS",
				"TIMETICK":  "TIMETICKS",
				"IPADRESS":  "IPADDRESS",
				"IPADDRES":  "IPADDRESS",
			}
			if correction, ok := corrections[upperType]; ok {
				issues = append(issues, ValidationIssue{
					Line:       lineNum,
					Severity:   "warning",
					Message:    fmt.Sprintf("Type '%s' appears to be misspelled, did you mean '%s'?", typeStr, correction),
					Original:   originalLine,
					Suggestion: strings.Replace(originalLine, typeStr+":", correction+":", 1),
					AutoFix:    true,
				})
			} else {
				issues = append(issues, ValidationIssue{
					Line:     lineNum,
					Severity: "warning",
					Message:  fmt.Sprintf("Unknown type '%s' - will be treated as STRING", typeStr),
					Original: originalLine,
					AutoFix:  false,
				})
			}
		}
	}

	return issues
}

// validateValue checks if the value is valid for the given type
func validateValue(lineNum int, typeStr, valueStr, originalLine string) []ValidationIssue {
	var issues []ValidationIssue
	upperType := strings.ToUpper(typeStr)

	switch upperType {
	case "STRING", "OCTET STRING":
		// Strings should ideally be quoted
		if valueStr != "" && !strings.HasPrefix(valueStr, "\"") {
			// Check if it looks like it should be quoted
			if !strings.HasPrefix(valueStr, "0x") && !strings.HasPrefix(valueStr, "varimib") && !strings.HasPrefix(valueStr, "fixed") {
				issues = append(issues, ValidationIssue{
					Line:       lineNum,
					Severity:   "info",
					Message:    "String value is not quoted (recommended for clarity)",
					Original:   originalLine,
					Suggestion: strings.Replace(originalLine, valueStr, fmt.Sprintf("\"%s\"", valueStr), 1),
					AutoFix:    true,
				})
			}
		}

	case "INTEGER", "INT":
		if valueStr == "" {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  "INTEGER value is empty",
				Original: originalLine,
				AutoFix:  false,
			})
		} else if _, err := strconv.ParseInt(valueStr, 10, 32); err != nil {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  fmt.Sprintf("Invalid INTEGER value '%s': %v", valueStr, err),
				Original: originalLine,
				AutoFix:  false,
			})
		}

	case "GAUGE", "GAUGE32", "COUNTER", "COUNTER32":
		if valueStr == "" {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  fmt.Sprintf("%s value is empty", upperType),
				Original: originalLine,
				AutoFix:  false,
			})
		} else if _, err := strconv.ParseUint(valueStr, 10, 32); err != nil {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  fmt.Sprintf("Invalid %s value '%s': %v", upperType, valueStr, err),
				Original: originalLine,
				AutoFix:  false,
			})
		}

	case "COUNTER64":
		if valueStr == "" {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  "COUNTER64 value is empty",
				Original: originalLine,
				AutoFix:  false,
			})
		} else if _, err := strconv.ParseUint(valueStr, 10, 64); err != nil {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  fmt.Sprintf("Invalid COUNTER64 value '%s': %v", valueStr, err),
				Original: originalLine,
				AutoFix:  false,
			})
		}

	case "TIMETICKS":
		// Valid formats: (12345) or 12345 or (12345) 0:02:03.45
		re := regexp.MustCompile(`^\(?\d+\)?`)
		if !re.MatchString(valueStr) {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "error",
				Message:  fmt.Sprintf("Invalid Timeticks format '%s' - expected (number) or number", valueStr),
				Original: originalLine,
				AutoFix:  false,
			})
		}

	case "IPADDRESS", "IP ADDRESS", "IPADDR":
		// Basic IP validation
		ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
		if valueStr != "" && !ipPattern.MatchString(valueStr) {
			issues = append(issues, ValidationIssue{
				Line:     lineNum,
				Severity: "warning",
				Message:  fmt.Sprintf("IP address '%s' may be malformed", valueStr),
				Original: originalLine,
				AutoFix:  false,
			})
		}

	case "OID", "OBJECT IDENTIFIER":
		// Should start with . or be empty
		if valueStr != "" && !strings.HasPrefix(valueStr, ".") && !strings.Contains(valueStr, "::") {
			issues = append(issues, ValidationIssue{
				Line:       lineNum,
				Severity:   "warning",
				Message:    "OID value should start with '.'",
				Original:   originalLine,
				Suggestion: strings.Replace(originalLine, ": "+valueStr, ": ."+valueStr, 1),
				AutoFix:    true,
			})
		}
	}

	return issues
}

// AutoFixWalkFile attempts to fix common issues in a walk file
// Returns the path to the fixed file and a list of fixes applied
func AutoFixWalkFile(filename string, outputPath string) (*ValidationResult, error) {
	// First validate to get list of issues
	validation, err := ValidateWalkFile(filename)
	if err != nil {
		return nil, err
	}

	// Read the file
	cleanPath := filepath.Clean(filename)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid walk file path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read walk file: %v", err)
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
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return nil, fmt.Errorf("failed to create backup: %v", err)
		}
		outputPath = absPath
	}

	// Write fixed content
	if err := os.WriteFile(outputPath, []byte(strings.Join(fixedContent, "\n")), 0644); err != nil {
		return nil, fmt.Errorf("failed to write fixed file: %v", err)
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

// ValidateAndSummarize provides a summary of walk file validation
func ValidateAndSummarize(filename string) (string, error) {
	result, err := ValidateWalkFile(filename)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Walk File Validation: %s\n", filename)
	sb.WriteString(strings.Repeat("=", 60) + "\n")
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
		sb.WriteString(strings.Repeat("-", 60) + "\n")

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

		sb.WriteString(strings.Repeat("-", 60) + "\n")
		fmt.Fprintf(&sb, "Summary: %d errors, %d warnings, %d info\n", errorCount, warningCount, infoCount)
		fmt.Fprintf(&sb, "Auto-fixable: %d issues\n", autoFixable)
	}

	return sb.String(), nil
}
