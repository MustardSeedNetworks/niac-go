package snmp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestMibZipMagic(t *testing.T) {
	writer := NewMibZipWriter()
	data := writer.Bytes()

	// Check magic header
	if len(data) < 7 {
		t.Fatalf("Expected at least 7 bytes for magic, got %d", len(data))
	}

	expectedMagic := "mibzip\x00"
	if string(data[:7]) != expectedMagic {
		t.Errorf("Expected magic %q, got %q", expectedMagic, string(data[:7]))
	}
}

func TestMibZipRoundTrip(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "Linux router"},
		{OID: "1.3.6.1.2.1.1.2.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.4.1.8072.3.2.10"},
		{OID: "1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(12345678)},
		{OID: "1.3.6.1.2.1.1.4.0", Type: gosnmp.OctetString, Value: "admin@example.com"},
		{OID: "1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: "router.example.com"},
	}

	// Compress
	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressed := writer.Bytes()
	t.Logf("Compressed %d entries into %d bytes", len(entries), len(compressed))

	// Decompress
	reader, err := NewMibZipReader(compressed)
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Verify
	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}

	for i, orig := range entries {
		exp := expanded[i]
		if exp.OID != orig.OID {
			t.Errorf("Entry %d: OID mismatch: expected %s, got %s", i, orig.OID, exp.OID)
		}
		if exp.Type != orig.Type {
			t.Errorf("Entry %d: Type mismatch: expected %v, got %v", i, orig.Type, exp.Type)
		}
	}
}

func TestMibZipIntegerTypes(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.2.2.1.1.1", Type: gosnmp.Integer, Value: 42},
		{OID: "1.3.6.1.2.1.2.2.1.1.2", Type: gosnmp.Integer, Value: -100},
		{OID: "1.3.6.1.2.1.2.2.1.1.3", Type: gosnmp.Integer, Value: 0},
		{OID: "1.3.6.1.2.1.2.2.1.1.4", Type: gosnmp.Integer, Value: 32767},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	for i, orig := range entries {
		exp := expanded[i]
		origVal := orig.Value.(int)
		expVal := exp.Value.(int)
		if expVal != origVal {
			t.Errorf("Entry %d: Value mismatch: expected %d, got %d", i, origVal, expVal)
		}
	}
}

func TestMibZipCounterTypes(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.2.2.1.10.1", Type: gosnmp.Counter32, Value: uint32(1000000)},
		{OID: "1.3.6.1.2.1.2.2.1.10.2", Type: gosnmp.Counter32, Value: uint32(0xFFFFFFFF)},
		{OID: "1.3.6.1.2.1.2.2.1.11.1", Type: gosnmp.Counter64, Value: uint64(9876543210123)},
		{OID: "1.3.6.1.2.1.2.2.1.12.1", Type: gosnmp.Gauge32, Value: uint32(500)},
		{OID: "1.3.6.1.2.1.2.2.1.13.1", Type: gosnmp.TimeTicks, Value: uint32(123456)},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}

	// Check Counter32 values
	if v := expanded[0].Value.(uint32); v != 1000000 {
		t.Errorf("Counter32 mismatch: expected 1000000, got %d", v)
	}
	if v := expanded[1].Value.(uint32); v != 0xFFFFFFFF {
		t.Errorf("Counter32 max mismatch: expected %d, got %d", uint32(0xFFFFFFFF), v)
	}

	// Check Counter64
	if v := expanded[2].Value.(uint64); v != 9876543210123 {
		t.Errorf("Counter64 mismatch: expected 9876543210123, got %d", v)
	}
}

func TestMibZipIPAddress(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.4.20.1.1.10.0.0.1", Type: gosnmp.IPAddress, Value: "10.0.0.1"},
		{OID: "1.3.6.1.2.1.4.20.1.1.192.168.1.1", Type: gosnmp.IPAddress, Value: "192.168.1.1"},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}

	if v := expanded[0].Value.(string); v != "10.0.0.1" {
		t.Errorf("IP address mismatch: expected 10.0.0.1, got %s", v)
	}
	if v := expanded[1].Value.(string); v != "192.168.1.1" {
		t.Errorf("IP address mismatch: expected 192.168.1.1, got %s", v)
	}
}

