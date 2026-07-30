package walkcapture_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	niacsnmp "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/walkcapture"
)

func TestCaptureRejectsInvalidRequests(t *testing.T) {
	requests := []walkcapture.Request{
		{Target: "switch.example", Version: "2c", Community: "secret"},
		{Target: "192.0.2.10", Version: "2c"},
		{
			Target:       "192.0.2.10",
			Version:      "3",
			Username:     "monitor",
			PrivProtocol: "aes",
			PrivPassword: "privpass1",
		},
		{Target: "192.0.2.10", Version: "2c", Community: "secret", TimeoutSecs: 61},
	}
	for _, request := range requests {
		if _, err := walkcapture.Capture(context.Background(), request); !errors.Is(
			err,
			walkcapture.ErrInvalidRequest,
		) {
			t.Errorf("Capture(%+v) error = %v, want ErrInvalidRequest", request, err)
		}
	}
}

func TestCaptureWalksLiveSNMPAgent(t *testing.T) {
	device := &config.Device{
		Name: "capture-switch", Type: "switch",
		Properties: map[string]string{"sysObjectID": "1.3.6.1.4.1.9.1.2238"},
		SNMPConfig: config.SNMPConfig{Community: "capture-secret", SysName: "capture-switch"},
		Interfaces: []config.Interface{
			{Name: "GigabitEthernet1/0/1", Type: "ethernet", Speed: 1000},
		},
	}
	agent := niacsnmp.NewAgentWithCommunity(device, "capture-secret", 0)
	port, stop := serveV2Agent(t, agent, "capture-secret")
	defer stop()

	content, err := walkcapture.Capture(context.Background(), walkcapture.Request{
		Target: "127.0.0.1", Port: port, Version: "2c", Community: "capture-secret", TimeoutSecs: 5,
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	text := string(content)
	for _, expected := range []string{"1.3.6.1.2.1.1.5.0", "capture-switch", "GigabitEthernet1/0/1"} {
		if !strings.Contains(text, expected) {
			t.Errorf("captured walk missing %q", expected)
		}
	}
}

func TestCaptureWalksLiveSNMPv3Agent(t *testing.T) {
	device := &config.Device{
		Name: "secure-switch", Type: "switch",
		Properties: map[string]string{"sysObjectID": "1.3.6.1.4.1.9.1.2238"},
		SNMPConfig: config.SNMPConfig{SysName: "secure-switch"},
	}
	agent := niacsnmp.NewAgent(device, 0)
	engine, err := niacsnmp.NewV3Engine(
		&config.SNMPv3Config{Enabled: true, Users: []config.SNMPv3User{{
			Username: "monitor", AuthProtocol: "sha", AuthPassword: "authpass1",
			PrivProtocol: "aes", PrivPassword: "privpass1",
		}}},
		net.HardwareAddr{2, 0, 0, 0, 0, 1},
	)
	if err != nil {
		t.Fatalf("NewV3Engine() error = %v", err)
	}
	port, stop := serveV3Agent(t, engine, agent.ProcessPDU)
	defer stop()

	content, err := walkcapture.Capture(context.Background(), walkcapture.Request{
		Target: "127.0.0.1", Port: port, Version: "3", Username: "monitor",
		AuthProtocol: "sha", AuthPassword: "authpass1", PrivProtocol: "aes", PrivPassword: "privpass1",
		TimeoutSecs: 5,
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if !strings.Contains(string(content), "secure-switch") {
		t.Fatalf("captured v3 walk missing sysName")
	}
}

func serveV2Agent(t *testing.T, agent *niacsnmp.Agent, community string) (uint16, func()) {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("listen UDP: %v", err)
	}
	done := make(chan struct{})
	go func() {
		buffer := make([]byte, 64*1024)
		decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
		for {
			_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			count, address, readErr := conn.ReadFromUDP(buffer)
			if readErr != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			request, decodeErr := decoder.SnmpDecodePacket(buffer[:count])
			if decodeErr != nil || request.Community != community {
				continue
			}
			response := &gosnmp.SnmpPacket{
				Version: request.Version, Community: community, PDUType: gosnmp.GetResponse,
				RequestID: request.RequestID, Error: gosnmp.NoError,
				Variables: agent.ProcessPDU(
					request.PDUType,
					request.Variables,
					int(request.NonRepeaters),
					request.MaxRepetitions,
				),
			}
			payload, marshalErr := response.MarshalMsg()
			if marshalErr == nil {
				_, _ = conn.WriteToUDP(payload, address)
			}
		}
	}()
	return uint16(conn.LocalAddr().(*net.UDPAddr).Port), func() {
		close(done)
		_ = conn.Close()
	}
}

func serveV3Agent(
	t *testing.T,
	engine *niacsnmp.V3Engine,
	process niacsnmp.ProcessFunc,
) (uint16, func()) {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("listen UDP: %v", err)
	}
	done := make(chan struct{})
	go func() {
		buffer := make([]byte, 64*1024)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			count, address, readErr := conn.ReadFromUDP(buffer)
			if readErr != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			response, responseErr := engine.Respond(buffer[:count], process)
			if responseErr == nil && response != nil {
				_, _ = conn.WriteToUDP(response, address)
			}
		}
	}()
	return uint16(conn.LocalAddr().(*net.UDPAddr).Port), func() {
		close(done)
		_ = conn.Close()
	}
}
