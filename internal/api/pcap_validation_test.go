package api

import (
	"encoding/base64"
	"errors"
	"testing"
)

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