func TestMibZipNullTypes(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.1.100.0", Type: gosnmp.NoSuchObject, Value: nil},
		{OID: "1.3.6.1.2.1.1.101.0", Type: gosnmp.NoSuchInstance, Value: nil},
		{OID: "1.3.6.1.2.1.1.102.0", Type: gosnmp.EndOfMibView, Value: nil},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}

	for i, exp := range expanded {
		if exp.Type != entries[i].Type {
			t.Errorf("Entry %d: Type mismatch: expected %v, got %v", i, entries[i].Type, exp.Type)
		}
	}
}

func TestMibZipOIDValue(t *testing.T) {
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.1.2.0", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.4.1.9.1.1"},
		{OID: "1.3.6.1.2.1.1.9.1.2.1", Type: gosnmp.ObjectIdentifier, Value: "1.3.6.1.2.1.31.1.1"},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if v := expanded[0].Value.(string); v != "1.3.6.1.4.1.9.1.1" {
		t.Errorf("OID value mismatch: expected 1.3.6.1.4.1.9.1.1, got %s", v)
	}
}

func TestIsMibZipFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mibzip-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid mibzip file
	mibzipPath := filepath.Join(tmpDir, "test.mibzip")
	writer := NewMibZipWriter()
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "test"},
	}
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if err := os.WriteFile(mibzipPath, writer.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write mibzip file: %v", err)
	}

	// Create a non-mibzip file
	textPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(textPath, []byte("not a mibzip file"), 0644); err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}

	// Test detection
	isMibzip, err := IsMibZipFile(mibzipPath)
	if err != nil {
		t.Errorf("IsMibZipFile error: %v", err)
	}
	if !isMibzip {
		t.Error("Expected mibzip file to be detected as mibzip")
	}

	isText, err := IsMibZipFile(textPath)
	if err != nil {
		t.Errorf("IsMibZipFile error for text: %v", err)
	}
	if isText {
		t.Error("Text file should not be detected as mibzip")
	}
}

func TestMibZipEmptyEntries(t *testing.T) {
	writer := NewMibZipWriter()
	if err := writer.Compress([]WalkEntry{}); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	data := writer.Bytes()
	// Should only contain magic
	if len(data) != 7 {
		t.Errorf("Expected only magic (7 bytes) for empty entries, got %d", len(data))
	}
}

func TestMibZipLargeSubIds(t *testing.T) {
	// Test OIDs with large sub-identifiers (e.g., enterprise OIDs)
	entries := []WalkEntry{
		{OID: "1.3.6.1.4.1.9.9.42.1.2.10.1.1.12345", Type: gosnmp.Integer, Value: 100},
		{OID: "1.3.6.1.4.1.2636.3.1.13.1.6.1", Type: gosnmp.OctetString, Value: "Juniper"},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	reader, err := NewMibZipReader(writer.Bytes())
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}

	for i, orig := range entries {
		if expanded[i].OID != orig.OID {
			t.Errorf("Entry %d: OID mismatch: expected %s, got %s", i, orig.OID, expanded[i].OID)
		}
	}
}

func TestCompareOIDs(t *testing.T) {
	tests := []struct {
		oid1     string
		oid2     string
		expected int
	}{
		{"1.3.6.1.2.1.1.1", "1.3.6.1.2.1.1.2", -1},
		{"1.3.6.1.2.1.1.2", "1.3.6.1.2.1.1.1", 1},
		{"1.3.6.1.2.1.1.1", "1.3.6.1.2.1.1.1", 0},
		{"1.3.6.1.2.1.1", "1.3.6.1.2.1.1.1", -1},
		{"1.3.6.1.2.1.1.1", "1.3.6.1.2.1.1", 1},
		{".1.3.6.1.2.1.1", "1.3.6.1.2.1.1", 0}, // Leading dot should be handled
		{"1.3.6.1.4.1.9", "1.3.6.1.2.1.1", 1},  // Enterprise > MIB-2
	}

	for _, tt := range tests {
		result := CompareOIDs(tt.oid1, tt.oid2)
		if result != tt.expected {
			t.Errorf("CompareOIDs(%s, %s) = %d, expected %d", tt.oid1, tt.oid2, result, tt.expected)
		}
	}
}

