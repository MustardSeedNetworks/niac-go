package main

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func TestPrintPacketsHexDump(t *testing.T) {
	packets := []cliclient.PacketData{
		{
			Timestamp: time.Now(),
			Length:    14,
			Device:    "router-1",
			Interface: "eth0",
			Data:      []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x08, 0x06},
		},
		{
			Timestamp: time.Now(),
			Length:    4,
			Device:    "",
			Interface: "",
			Data:      []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printPacketsHexDump panicked: %v", r)
		}
	}()
	printPacketsHexDump(packets)
}

func TestPrintPacketsHexDumpEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printPacketsHexDump panicked: %v", r)
		}
	}()
	printPacketsHexDump([]cliclient.PacketData{})
}

func TestOutputPacketsJSON(t *testing.T) {
	packets := []cliclient.PacketData{
		{
			Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Length:    14,
			Device:    "router-1",
			Interface: "eth0",
			Data:      []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputPacketsJSON panicked: %v", r)
		}
	}()
	outputPacketsJSON(packets)
}

func TestOutputDumpJSON(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{
			name: "success result",
			data: map[string]any{
				"success": true,
				"packets": []any{},
				"count":   0,
			},
		},
		{
			name: "error result",
			data: map[string]any{
				"success": false,
				"error":   "socket not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("outputDumpJSON panicked: %v", r)
				}
			}()
			outputDumpJSON(tt.data)
		})
	}
}

func TestPacketDumpStruct(t *testing.T) {
	dump := PacketDump{
		Index:   1,
		Device:  "sw1",
		HexDump: "0102",
	}

	if dump.Index != 1 {
		t.Errorf("Index = %d, want 1", dump.Index)
	}
	if dump.Device != "sw1" {
		t.Errorf("Device = %q, want %q", dump.Device, "sw1")
	}
	if dump.HexDump != "0102" {
		t.Errorf("HexDump = %q, want %q", dump.HexDump, "0102")
	}
}
