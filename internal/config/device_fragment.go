package config

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// DeviceFragment is one device's byte range inside a whole-config YAML
// document. The detail pane edits a single device, but the file belongs to the
// operator: re-serialising the whole config to persist one edit reformats it
// and drops the comments they wrote, so the edit is put back as a byte range
// instead — everything outside [Start, End) is copied through untouched.
//
// This mirrors ui/src/utils/device-fragment.ts, which did the same splice in
// the browser while a per-device save still went through the whole-config
// endpoint. That endpoint is now admin-only (#2173), so the splice moved here.
type DeviceFragment struct {
	// Start is the offset of the first byte of the device's `- ` line.
	Start int
	// End is the offset just past the device's block.
	End int
	// Indent is the whitespace the list items sit at, restored on splice.
	Indent string
}

// FindDeviceFragment locates one device in a whole-config YAML document by
// name. It reports false when the document does not parse, has no `devices`
// sequence, or has no device by that name — the caller then falls back to
// re-serialising rather than splicing on a guess.
func FindDeviceFragment(source, deviceName string) (DeviceFragment, bool) {
	devices, ok := deviceSequence(source)
	if !ok {
		return DeviceFragment{}, false
	}

	lines := strings.SplitAfter(source, "\n")
	offsets := lineOffsets(lines)

	for _, item := range devices {
		if item.Kind != yaml.MappingNode || mappingValue(item, "name") != deviceName {
			continue
		}
		// item.Line is 1-based and points at the `- name:` line, because a
		// block mapping's first key shares that line with the sequence marker.
		if item.Line < 1 || item.Line > len(lines) {
			return DeviceFragment{}, false
		}

		start := offsets[item.Line-1]

		indent, marked := markerIndent(lines[item.Line-1])
		if !marked {
			return DeviceFragment{}, false
		}

		end := blockEnd(lines, offsets, item.Line-1, len(indent))

		return DeviceFragment{Start: start, End: end, Indent: indent}, true
	}

	return DeviceFragment{}, false
}

// SpliceDeviceFragment puts an edited device back into the whole config,
// re-indented to the depth the list uses. Every byte outside the fragment's
// range is preserved exactly, including comments and the spacing between
// devices.
func SpliceDeviceFragment(source string, fragment DeviceFragment, replacement string) string {
	marker := fragment.Indent + "- "
	bodyIndent := strings.Repeat(" ", len(marker))

	var block strings.Builder

	for i, line := range strings.Split(strings.TrimSuffix(replacement, "\n"), "\n") {
		if i > 0 {
			block.WriteString("\n")
		}

		if line == "" {
			continue
		}

		if i == 0 {
			block.WriteString(marker)
		} else {
			block.WriteString(bodyIndent)
		}

		block.WriteString(line)
	}

	return source[:fragment.Start] + block.String() + "\n" + source[fragment.End:]
}

// deviceSequence returns the items of the top-level `devices` sequence.
func deviceSequence(source string) ([]*yaml.Node, bool) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, false
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil, false
	}

	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, false
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "devices" {
			continue
		}

		value := root.Content[i+1]
		if value.Kind != yaml.SequenceNode {
			return nil, false
		}

		return value.Content, true
	}

	return nil, false
}

// blockEnd walks forward from the device's `- ` line to the first line that
// leaves the item — a non-blank line indented no deeper than the marker, which
// is the next device, a comment written at the list's own level, or a dedented
// key. Trailing blank lines are then given back to the gap between devices, so
// editing one device does not close up the spacing around it.
func blockEnd(lines []string, offsets []int, markerIndex, indent int) int {
	end := len(lines)

	for i := markerIndex + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}

		if lineIndent(lines[i]) <= indent {
			end = i

			break
		}
	}

	for end > markerIndex+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	if end >= len(lines) {
		return offsets[len(offsets)-1]
	}

	return offsets[end]
}

// markerIndent returns the whitespace before a sequence item's `- ` marker.
func markerIndent(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, "- ") {
		return "", false
	}

	return line[:len(line)-len(trimmed)], true
}

// lineIndent counts the leading spaces of a line.
func lineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

// lineOffsets returns the byte offset each line starts at, plus a final entry
// for the end of the document so callers can address "just past the last line".
func lineOffsets(lines []string) []int {
	offsets := make([]int, len(lines)+1)

	total := 0
	for i, line := range lines {
		offsets[i] = total
		total += len(line)
	}

	offsets[len(lines)] = total

	return offsets
}

// mappingValue returns a mapping's scalar value for key, or "".
func mappingValue(mapping *yaml.Node, key string) string {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value
		}
	}

	return ""
}
