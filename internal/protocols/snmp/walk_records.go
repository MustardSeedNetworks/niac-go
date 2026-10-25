package snmp

import (
	"bufio"
	"fmt"
	"strings"
)

// walkRecordScanner retains physical newlines and whitespace inside net-snmp
// quoted strings. A new binding terminates a malformed unfinished string.
type walkRecordScanner struct {
	scanner      *bufio.Scanner
	text         string
	line         int
	physicalLine int
	pending      string
	hasPending   bool
	err          error
}

func (s *walkRecordScanner) Scan() bool {
	var line string
	if s.hasPending {
		line = s.pending
		s.hasPending = false
	} else {
		if !s.scanner.Scan() {
			return false
		}
		s.physicalLine++
		line = s.scanner.Text()
	}
	s.line = s.physicalLine
	s.text = strings.TrimSpace(line)
	value, quoted := quotedWalkString(s.text)
	if !quoted || (len(value) > 1 && hasClosingWalkQuote(value)) {
		return true
	}
	var record strings.Builder
	record.WriteString(strings.TrimLeft(line, " \t"))
	for s.scanner.Scan() {
		s.physicalLine++
		next := s.scanner.Text()
		if isWalkRecordBoundary(next) {
			s.pending, s.hasPending = next, true
			s.text = record.String()
			return true
		}
		if record.Len()+len(next)+1 > walkScanBufMax {
			s.err = fmt.Errorf("quoted walk value exceeds %d bytes", walkScanBufMax)
			return false
		}
		record.WriteByte('\n')
		record.WriteString(next)
		if hasClosingWalkQuote(next) {
			s.text = record.String()
			return true
		}
	}
	s.text = record.String()
	return true
}

func (s *walkRecordScanner) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.scanner.Err()
}

func quotedWalkString(line string) (string, bool) {
	_, rest, ok := strings.Cut(line, "=")
	if !ok {
		return "", false
	}
	typ, value, ok := strings.Cut(rest, ":")
	typ = strings.ToUpper(strings.TrimSpace(typ))
	value = strings.TrimSpace(value)
	return value, ok && (typ == snmpTypeSTRING || typ == "OCTET STRING") &&
		strings.HasPrefix(value, "\"")
}

func hasClosingWalkQuote(value string) bool {
	value = strings.TrimRight(value, " \t\r")
	if !strings.HasSuffix(value, "\"") {
		return false
	}
	escaped := false
	for i := len(value) - 1; i > 0 && value[i-1] == '\\'; i-- {
		escaped = !escaped
	}
	return !escaped
}

func isWalkRecordBoundary(line string) bool {
	key, _, ok := strings.Cut(line, "=")
	key = strings.TrimSpace(key)
	if !ok || strings.ContainsAny(key, " \t") {
		return false
	}
	_, resolved := NormalizeWalkOID(key)
	return resolved || strings.Contains(key, "::")
}
