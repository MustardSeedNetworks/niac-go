package snmp

import (
	"bytes"
	"encoding/binary"
	"io"

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

// mibNode represents a node in the MIB tree structure.
type mibNode struct {
	subIDs   []int
	isLeaf   bool
	asnType  gosnmp.Asn1BER
	value    any
	children map[int]*mibNode
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

// For binary package compatibility.
var _ = binary.BigEndian
