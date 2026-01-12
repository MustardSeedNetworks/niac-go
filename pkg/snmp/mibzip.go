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
	root := &mibNode{subIds: []int{1}, children: make(map[int]*mibNode)} // Start at iso(1)

	for _, entry := range sortedEntries {
		subIds := parseOIDToSubIds(entry.OID)
		if len(subIds) < 2 {
			continue
		}

		// Navigate/create path in tree
		current := root

		for i := 1; i < len(subIds)-1; i++ { // Skip first (iso) and last (leaf)
			subId := subIds[i]

			child, exists := current.children[subId]

			if !exists {
				child = &mibNode{
					subIds:   append(current.subIds, subId),
					children: make(map[int]*mibNode),
				}
				current.children[subId] = child
			}

			current = child
		}

		// Add leaf
		leafSubId := subIds[len(subIds)-1]
		leaf := &mibNode{
			subIds:   append(current.subIds, leafSubId),
			isLeaf:   true,
			asnType:  entry.Type,
			value:    entry.Value,
			children: make(map[int]*mibNode),
		}
		current.children[leafSubId] = leaf
	}

	// Serialize tree - start with DOWN 1 for iso(1) root
	w.buf.WriteByte(mibCmdDown)
	w.putLength(1) // iso(1)
	w.serializeNode(root)

	return nil
}

type mibNode struct {
	subIds   []int
	isLeaf   bool
	asnType  gosnmp.Asn1BER
	value    any
	children map[int]*mibNode
}

func (w *MibZipWriter) serializeNode(node *mibNode) {
	if node.isLeaf {
		w.buf.WriteByte(mibCmdLeaf)
		w.putLength(node.subIds[len(node.subIds)-1])
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
	if x < 128 {
		w.buf.WriteByte(byte(x))
	} else {
		// Count bytes needed
		size := 1

		tmp := x
		for tmp > 255 {
			tmp >>= 8
			size++
		}

		w.buf.WriteByte(byte(size | 0x80))

		for i := size - 1; i >= 0; i-- {
			w.buf.WriteByte(byte((x >> (8 * i)) & 0xFF))
		}
	}
}

// encodeASNValue encodes an ASN.1 value.
func (w *MibZipWriter) encodeASNValue(asnType gosnmp.Asn1BER, value any) {
	w.buf.WriteByte(byte(asnType))

	switch asnType {
	case gosnmp.OctetString:
		str := ""

		switch v := value.(type) {
		case string:
			str = v
		case []byte:
			str = string(v)
		}

		w.putLength(len(str))
		w.buf.WriteString(str)

	case gosnmp.Integer:
		var intVal int32

		switch v := value.(type) {
		case int:
			intVal = int32(v) //nolint:gosec // G115: SNMP Integer bounded by protocol
		case int32:
			intVal = v
		case int64:
			intVal = int32(v) //nolint:gosec // G115: SNMP Integer bounded by protocol
		}

		w.encodeInteger(int64(intVal))

	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32, gosnmp.TimeTicks:
		var uintVal uint32

		switch v := value.(type) {
		case uint:
			uintVal = uint32(v) //nolint:gosec // G115: SNMP 32-bit bounded by protocol
		case uint32:
			uintVal = v
		case uint64:
			uintVal = uint32(v) //nolint:gosec // G115: SNMP 32-bit bounded by protocol
		case int:
			uintVal = uint32(v) //nolint:gosec // G115: SNMP 32-bit bounded by protocol
		}

		w.encodeUnsigned(uint64(uintVal))

	case gosnmp.Counter64:
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

	case gosnmp.ObjectIdentifier:
		oid := ""

		switch v := value.(type) {
		case string:
			oid = v
		}

		encoded := encodeOID(oid)
		w.putLength(len(encoded))
		w.buf.Write(encoded)

	case gosnmp.IPAddress:
		ip := ""

		switch v := value.(type) {
		case string:
			ip = v
		}

		parts := strings.Split(ip, ".")

		w.putLength(4)

		for _, p := range parts {
			n, _ := strconv.Atoi(p)
			w.buf.WriteByte(byte(n))
		}

	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		w.putLength(0)

	default:
		// Default: encode as octet string
		str := fmt.Sprintf("%v", value)
		w.putLength(len(str))
		w.buf.WriteString(str)
	}
}

func (w *MibZipWriter) encodeInteger(value int64) {
	// Determine number of bytes needed
	size := 1

	if value < 0 {
		tmp := value
		for tmp < -128 || tmp > 127 {
			tmp >>= 8
			size++
		}
	} else {
		tmp := value
		for tmp > 127 {
			tmp >>= 8
			size++
		}
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (8 * i)) & 0xFF))
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
	if (value >> ((size - 1) * 8)) > 127 {
		size++
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (8 * i)) & 0xFF))
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
	var entries []WalkEntry

	currentOID := []int{1} // Start at iso(1)

	// Read initial DOWN command for iso node
	if r.pos < r.length {
		cmd := r.get8()
		if cmd != mibCmdDown {
			return nil, fmt.Errorf("%w: got %d", ErrExpectedDownCommand, cmd)
		}

		subId, err := r.getLength()
		if err != nil {
			return nil, err
		}

		if subId != 1 {
			return nil, fmt.Errorf("%w: got %d", ErrExpectedISONode, subId)
		}
	}

	lastCmd := byte(0)

	for r.pos < r.length {
		cmd := r.get8()

		switch cmd {
		case mibCmdDown:
			subId, err := r.getLength()
			if err != nil {
				return nil, err
			}

			if lastCmd == mibCmdUp {
				// Sibling: pop and push
				currentOID = currentOID[:len(currentOID)-1]
			}

			currentOID = append(currentOID, subId)

		case mibCmdLeaf:
			subId, err := r.getLength()
			if err != nil {
				return nil, err
			}

			if lastCmd == mibCmdLeaf {
				// Sibling leaf: replace last
				currentOID = currentOID[:len(currentOID)-1]
			}

			currentOID = append(currentOID, subId)

			asnType, value, err := r.decodeASNValue()
			if err != nil {
				return nil, err
			}

			// Build OID string
			var sb strings.Builder

			for i, id := range currentOID {
				if i > 0 {
					sb.WriteByte('.')
				}

				sb.WriteString(strconv.Itoa(id))
			}

			oidStr := sb.String()

			entries = append(entries, WalkEntry{
				OID:   oidStr,
				Type:  asnType,
				Value: value,
			})

		case mibCmdUp:
			if len(currentOID) > 1 {
				currentOID = currentOID[:len(currentOID)-1]
			}

		default:
			return nil, fmt.Errorf("%w: %d at pos %d", ErrUnknownMibzipCommand, cmd, r.pos-1)
		}

		lastCmd = cmd
	}

	return entries, nil
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
	if (b & 0x80) == 0 {
		return int(b), nil
	}

	numBytes := int(b & 0x7F)
	if numBytes > 4 {
		return 0, fmt.Errorf("%w: %d bytes", ErrLengthTooLong, numBytes)
	}

	length := 0
	for range numBytes {
		length = (length << 8) | int(r.get8())
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
			if i == 0 && (b&0x80) != 0 {
				value = -1 // Sign extend
			}

			value = (value << 8) | int64(b)
		}

		return asnType, int(value), nil

	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32, gosnmp.TimeTicks:
		data := r.getBytes(length)

		var value uint64

		for _, b := range data {
			value = (value << 8) | uint64(b)
		}

		return asnType, uint32(value), nil //nolint:gosec // G115: SNMP 32-bit counter bounded by protocol

	case gosnmp.Counter64:
		data := r.getBytes(length)

		var value uint64

		for _, b := range data {
			value = (value << 8) | uint64(b)
		}

		return asnType, value, nil

	case gosnmp.ObjectIdentifier:
		return asnType, decodeOID(r.getBytes(length)), nil

	case gosnmp.IPAddress:
		data := r.getBytes(length)
		if len(data) >= 4 {
			return asnType, fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3]), nil
		}

		return asnType, "", nil

	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		r.getBytes(length) // Skip

		return asnType, nil, nil

	default:
		return asnType, string(r.getBytes(length)), nil
	}
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
	entries, err := ParseWalkFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse walk file: %w", err)
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	outFile, err := os.OpenFile(filepath.Clean(outputFile), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	defer func() { _ = outFile.Close() }()

	bufWriter := bufio.NewWriter(outFile)
	if _, err := writer.WriteTo(bufWriter); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}
	return nil
}

