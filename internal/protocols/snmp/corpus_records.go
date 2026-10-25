package snmp

import (
	"bufio"
	"fmt"
	"strings"
)

// corpusRecordScanner assembles logical records for the corpus replay
// independently of walkRecordScanner. A net-snmp STRING whose quoted value
// spans physical lines is joined back into one record; every other line is
// one record. snmprec recordings are always one record per line.
//
// It deliberately shares no code with the production reader: the replay uses
// it to derive the expected value of each binding, and an oracle built from
// the code under test would agree with that code's bugs.
type corpusRecordScanner struct {
	scanner   *bufio.Scanner
	recording bool
	text      string
	pending   string
	havePend  bool
}

// Scan advances to the next logical record.
func (s *corpusRecordScanner) Scan() bool {
	raw, ok := s.nextPhysical()
	if !ok {
		return false
	}
	s.text = raw
	if s.recording || !corpusOpensMultiline(raw) {
		return true
	}
	for s.scanner.Scan() {
		next := s.scanner.Text()
		if corpusStartsRecord(next) {
			s.pending, s.havePend = next, true

			return true
		}
		s.text += "\n" + next
		if corpusQuoteClosed(next) {
			return true
		}
	}

	return true
}

// Text returns the current logical record, physical lines joined by "\n".
func (s *corpusRecordScanner) Text() string { return s.text }

// Err returns the underlying scanner's error.
func (s *corpusRecordScanner) Err() error {
	if err := s.scanner.Err(); err != nil {
		return fmt.Errorf("read record: %w", err)
	}

	return nil
}

func (s *corpusRecordScanner) nextPhysical() (string, bool) {
	if s.havePend {
		s.havePend = false

		return s.pending, true
	}
	if !s.scanner.Scan() {
		return "", false
	}

	return s.scanner.Text(), true
}

// corpusStringValue splits "OID = STRING: value" into its value when the
// declared type is a string type.
func corpusStringValue(record string) (string, bool) {
	_, rhs, ok := strings.Cut(record, "=")
	kind, value, colon := strings.Cut(rhs, ":")
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if !ok || !colon || (kind != "STRING" && kind != "OCTET STRING") {
		return "", false
	}

	return value, true
}

// corpusOpensMultiline reports a string value that opens a quote it does not
// close on the same line.
func corpusOpensMultiline(record string) bool {
	value, ok := corpusStringValue(record)
	if !ok {
		return false
	}
	value = strings.TrimLeft(value, " \t")
	if !strings.HasPrefix(value, `"`) {
		return false
	}

	return len(value) <= 1 || !corpusQuoteClosed(value)
}

// corpusStartsRecord reports a physical line that begins a new binding, which
// ends an unterminated quoted value instead of being swallowed by it.
func corpusStartsRecord(line string) bool {
	key, _, binding := strings.Cut(line, "=")
	key = strings.TrimSpace(key)
	if !binding || key == "" || strings.ContainsAny(key, " \t") {
		return false
	}
	_, numeric := NormalizeWalkOID(key)

	return numeric || strings.Contains(key, "::")
}

// corpusQuoteClosed reports a line ending in an unescaped double quote.
func corpusQuoteClosed(line string) bool {
	line = strings.TrimRight(line, " \t\r")
	if !strings.HasSuffix(line, `"`) {
		return false
	}
	body := strings.TrimSuffix(line, `"`)
	slashes := len(body) - len(strings.TrimRight(body, `\`))

	return slashes%2 == 0
}

// corpusLiteralString returns the bytes between the quotes of a STRING record
// exactly as recorded, or the bare value of an unquoted one. Escapes are not
// interpreted; the replay compares the bytes as the recording carries them.
func corpusLiteralString(record string) (string, bool) {
	value, ok := corpusStringValue(record)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, `"`) {
		return value, true
	}
	if len(value) < 2 || !corpusQuoteClosed(value) {
		return "", false
	}

	return value[1 : len(value)-1], true
}
