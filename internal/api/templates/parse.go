package templates

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Scan walks baseDir for YAML template files and returns the parsed
// metadata for each. Entries that error or that are not .yaml/.yml are
// skipped; the walk continues.
func Scan(baseDir string) ([]Template, error) {
	var found []Template

	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		// Skip entries with errors
		if walkErr != nil || d.IsDir() {
			return nil //nolint:nilerr // WalkDir: skip errors, continue walking
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		found = append(found, parseTemplateFile(path))
		return nil
	})

	return found, err
}

// parseTemplateFile extracts template metadata from a file.
//
// Templates can opt-in to a small front-matter block at the very top of
// the YAML for richer UI metadata. The parser walks consecutive comment
// lines and recognises:
//
//	# Display: Enterprise Router
//	# Description: A full enterprise router persona with LLDP, CDP, SNMP, ...
//
// When present these win over the legacy "first non-empty comment line"
// heuristic. The keys are case-insensitive and the rest of the file is
// untouched.
func parseTemplateFile(path string) Template {
	// Extract name from filename without extension
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	meta := extractFrontMatter(path)

	displayName := meta["display"]

	description := meta["description"]
	if description == "" {
		// Fall back to the legacy first-comment heuristic for templates
		// that haven't been migrated yet.
		description = extractDescription(path)
	}
	if description == "" {
		base := displayName
		if base == "" {
			base = name
		}
		description = base + " configuration template"
	}

	// Count devices in the YAML file
	deviceCount := CountDevicesInFile(path)

	// Determine type: front-matter wins, otherwise infer from name+path.
	templateType := strings.ToLower(strings.TrimSpace(meta["type"]))
	if templateType == "" {
		templateType = determineTemplateType(path, name)
	}

	// Tags: front-matter "tags:" comma-separated wins, otherwise infer.
	tags := splitTags(meta["tags"])
	if len(tags) == 0 {
		tags = generateTags(path)
	}

	// Vendor is optional. We lowercase so the UI's group-by key is
	// stable regardless of how the YAML author cased it.
	vendor := strings.ToLower(strings.TrimSpace(meta["vendor"]))

	return Template{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		DeviceCount: deviceCount,
		Type:        templateType,
		Vendor:      vendor,
		Tags:        tags,
	}
}

// splitTags parses a comma-separated tags string into a slice. Empty tags
// are skipped and whitespace is trimmed.
func splitTags(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// extractFrontMatter pulls "# Key: Value" lines from the top of a YAML
// file into a map. Stops at the first non-comment, non-blank line. Keys
// are lowercased; values are trimmed.
func extractFrontMatter(path string) map[string]string {
	// filepath.Clean keeps gosec G304 quiet — callers walk a
	// server-owned templates directory, but cleaning is cheap and
	// satisfies the linter without a //nolint exception.
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil
	}
	meta := make(map[string]string)
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		comment, isComment := strings.CutPrefix(trimmed, "#")
		if !isComment {
			break
		}
		comment = strings.TrimSpace(comment)
		key, value, ok := strings.Cut(comment, ":")
		if !ok {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(key))
		v := strings.TrimSpace(value)
		if k != "" && v != "" {
			meta[k] = v
		}
	}
	return meta
}

// CountDevicesInFile counts the number of devices defined in a YAML
// config. It is a shared config-file utility: both the template parser
// and the user-config parser in the api package use it.
func CountDevicesInFile(path string) int {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return 1 // Default to 1 if can't read
	}

	// Simple heuristic: count lines that start with "- name:" (device entries)
	count := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- name:") {
			count++
		}
	}

	if count == 0 {
		return 1 // At least 1 device assumed
	}
	return count
}

// determineTemplateType determines the template type from its path and name.
func determineTemplateType(path, name string) string {
	pathLower := strings.ToLower(path)
	nameLower := strings.ToLower(name)

	// Check filename first for specific types
	switch {
	case strings.Contains(nameLower, "router"):
		return typeRouter
	case strings.Contains(nameLower, "switch"):
		return typeSwitch
	case strings.Contains(nameLower, "ap") || strings.Contains(nameLower, "wireless") ||
		strings.Contains(nameLower, "access-point"):
		return typeAccessPoint
	case strings.Contains(nameLower, "firewall") || strings.Contains(nameLower, "asa") ||
		strings.Contains(nameLower, "srx") || strings.Contains(nameLower, "ftd"):
		return typeFirewall
	case strings.Contains(nameLower, "server"):
		return typeServer
	case strings.Contains(nameLower, "complete") || strings.Contains(nameLower, "full"):
		return typeComplete
	}

	// Check path for type hints
	switch {
	case strings.Contains(pathLower, "/services/"):
		return typeServer
	case strings.Contains(pathLower, "/combinations/"):
		return typeComplete
	case strings.Contains(pathLower, "/vendors/"):
		return typeCustom
	default:
		return typeBasic
	}
}

// extractDescription extracts description from first comment line of YAML file.
func extractDescription(path string) string {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if desc, found := strings.CutPrefix(line, "#"); found {
			desc = strings.TrimSpace(desc)
			if desc != "" && !strings.HasPrefix(strings.ToLower(desc), "yaml") {
				return desc
			}
		} else if line != "" {
			// First non-comment, non-empty line - stop
			break
		}
	}
	return ""
}

// generateTags generates tags from the file path.
func generateTags(path string) []string {
	tags := []string{}
	pathLower := strings.ToLower(path)

	// Add tags based on path components
	tagPatterns := map[string]string{
		"router":     "router",
		"switch":     "switch",
		"wireless":   "wireless",
		"ap":         "wireless",
		"snmp":       "snmp",
		"firewall":   "security",
		"security":   "security",
		"datacenter": "datacenter",
		"enterprise": "enterprise",
		"campus":     "enterprise",
	}

	for pattern, tag := range tagPatterns {
		if strings.Contains(pathLower, pattern) && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}

	// Add vendor tags
	vendors := []string{
		"cisco", "juniper", "arista", "dell", "aruba",
		"meraki", "fortinet", "paloalto", "extreme", "foundry",
	}
	for _, vendor := range vendors {
		if strings.Contains(pathLower, vendor) {
			tags = append(tags, vendor)
		}
	}

	return tags
}
