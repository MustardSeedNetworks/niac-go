package protocols

import (
	"fmt"
	"os"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// HandleLDAPRequest handles LDAP bind/search requests.
func (h *HealthCheckHandler) HandleLDAPRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "LDAP request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Parse LDAP message and send appropriate response
	response := h.generateLDAPResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleRTSPRequest handles RTSP OPTIONS/DESCRIBE requests.
func (h *HealthCheckHandler) HandleRTSPRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "RTSP request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateRTSPResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleMySQLRequest handles MySQL connection handshake.
func (h *HealthCheckHandler) HandleMySQLRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	// For initial connection, send MySQL greeting packet
	if tcpLayer.SYN && !tcpLayer.ACK {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "MySQL connection from %s (devices: %v)\n",
				ipLayer.SrcIP, getDeviceNames(devices))
		}

		return // Let HandleTCPConnect handle SYN-ACK
	}

	// If we have payload (after handshake), send greeting
	if len(tcpLayer.Payload) > 0 {
		response := h.generateMySQLGreeting(devices)
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandlePostgresRequest handles PostgreSQL connection handshake.
func (h *HealthCheckHandler) HandlePostgresRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "PostgreSQL request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Check if this is SSL request or startup message
	response := h.generatePostgresResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleMSSQLRequest handles MS SQL Server connection.
func (h *HealthCheckHandler) HandleMSSQLRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "MSSQL request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateMSSQLResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleModbusRequest handles Modbus TCP requests.
func (h *HealthCheckHandler) HandleModbusRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < modbusMinHeader { // Modbus TCP header is 7 bytes + function code
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Modbus TCP request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateModbusResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleDICOMRequest handles DICOM association requests.
func (h *HealthCheckHandler) HandleDICOMRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < dicomMinHeader { // Minimum DICOM PDU header
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DICOM request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateDICOMResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleHL7Request handles HL7 MLLP messages.
func (h *HealthCheckHandler) HandleHL7Request(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < mllpMinEnvelope { // Minimum MLLP envelope
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "HL7 MLLP request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateHL7Response(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleOPCUARequest handles OPC UA connection requests.
func (h *HealthCheckHandler) HandleOPCUARequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < opcuaMinHeader { // Minimum OPC UA header
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "OPC UA request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateOPCUAResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// HandleSMBRequest handles SMB/CIFS connection requests.
func (h *HealthCheckHandler) HandleSMBRequest(
	_ *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < netbiosSessionHeader { // Minimum NetBIOS session header
		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "SMB request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateSMBResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}
