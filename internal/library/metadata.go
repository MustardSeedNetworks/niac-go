package library

import "strings"

const (
	// metadataMaxScanLines bounds how many lines we scan for the
	// # Description / # Use Case lines at the top of each network YAML.
	// Scanning stops on the first non-comment, non-blank line anyway.
	metadataMaxScanLines = 40
	// metadataReadBytes caps the byte window we pull off the start of
	// each file for header parsing. Leading comment blocks are rarely
	// over 1 KB; 4 KB gives margin without slurping the device table.
	metadataReadBytes = 4096
)

// parseHeaderMetadata reads the leading comment block of a network
// YAML and pulls out a description + use case. Conventions:
//
//   - A `# Description: …` line wins for description.
//   - Otherwise the first non-empty `# …` line is used.
//   - A `# Use Case: …` line is captured verbatim.
//
// Scanning stops at the first non-comment, non-blank line so we never
// read into the actual YAML body. Same parser shape as the templates
// package — kept here so the library package stays self-contained.
func parseHeaderMetadata(content []byte) (string, string) {
	head := content
	if len(head) > metadataReadBytes {
		head = head[:metadataReadBytes]
	}

	lines := strings.Split(string(head), "\n")
	limit := min(len(lines), metadataMaxScanLines)

	var description, useCase, firstComment string
	for i := range limit {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
		stripped := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if stripped == "" {
			continue
		}

		if desc, ok := matchPrefix(stripped, "Description:"); ok && description == "" {
			description = desc
		}
		if uc, ok := matchPrefix(stripped, "Use Case:"); ok && useCase == "" {
			useCase = uc
		}
		if firstComment == "" {
			firstComment = stripped
		}
	}

	if description == "" {
		description = firstComment
	}
	return description, useCase
}

func matchPrefix(line, prefix string) (string, bool) {
	if !strings.HasPrefix(strings.ToLower(line), strings.ToLower(prefix)) {
		return "", false
	}
	return strings.TrimSpace(line[len(prefix):]), true
}