// Helper functions

func parseOIDToSubIds(oid string) []int {
	oid = strings.TrimPrefix(oid, ".")
	parts := strings.Split(oid, ".")

	subIds := make([]int, len(parts))

	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		subIds[i] = n
	}

	return subIds
}

func encodeOID(oid string) []byte {
	subIds := parseOIDToSubIds(oid)
	if len(subIds) < 2 {
		return nil
	}

	// First two octets are combined
	buf := []byte{byte(40*subIds[0] + subIds[1])}

	for i := 2; i < len(subIds); i++ {
		subId := subIds[i]
		if subId < 128 {
			buf = append(buf, byte(subId))
		} else {
			// Variable length encoding
			var encoded []byte
			for subId > 0 {
				encoded = append([]byte{byte(subId&0x7F) | 0x80}, encoded...)
				subId >>= 7
			}

			encoded[len(encoded)-1] &= 0x7F // Clear high bit of last byte
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
	subIds := []int{int(data[0]) / 40, int(data[0]) % 40}

	i := 1
	for i < len(data) {
		var subId int

		for {
			b := data[i]
			i++

			subId = (subId << 7) | int(b&0x7F)

			if (b & 0x80) == 0 {
				break
			}

			if i >= len(data) {
				break
			}
		}

		subIds = append(subIds, subId)
	}

	parts := make([]string, len(subIds))
	for i, id := range subIds {
		parts[i] = strconv.Itoa(id)
	}

	return strings.Join(parts, ".")
}

// CompareOIDs compares two OID strings lexicographically.
func CompareOIDs(oid1, oid2 string) int {
	subIds1 := parseOIDToSubIds(oid1)
	subIds2 := parseOIDToSubIds(oid2)

	minLen := min(len(subIds1), len(subIds2))

	for i := range minLen {
		if subIds1[i] < subIds2[i] {
			return -1
		}

		if subIds1[i] > subIds2[i] {
			return 1
		}
	}

	if len(subIds1) < len(subIds2) {
		return -1
	}

	if len(subIds1) > len(subIds2) {
		return 1
	}

	return 0
}

// For binary package compatibility.
var _ = binary.BigEndian
