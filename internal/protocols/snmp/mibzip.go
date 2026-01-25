package snmp

import (
	"bytes"
	"encoding/binary"
	"io"
	"sort"

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

// mibNode represents a node in the MIB tree during compression.
type mibNode struct {
	subIDs   []int
	isLeaf   bool
	asnType  gosnmp.Asn1BER
	value    any
	children map[int]*mibNode
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

// For binary package compatibility.
var _ = binary.BigEndian
