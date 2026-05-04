package snmp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// MibZip command bytes.
const (
	mibzipMagic = "mibzip\x00" // 7 bytes
	mibCmdDown  = 0x01         // Descend to child node
	mibCmdLeaf  = 0x02         // Leaf node with value
	mibCmdUp    = 0x03         // Return to parent
)

// MibZipWriter compresses walk file entries into MibZip format.
type MibZipWriter struct {
	buf *bytes.Buffer
}

// MibZipReader expands MibZip format back into walk entries.
type MibZipReader struct {
	data   []byte
	pos    int
	length int
}

// NewMibZipWriter creates a new MibZip compressor.
func NewMibZipWriter() *MibZipWriter {
	buf := &bytes.Buffer{}
	buf.WriteString(mibzipMagic)

	return &MibZipWriter{buf: buf}
}

// Compress compresses walk entries into MibZip format.
func (w *MibZipWriter) Compress(entries []WalkEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Sort entries by OID for proper tree construction
	sortedEntries := make([]WalkEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return CompareOIDs(sortedEntries[i].OID, sortedEntries[j].OID) < 0
	})

	// Build tree structure
	root := &mibNode{subIDs: []int{1}, children: make(map[int]*mibNode)} // Start at iso(1)

	for _, entry := range sortedEntries {
		subIDs := parseOIDToSubIDs(entry.OID)
		if len(subIDs) < OIDPartsMinPDU {
			continue
		}

		// Navigate/create path in tree
		current := root

		for i := 1; i < len(subIDs)-1; i++ { // Skip first (iso) and last (leaf)
			subID := subIDs[i]

			child, exists := current.children[subID]

			if !exists {
				child = &mibNode{
					subIDs:   append(current.subIDs, subID),
					children: make(map[int]*mibNode),
				}
				current.children[subID] = child
			}

			current = child
		}

		// Add leaf
		leafSubID := subIDs[len(subIDs)-1]
		leaf := &mibNode{
			subIDs:   append(current.subIDs, leafSubID),
			isLeaf:   true,
			asnType:  entry.Type,
			value:    entry.Value,
			children: make(map[int]*mibNode),
		}
		current.children[leafSubID] = leaf
	}

	// Serialize tree - start with DOWN 1 for iso(1) root
	w.buf.WriteByte(mibCmdDown)
	w.putLength(1) // iso(1)
	w.serializeNode(root)

	return nil
}

type mibNode struct {
	subIDs   []int
	isLeaf   bool
	asnType  gosnmp.Asn1BER
	value    any
	children map[int]*mibNode
}

func (w *MibZipWriter) serializeNode(node *mibNode) {
	if node.isLeaf {
		w.buf.WriteByte(mibCmdLeaf)
		w.putLength(node.subIDs[len(node.subIDs)-1])
		w.encodeASNValue(node.asnType, node.value)

		return
	}

	// Get sorted child keys for deterministic output
	keys := make([]int, 0, len(node.children))
	for k := range node.children {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	// Process children
	for _, k := range keys {
		child := node.children[k]
		if child.isLeaf {
			w.buf.WriteByte(mibCmdLeaf)
			w.putLength(k)
			w.encodeASNValue(child.asnType, child.value)
		} else {
			w.buf.WriteByte(mibCmdDown)
			w.putLength(k)
			w.serializeNode(child)
		}
	}

	w.buf.WriteByte(mibCmdUp)
}

// putLength writes a BER-encoded length.
func (w *MibZipWriter) putLength(x int) {
	if x < BERLongFormIndicator {
		w.buf.WriteByte(safeconv.Byte(x))
	} else {
		// Count bytes needed
		size := 1

		tmp := x
		for tmp > BERMaxShortLen+BERLongFormIndicator {
			tmp >>= BitShiftByte
			size++
		}

		w.buf.WriteByte(safeconv.Byte(size | BERHighBitMask))

		for i := size - 1; i >= 0; i-- {
			w.buf.WriteByte(byte((x >> (BitShiftByte * i)) & BERFullByte))
		}
	}
}

// encodeASNValue encodes an ASN.1 value.
func (w *MibZipWriter) encodeASNValue(asnType gosnmp.Asn1BER, value any) {
	w.buf.WriteByte(byte(asnType))

	switch asnType {
	case gosnmp.OctetString:
		w.encodeOctetStringValue(value)
	case gosnmp.Integer:
		w.encodeIntegerValue(value)
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32, gosnmp.TimeTicks:
		w.encodeUnsigned32Value(value)
	case gosnmp.Counter64:
		w.encodeCounter64Value(value)
	case gosnmp.ObjectIdentifier:
		w.encodeOIDValue(value)
	case gosnmp.IPAddress:
		w.encodeIPAddressValue(value)
	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		w.putLength(0)
	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble:
		// Encode as octet string for unsupported types
		str := fmt.Sprintf("%v", value)
		w.putLength(len(str))
		w.buf.WriteString(str)
	}
}

func (w *MibZipWriter) encodeOctetStringValue(value any) {
	str := ""

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	}

	w.putLength(len(str))
	w.buf.WriteString(str)
}

