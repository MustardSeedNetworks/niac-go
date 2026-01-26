package snmp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// NewMibZipReader creates a new MibZip decompressor.
func NewMibZipReader(data []byte) (*MibZipReader, error) {
	if len(data) < len(mibzipMagic) {
		return nil, ErrDataTooShortForMagic
	}

	if string(data[:len(mibzipMagic)]) != mibzipMagic {
		return nil, ErrInvalidMibzipMagic
	}

	return &MibZipReader{
		data:   data,
		pos:    len(mibzipMagic),
		length: len(data),
	}, nil
}

// Expand decompresses MibZip data into walk entries.
func (r *MibZipReader) Expand() ([]WalkEntry, error) {
	currentOID := []int{1} // Start at iso(1)

	if err := r.readInitialDownCommand(); err != nil {
		return nil, err
	}

	return r.expandEntries(currentOID)
}

// readInitialDownCommand reads and validates the initial DOWN command for iso node.
func (r *MibZipReader) readInitialDownCommand() error {
	if r.pos >= r.length {
		return nil
	}

	cmd := r.get8()
	if cmd != mibCmdDown {
		return fmt.Errorf("%w: got %d", ErrExpectedDownCommand, cmd)
	}

	subID, err := r.getLength()
	if err != nil {
		return err
	}

	if subID != 1 {
		return fmt.Errorf("%w: got %d", ErrExpectedISONode, subID)
	}

	return nil
}

// expandEntries processes all commands and builds the walk entries.
func (r *MibZipReader) expandEntries(currentOID []int) ([]WalkEntry, error) {
	var entries []WalkEntry
	lastCmd := byte(0)

	for r.pos < r.length {
		cmd := r.get8()

		var err error
		currentOID, entries, err = r.processCommand(cmd, lastCmd, currentOID, entries)
		if err != nil {
			return nil, err
		}

		lastCmd = cmd
	}

	return entries, nil
}

// processCommand handles a single mibzip command.
func (r *MibZipReader) processCommand(
	cmd, lastCmd byte,
	currentOID []int,
	entries []WalkEntry,
) ([]int, []WalkEntry, error) {
	switch cmd {
	case mibCmdDown:
		return r.handleDownCommand(lastCmd, currentOID, entries)
	case mibCmdLeaf:
		return r.handleLeafCommand(lastCmd, currentOID, entries)
	case mibCmdUp:
		return r.handleUpCommand(currentOID, entries)
	default:
		return nil, nil, fmt.Errorf("%w: %d at pos %d", ErrUnknownMibzipCommand, cmd, r.pos-1)
	}
}

// handleDownCommand processes a DOWN command (descend to child node).
func (r *MibZipReader) handleDownCommand(
	lastCmd byte,
	currentOID []int,
	entries []WalkEntry,
) ([]int, []WalkEntry, error) {
	subID, err := r.getLength()
	if err != nil {
		return nil, nil, err
	}

	if lastCmd == mibCmdUp {
		// Sibling: pop and push
		currentOID = currentOID[:len(currentOID)-1]
	}

	currentOID = append(currentOID, subID)
	return currentOID, entries, nil
}

// handleLeafCommand processes a LEAF command (leaf node with value).
func (r *MibZipReader) handleLeafCommand(
	lastCmd byte,
	currentOID []int,
	entries []WalkEntry,
) ([]int, []WalkEntry, error) {
	subID, err := r.getLength()
	if err != nil {
		return nil, nil, err
	}

	if lastCmd == mibCmdLeaf {
		// Sibling leaf: replace last
		currentOID = currentOID[:len(currentOID)-1]
	}

	currentOID = append(currentOID, subID)

	asnType, value, err := r.decodeASNValue()
	if err != nil {
		return nil, nil, err
	}

	oidStr := buildOIDString(currentOID)
	entries = append(entries, WalkEntry{
		OID:   oidStr,
		Type:  asnType,
		Value: value,
	})

	return currentOID, entries, nil
}

// handleUpCommand processes an UP command (return to parent).
func (r *MibZipReader) handleUpCommand(
	currentOID []int,
	entries []WalkEntry,
) ([]int, []WalkEntry, error) {
	if len(currentOID) > 1 {
		currentOID = currentOID[:len(currentOID)-1]
	}
	return currentOID, entries, nil
}

// buildOIDString constructs an OID string from sub-IDs.
func buildOIDString(subIDs []int) string {
	var sb strings.Builder
	for i, id := range subIDs {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(strconv.Itoa(id))
	}
	return sb.String()
}

func (r *MibZipReader) get8() byte {
	if r.pos >= r.length {
		return 0
	}

	b := r.data[r.pos]
	r.pos++

	return b
}

func (r *MibZipReader) getLength() (int, error) {
	b := r.get8()
	if (b & BERHighBitMask) == 0 {
		return int(b), nil
	}

	numBytes := int(b & BERLowMask)
	if numBytes > MaxBERLengthBytes {
		return 0, fmt.Errorf("%w: %d bytes", ErrLengthTooLong, numBytes)
	}

	length := 0
	for range numBytes {
		length = (length << BitShiftByte) | int(r.get8())
	}

	return length, nil
}

func (r *MibZipReader) getBytes(n int) []byte {
	if r.pos+n > r.length {
		n = r.length - r.pos
	}

	data := r.data[r.pos : r.pos+n]
	r.pos += n

	return data
}

func (r *MibZipReader) decodeASNValue() (gosnmp.Asn1BER, any, error) {
	asnType := gosnmp.Asn1BER(r.get8())

	length, err := r.getLength()
	if err != nil {
		return 0, nil, err
	}

	switch asnType {
	case gosnmp.OctetString:
		return asnType, string(r.getBytes(length)), nil

	case gosnmp.Integer:
		data := r.getBytes(length)

		var value int64

		for i, b := range data {
			if i == 0 && (b&BERHighBitMask) != 0 {
				value = -1 // Sign extend
			}

			value = (value << BitShiftByte) | int64(b)
		}

		return asnType, int(value), nil

	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32, gosnmp.TimeTicks:
		data := r.getBytes(length)

		var value uint64

		for _, b := range data {
			value = (value << BitShiftByte) | uint64(b)
		}

		return asnType, safeUint32FromUint64(value), nil

	case gosnmp.Counter64:
		data := r.getBytes(length)

		var value uint64

		for _, b := range data {
			value = (value << BitShiftByte) | uint64(b)
		}

		return asnType, value, nil

	case gosnmp.ObjectIdentifier:
		return asnType, decodeOID(r.getBytes(length)), nil

	case gosnmp.IPAddress:
		data := r.getBytes(length)
		if len(data) >= IPAddressOctets {
			return asnType, fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3]), nil
		}

		return asnType, "", nil

	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		r.getBytes(length) // Skip

		return asnType, nil, nil

	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble:
		// Decode as string for unsupported types
		return asnType, string(r.getBytes(length)), nil
	}

	// Unreachable - all cases handled above
	return asnType, string(r.getBytes(length)), nil
}
