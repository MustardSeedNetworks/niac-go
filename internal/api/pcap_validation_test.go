package api

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestValidatePCAPStructure(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "too small",
			data:    []byte{0x01, 0x02},
			wantErr: ErrFileTooSmallForPCAP,
		},
		{
			name:    "empty",
			data:    []byte{},
			wantErr: ErrFileTooSmallForPCAP,
		},
		{
			name: "invalid magic number",
			data: []byte{
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
		{
			name: "valid little-endian pcap",
			data: []byte{
				0xd4, 0xc3, 0xb2, 0xa1, // Magic (LE)
				0x02, 0x00, 0x04, 0x00, // Version 2.4
				0x00, 0x00, 0x00, 0x00, // Timezone
				0x00, 0x00, 0x00, 0x00, // Accuracy
				0xff, 0xff, 0x00, 0x00, // Snapshot length
				0x01, 0x00, 0x00, 0x00, // Link type (Ethernet)
			},
			wantErr: nil,
		},
		{
			name: "valid big-endian pcap",
			data: []byte{
				0xa1, 0xb2, 0xc3, 0xd4, // Magic (BE)
				0x00, 0x02, 0x00, 0x04, // Version 2.4
				0x00, 0x00, 0x00, 0x00, // Timezone
				0x00, 0x00, 0x00, 0x00, // Accuracy
				0x00, 0x00, 0xff, 0xff, // Snapshot length
				0x00, 0x00, 0x00, 0x01, // Link type (Ethernet)
			},
			wantErr: nil,
		},
		{
			name: "pcap too short for header",
			data: []byte{
				0xd4, 0xc3, 0xb2, 0xa1, // Magic (LE)
				0x02, 0x00, // Incomplete
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
		{
			name: "pcap wrong version",
			data: []byte{
				0xd4, 0xc3, 0xb2, 0xa1, // Magic (LE)
				0x03, 0x00, 0x04, 0x00, // Version 3.4 (invalid)
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0xff, 0xff, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00,
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
		{
			name: "pcap unknown link type",
			data: []byte{
				0xd4, 0xc3, 0xb2, 0xa1, // Magic (LE)
				0x02, 0x00, 0x04, 0x00, // Version 2.4
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0xff, 0xff, 0x00, 0x00,
				0xFF, 0x00, 0x00, 0x00, // Unknown link type (255)
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
		{
			name: "valid pcapng",
			data: []byte{
				0x0a, 0x0d, 0x0d, 0x0a, // pcapng magic
				0x00, 0x00, 0x00, 0x1c, // Block total length
				0x1a, 0x2b, 0x3c, 0x4d, // Byte-Order Magic (BE)
				0x00, 0x01, 0x00, 0x00, // Major/Minor version
				0xff, 0xff, 0xff, 0xff, // Section Length (unknown)
				0xff, 0xff, 0xff, 0xff,
				0x00, 0x00, 0x00, 0x1c, // Block total length (repeat)
			},
			wantErr: nil,
		},
		{
			name: "pcapng too short for SHB",
			data: []byte{
				0x0a, 0x0d, 0x0d, 0x0a, // pcapng magic
				0x00, 0x00, 0x00, 0x1c, // Block total length
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
		{
			name: "pcapng invalid byte-order magic",
			data: []byte{
				0x0a, 0x0d, 0x0d, 0x0a, // pcapng magic
				0x00, 0x00, 0x00, 0x1c,
				0x00, 0x00, 0x00, 0x00, // Invalid BOM
				0x00, 0x01, 0x00, 0x00,
				0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff,
				0x00, 0x00, 0x00, 0x1c,
			},
			wantErr: ErrInvalidPCAPMagicNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePCAPStructure(tt.data)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validatePCAPStructure() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("validatePCAPStructure() = nil, want %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validatePCAPStructure() = %v, want error wrapping %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePCAPGlobalHeaderEndianness(t *testing.T) {
	// Big-endian valid pcap
	beData := []byte{
		0xa1, 0xb2, 0xc3, 0xd4,
		0x00, 0x02, 0x00, 0x04, // version 2.4 big-endian
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff,
		0x00, 0x00, 0x00, 0x01, // Ethernet big-endian
	}

	if err := validatePCAPGlobalHeader(beData, true); err != nil {
		t.Errorf("big-endian valid pcap: %v", err)
	}

	// Little-endian valid pcap
	leData := []byte{
		0xd4, 0xc3, 0xb2, 0xa1,
		0x02, 0x00, 0x04, 0x00, // version 2.4 little-endian
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0xff, 0xff, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, // Ethernet little-endian
	}

	if err := validatePCAPGlobalHeader(leData, false); err != nil {
		t.Errorf("little-endian valid pcap: %v", err)
	}
}

func TestValidLinkTypes(t *testing.T) {
	// Verify some known link types are in the map
	expectedTypes := []uint32{0, 1, 6, 101, 113, 127}

	for _, lt := range expectedTypes {
		if !validLinkTypes[lt] {
			t.Errorf("link type %d should be valid", lt)
		}
	}

	// Verify unknown types are not
	unknownTypes := []uint32{2, 3, 4, 5, 999, 65535}
	for _, lt := range unknownTypes {
		if validLinkTypes[lt] {
			t.Errorf("link type %d should not be valid", lt)
		}
	}
}

func TestProcessInlineDataTooLarge(t *testing.T) {
	server, _ := newTestServer(t)

	// Create data that exceeds the size limit
	largeData := make([]byte, MaxPCAPUploadSize+1)
	encoded := base64.StdEncoding.EncodeToString(largeData)

	_, err := server.processInlineData(encoded)
	if err == nil {
		t.Error("processInlineData should fail for oversized data")
	}
}

func TestProcessInlineDataInvalidBase64(t *testing.T) {
	server, _ := newTestServer(t)

	_, err := server.processInlineData("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("processInlineData should fail for invalid base64")
	}
}

func TestProcessInlineDataInvalidPCAP(t *testing.T) {
	server, _ := newTestServer(t)

	// Valid base64 but not a valid PCAP
	data := base64.StdEncoding.EncodeToString([]byte("this is not a pcap file at all"))
	_, err := server.processInlineData(data)
	if err == nil {
		t.Error("processInlineData should fail for invalid PCAP data")
	}
}

func TestPrepareReplayRequestEmptyFileAndData(t *testing.T) {
	server, _ := newTestServer(t)

	req := ReplayRequest{File: "", InlineData: ""}
	_, err := server.prepareReplayRequest(req)
	if !errors.Is(err, ErrPcapFilePathOrDataRequired) {
		t.Errorf("prepareReplayRequest empty = %v, want %v", err, ErrPcapFilePathOrDataRequired)
	}
}

func TestPrepareReplayRequestWhitespaceFile(t *testing.T) {
	server, _ := newTestServer(t)

	req := ReplayRequest{File: "   ", InlineData: ""}
	_, err := server.prepareReplayRequest(req)
	if !errors.Is(err, ErrPcapFilePathOrDataRequired) {
		t.Errorf("prepareReplayRequest whitespace = %v, want %v", err, ErrPcapFilePathOrDataRequired)
	}
}

func TestValidatePcapFilePathEmpty(t *testing.T) {
	server, _ := newTestServer(t)

	_, err := server.validatePcapFilePath("")
	if err == nil {
		t.Error("validatePcapFilePath('') should fail")
	}
}

func TestValidatePcapFilePathNonExistent(t *testing.T) {
	server, _ := newTestServer(t)

	_, err := server.validatePcapFilePath("nonexistent.pcap")
	if err == nil {
		t.Error("validatePcapFilePath(nonexistent) should fail")
	}
}