func (w *MibZipWriter) encodeIntegerValue(value any) {
	var intVal int32

	switch v := value.(type) {
	case int:
		intVal = safeInt32(int64(v))
	case int32:
		intVal = v
	case int64:
		intVal = safeInt32(v)
	}

	w.encodeInteger(int64(intVal))
}

func (w *MibZipWriter) encodeUnsigned32Value(value any) {
	var uintVal uint32

	switch v := value.(type) {
	case uint:
		uintVal = safeUint32FromUint64(uint64(v))
	case uint32:
		uintVal = v
	case uint64:
		uintVal = safeUint32FromUint64(v)
	case int:
		uintVal = safeUint32FromInt(v)
	}

	w.encodeUnsigned(uint64(uintVal))
}

func (w *MibZipWriter) encodeCounter64Value(value any) {
	var uintVal uint64

	switch v := value.(type) {
	case uint64:
		uintVal = v
	case uint:
		uintVal = uint64(v)
	case uint32:
		uintVal = uint64(v)
	}

	w.encodeUnsigned(uintVal)
}

func (w *MibZipWriter) encodeOIDValue(value any) {
	oid := ""

	if v, ok := value.(string); ok {
		oid = v
	}

	encoded := encodeOID(oid)
	w.putLength(len(encoded))
	w.buf.Write(encoded)
}

func (w *MibZipWriter) encodeIPAddressValue(value any) {
	ip := ""

	if v, ok := value.(string); ok {
		ip = v
	}

	parts := strings.Split(ip, ".")

	w.putLength(IPAddressOctets)

	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		w.buf.WriteByte(safeconv.Byte(n))
	}
}

func (w *MibZipWriter) encodeInteger(value int64) {
	// Determine number of bytes needed
	size := 1

	if value < 0 {
		tmp := value
		for tmp < -128 || tmp > BERMaxShortLen {
			tmp >>= 8
			size++
		}
	} else {
		tmp := value
		for tmp > BERMaxShortLen {
			tmp >>= 8
			size++
		}
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (BitShiftByte * i)) & BERFullByte))
	}
}

func (w *MibZipWriter) encodeUnsigned(value uint64) {
	size := 1

	tmp := value
	for tmp > 255 {
		tmp >>= 8
		size++
	}
	// Add leading zero if high bit is set (to keep positive)
	if (value >> ((size - 1) * BitShiftByte)) > BERMaxShortLen {
		size++
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (BitShiftByte * i)) & BERFullByte))
	}
}

// Bytes returns the compressed data.
func (w *MibZipWriter) Bytes() []byte {
	return w.buf.Bytes()
}

// WriteTo writes the compressed data to a writer.
func (w *MibZipWriter) WriteTo(writer io.Writer) (int64, error) {
	n, err := writer.Write(w.buf.Bytes())

	return int64(n), err
}

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