func TestMibZipInvalidMagic(t *testing.T) {
	invalidData := []byte("notamib")
	_, err := NewMibZipReader(invalidData)
	if err == nil {
		t.Error("Expected error for invalid magic")
	}
}

func TestMibZipDataTooShort(t *testing.T) {
	shortData := []byte("mib")
	_, err := NewMibZipReader(shortData)
	if err == nil {
		t.Error("Expected error for data too short")
	}
}

func TestEncodeDecodeOID(t *testing.T) {
	testOIDs := []string{
		"1.3.6.1.2.1.1.1",
		"1.3.6.1.4.1.9.9.42.1.2.10.1.1.12345",
		"1.3.6.1.2.1.31.1.1.1.1.1",
	}

	for _, oid := range testOIDs {
		encoded := encodeOID(oid)
		decoded := decodeOID(encoded)
		if decoded != oid {
			t.Errorf("OID round-trip failed: original %s, got %s", oid, decoded)
		}
	}
}

func TestBERLengthEncoding(t *testing.T) {
	writer := NewMibZipWriter()

	tests := []struct {
		value    int
		expected []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7F}},
		{128, []byte{0x81, 0x80}},
		{255, []byte{0x81, 0xFF}},
		{256, []byte{0x82, 0x01, 0x00}},
		{65535, []byte{0x82, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		writer.buf.Reset()
		writer.putLength(tt.value)
		result := writer.buf.Bytes()
		if !bytes.Equal(result, tt.expected) {
			t.Errorf("putLength(%d) = %v, expected %v", tt.value, result, tt.expected)
		}
	}
}

func TestMibZipInterfaceTable(t *testing.T) {
	// Simulate a typical interface table walk
	entries := []WalkEntry{
		{OID: "1.3.6.1.2.1.2.2.1.1.1", Type: gosnmp.Integer, Value: 1},
		{OID: "1.3.6.1.2.1.2.2.1.1.2", Type: gosnmp.Integer, Value: 2},
		{OID: "1.3.6.1.2.1.2.2.1.2.1", Type: gosnmp.OctetString, Value: "eth0"},
		{OID: "1.3.6.1.2.1.2.2.1.2.2", Type: gosnmp.OctetString, Value: "eth1"},
		{OID: "1.3.6.1.2.1.2.2.1.3.1", Type: gosnmp.Integer, Value: 6},  // ethernetCsmacd
		{OID: "1.3.6.1.2.1.2.2.1.3.2", Type: gosnmp.Integer, Value: 6},
		{OID: "1.3.6.1.2.1.2.2.1.5.1", Type: gosnmp.Gauge32, Value: uint32(1000000000)},
		{OID: "1.3.6.1.2.1.2.2.1.5.2", Type: gosnmp.Gauge32, Value: uint32(1000000000)},
		{OID: "1.3.6.1.2.1.2.2.1.10.1", Type: gosnmp.Counter32, Value: uint32(12345678)},
		{OID: "1.3.6.1.2.1.2.2.1.10.2", Type: gosnmp.Counter32, Value: uint32(87654321)},
		{OID: "1.3.6.1.2.1.2.2.1.16.1", Type: gosnmp.Counter32, Value: uint32(11111111)},
		{OID: "1.3.6.1.2.1.2.2.1.16.2", Type: gosnmp.Counter32, Value: uint32(22222222)},
	}

	writer := NewMibZipWriter()
	if err := writer.Compress(entries); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressed := writer.Bytes()
	originalSize := 0
	for _, e := range entries {
		originalSize += len(e.OID) + 10 // Rough estimate
	}
	t.Logf("Compressed interface table: %d entries, %d -> %d bytes",
		len(entries), originalSize, len(compressed))

	reader, err := NewMibZipReader(compressed)
	if err != nil {
		t.Fatalf("NewMibZipReader failed: %v", err)
	}

	expanded, err := reader.Expand()
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(expanded) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(expanded))
	}
}
