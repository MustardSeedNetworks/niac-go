// Package walkmeta carries the provenance header a walk file declares about
// itself. It is a leaf package on purpose: the sanitizer, the synthesiser, the
// library listing and the CLI all need the same one line, and none of them may
// depend on each other.
package walkmeta

import (
	"bufio"
	"bytes"
	"strings"
)

// Provenance says where a walk file's content came from. It answers a
// different question from library.Source, which says where the *file* came
// from on this machine (shipped, bundled or authored here).
type Provenance string

const (
	// Captured is an SNMP walk of a real device, put through `niac sanitize`.
	Captured Provenance = "captured"
	// Generated is content NIAC built from a profile. It answers, but nothing
	// ever measured it against a device.
	Generated Provenance = "generated"
	// Unknown is a walk with no header. Shipped walks may not be Unknown —
	// scripts/check-starter-walks.sh is the gate.
	Unknown Provenance = ""
)

// headerPrefix introduces the line, in the `# Key: value` form the library's
// other header metadata already uses.
const headerPrefix = "# Source:"

// maxScanLines bounds how far into a file Parse looks. A walk's header is the
// run of comment lines at the top; anything past the first OID is content.
const maxScanLines = 40

// Line renders the header line a tool writes, with no trailing newline.
func Line(provenance Provenance) string {
	return headerPrefix + " " + string(provenance)
}

// Parse reads the provenance a walk declares, or Unknown if it declares none.
// It stops at the first line that is not a comment, so it never scans a large
// capture beyond its header.
func Parse(content []byte) Provenance {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for line := 0; line < maxScanLines && scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if !strings.HasPrefix(text, "#") {
			return Unknown
		}
		// One exact spelling, matching what Line writes and what
		// scripts/check-starter-walks.sh enforces. A reader that accepted
		// more than the gate does would let a walk ship in a form the gate
		// calls unlabelled.
		switch text {
		case Line(Captured):
			return Captured
		case Line(Generated):
			return Generated
		}
	}
	return Unknown
}

// Ensure returns content carrying a provenance header. Content that already
// declares one is returned unchanged, whatever it declares: a tool re-running
// over a file must not relabel what an earlier tool decided.
func Ensure(content []byte, provenance Provenance) []byte {
	if Parse(content) != Unknown {
		return content
	}
	header := append([]byte(Line(provenance)), '\n')
	return append(header, content...)
}