// IsMibZipFile checks if a file is in MibZip format.
func IsMibZipFile(filename string) (bool, error) {
	cleanPath := filepath.Clean(filename)

	file, err := os.Open(cleanPath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() { _ = file.Close() }()

	magic := make([]byte, len(mibzipMagic))

	n, err := file.Read(magic)
	if err != nil || n < len(mibzipMagic) {
		return false, nil
	}

	return string(magic) == mibzipMagic, nil
}

// ParseMibZipFile reads and expands a MibZip file.
func ParseMibZipFile(filename string) ([]WalkEntry, error) {
	// Security: validate path
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return nil, ErrDirectoryTraversal
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mibzip file: %w", err)
	}

	reader, err := NewMibZipReader(data)
	if err != nil {
		return nil, err
	}

	return reader.Expand()
}

// CompressMibZipFile compresses a walk file to MibZip format.
func CompressMibZipFile(inputFile, outputFile string) error {
	entries, parseErr := ParseWalkFile(inputFile)
	if parseErr != nil {
		return fmt.Errorf("failed to parse walk file: %w", parseErr)
	}

	writer := NewMibZipWriter()

	compressErr := writer.Compress(entries)
	if compressErr != nil {
		return fmt.Errorf("failed to compress: %w", compressErr)
	}

	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	outFile, openErr := os.OpenFile(
		filepath.Clean(outputFile),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if openErr != nil {
		return fmt.Errorf("failed to create output file: %w", openErr)
	}

	defer func() { _ = outFile.Close() }()

	bufWriter := bufio.NewWriter(outFile)

	_, writeErr := writer.WriteTo(bufWriter)
	if writeErr != nil {
		return fmt.Errorf("failed to write: %w", writeErr)
	}

	flushErr := bufWriter.Flush()
	if flushErr != nil {
		return fmt.Errorf("failed to flush buffer: %w", flushErr)
	}

	return nil
}

// Helper functions

func parseOIDToSubIDs(oid string) []int {
	oid = strings.TrimPrefix(oid, ".")
	parts := strings.Split(oid, ".")

	subIDs := make([]int, len(parts))

	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		subIDs[i] = n
	}

	return subIDs
}

func encodeOID(oid string) []byte {
	subIDs := parseOIDToSubIDs(oid)
	if len(subIDs) < OIDPartsMinPDU {
		return nil
	}

	// First two octets are combined
	buf := []byte{safeconv.Byte(40*subIDs[0] + subIDs[1])}

	for i := 2; i < len(subIDs); i++ {
		subID := subIDs[i]
		if subID < OIDSubIDThreshold {
			buf = append(buf, safeconv.Byte(subID))
		} else {
			// Variable length encoding
			var encoded []byte
			for subID > 0 {
				encoded = append([]byte{byte(subID&BERLowMask) | BERHighBitMask}, encoded...)
				subID >>= OIDShiftBits
			}

			encoded[len(encoded)-1] &= BERLowMask // Clear high bit of last byte
			buf = append(buf, encoded...)
		}
	}

	return buf
}

func decodeOID(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// First byte encodes first two sub-IDs
	subIDs := []int{int(data[0]) / 40, int(data[0]) % 40}

	i := 1
	for i < len(data) {
		var subID int

		for {
			b := data[i]
			i++

			subID = (subID << OIDShiftBits) | int(b&BERLowMask)

			if (b & BERHighBitMask) == 0 {
				break
			}

			if i >= len(data) {
				break
			}
		}

		subIDs = append(subIDs, subID)
	}

	parts := make([]string, len(subIDs))
	for i, id := range subIDs {
		parts[i] = strconv.Itoa(id)
	}

	return strings.Join(parts, ".")
}

// CompareOIDs compares two OID strings lexicographically.
func CompareOIDs(oid1, oid2 string) int {
	subIDs1 := parseOIDToSubIDs(oid1)
	subIDs2 := parseOIDToSubIDs(oid2)

	minLen := min(len(subIDs1), len(subIDs2))

	for i := range minLen {
		if subIDs1[i] < subIDs2[i] {
			return -1
		}

		if subIDs1[i] > subIDs2[i] {
			return 1
		}
	}

	if len(subIDs1) < len(subIDs2) {
		return -1
	}

	if len(subIDs1) > len(subIDs2) {
		return 1
	}

	return 0
}

// For binary package compatibility.
var _ = binary.BigEndian
