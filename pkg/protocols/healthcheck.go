package protocols

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Additional TCP port constants for health check protocols
// Note: TCPPortHTTPS (443) is already defined in tcp.go.
const (
	TCPPortLDAP     = 389
	TCPPortLDAPS    = 636
	TCPPortRTSP     = 554
	TCPPortMySQL    = 3306
	TCPPortPostgres = 5432
	TCPPortMSSQL    = 1433
	TCPPortModbus   = 502
	TCPPortDICOM    = 104
	TCPPortHL7      = 2575
	TCPPortOPCUA    = 4840
	TCPPortSMB      = 445
)

// HealthCheckHandler handles various health check protocol requests.
type HealthCheckHandler struct {
	stack *Stack
}

// NewHealthCheckHandler creates a new health check handler.
func NewHealthCheckHandler(stack *Stack) *HealthCheckHandler {
	return &HealthCheckHandler{
		stack: stack,
	}
}

// HandleTCPConnect handles TCP SYN for health check ports (returns SYN-ACK to indicate service is up).
func (h *HealthCheckHandler) HandleTCPConnect(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	port uint16,
) {
	debugLevel := h.stack.GetDebugLevel()

	// Only respond to SYN packets
	if !tcpLayer.SYN || tcpLayer.ACK {
		return
	}

	if debugLevel >= 2 {
		_, _ = fmt.Fprintf(os.Stdout, "Health check TCP SYN on port %d from %s (devices: %v)\n",
			port, ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Send SYN-ACK to indicate service is available
	h.sendSYNACK(ipLayer, tcpLayer, devices)
}

// HandleLDAPRequest handles LDAP bind/search requests.
func (h *HealthCheckHandler) HandleLDAPRequest(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	// For initial connection, send MySQL greeting packet
	if tcpLayer.SYN && !tcpLayer.ACK {
		if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < 8 { // Modbus TCP header is 7 bytes + function code
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < 6 { // Minimum DICOM PDU header
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < 3 { // Minimum MLLP envelope
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < 8 { // Minimum OPC UA header
		return
	}

	if debugLevel >= 2 {
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
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) < 4 { // Minimum NetBIOS session header
		return
	}

	if debugLevel >= 2 {
		_, _ = fmt.Fprintf(os.Stdout, "SMB request from %s (devices: %v)\n",
			ipLayer.SrcIP, getDeviceNames(devices))
	}

	response := h.generateSMBResponse(tcpLayer.Payload, devices)
	if response != nil {
		h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
	}
}

// sendSYNACK sends a TCP SYN-ACK response.
func (h *HealthCheckHandler) sendSYNACK(ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	// Get source MAC
	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "Cannot send SYN-ACK: no MAC for %s\n", ipLayer.SrcIP)
		}

		return
	}

	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       srcMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipReply := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP SYN-ACK
	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     1000, // Initial sequence number
		Ack:     tcpLayer.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  65535,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply)
	if err != nil {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing SYN-ACK: %v\n", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pkt)

	if debugLevel >= 3 {
		_, _ = fmt.Fprintf(os.Stdout, "Sent TCP SYN-ACK from %s:%d to %s:%d device=%s\n",
			ipReply.SrcIP, tcpReply.SrcPort, ipReply.DstIP, tcpReply.DstPort, device.Name)
	}
}

// sendTCPResponse sends a TCP response with payload.
func (h *HealthCheckHandler) sendTCPResponse(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	payload []byte,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 || len(payload) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		return
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       srcMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipReply := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	payloadLen := min(len(tcpLayer.Payload), 0xFFFFFFFF)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack: tcpLayer.Seq + uint32(
			payloadLen,
		),
		PSH:    true,
		ACK:    true,
		Window: 65535,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(payload))
	if err != nil {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing TCP response: %v\n", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pkt)
}

// generateLDAPResponse creates an LDAP BindResponse or SearchResultDone.
func (h *HealthCheckHandler) generateLDAPResponse(request []byte, devices []*config.Device) []byte {
	// LDAP uses BER/DER encoding. We'll send a simple BindResponse success.
	// Message format: SEQUENCE { messageID INTEGER, BindResponse APPLICATION 1 { resultCode, matchedDN, diagnosticMessage } }
	if len(request) < 5 {
		return nil
	}

	// Extract message ID from request (simplified parsing)
	var messageID byte = 1

	if len(request) > 4 && request[0] == 0x30 { // SEQUENCE tag
		if request[2] == 0x02 { // INTEGER tag for messageID
			messageID = request[4]
		}
	}

	// Build minimal BindResponse success
	// SEQUENCE { INTEGER messageID, [APPLICATION 1] { ENUMERATED 0 (success), OCTET STRING "", OCTET STRING "" } }
	response := []byte{
		0x30, 0x0c, // SEQUENCE, length 12
		0x02, 0x01, messageID, // INTEGER messageID
		0x61, 0x07, // [APPLICATION 1] BindResponse, length 7
		0x0a, 0x01, 0x00, // ENUMERATED 0 (success)
		0x04, 0x00, // matchedDN: empty OCTET STRING
		0x04, 0x00, // diagnosticMessage: empty OCTET STRING
	}

	return response
}

// generateRTSPResponse creates an RTSP OPTIONS response.
func (h *HealthCheckHandler) generateRTSPResponse(request []byte, devices []*config.Device) []byte {
	requestStr := string(request)

	// Parse CSeq from request
	cseq := "1"

	lines := strings.SplitSeq(requestStr, "\r\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "CSeq:"); ok {
			cseq = strings.TrimSpace(after)

			break
		}
	}

	deviceName := "NIAC-Camera"
	if len(devices) > 0 {
		deviceName = devices[0].Name
	}

	// Build RTSP response
	var response strings.Builder

	response.WriteString("RTSP/1.0 200 OK\r\n")
	response.WriteString(fmt.Sprintf("CSeq: %s\r\n", cseq))
	response.WriteString(fmt.Sprintf("Server: %s RTSP Server (NIAC-Go)\r\n", deviceName))
	response.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123)))
	response.WriteString("Public: OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN\r\n")
	response.WriteString("\r\n")

	return []byte(response.String())
}

// generateMySQLGreeting creates a MySQL protocol greeting packet.
func (h *HealthCheckHandler) generateMySQLGreeting(devices []*config.Device) []byte {
	serverVersion := "5.7.32-NIAC-Go"
	if len(devices) > 0 && devices[0].Name != "" {
		serverVersion = "5.7.32-" + devices[0].Name
	}

	// MySQL greeting packet format:
	// 3 bytes: payload length
	// 1 byte: sequence id (0)
	// 1 byte: protocol version (10)
	// null-terminated: server version
	// 4 bytes: connection id
	// 8 bytes: auth plugin data part 1
	// 1 byte: filler (0)
	// 2 bytes: capability flags (lower 2 bytes)
	// 1 byte: character set
	// 2 bytes: status flags
	// 2 bytes: capability flags (upper 2 bytes)
	// 1 byte: auth plugin data length
	// 10 bytes: reserved
	// 13 bytes: auth plugin data part 2 (null terminated)
	// null-terminated: auth plugin name

	versionBytes := []byte(serverVersion)
	authData := []byte("NIAC-AUTH-DATA!!")
	authPluginName := []byte("mysql_native_password")

	payloadLen := 1 + len(versionBytes) + 1 + 4 + 8 + 1 + 2 + 1 + 2 + 2 + 1 + 10 + 13 + len(authPluginName) + 1

	packet := make([]byte, 4+payloadLen)

	// Packet header
	packet[0] = byte(payloadLen)
	packet[1] = byte(payloadLen >> 8)
	packet[2] = byte(payloadLen >> 16)
	packet[3] = 0 // sequence id

	offset := 4
	packet[offset] = 10 // protocol version
	offset++

	copy(packet[offset:], versionBytes)
	offset += len(versionBytes)
	packet[offset] = 0 // null terminator
	offset++

	// Connection ID
	binary.LittleEndian.PutUint32(packet[offset:], 1)
	offset += 4

	// Auth plugin data part 1
	copy(packet[offset:], authData[:8])
	offset += 8

	packet[offset] = 0 // filler
	offset++

	// Capability flags (lower)
	binary.LittleEndian.PutUint16(packet[offset:], 0xF7FF)
	offset += 2

	packet[offset] = 0x21 // character set (utf8)
	offset++

	// Status flags
	binary.LittleEndian.PutUint16(packet[offset:], 0x0002)
	offset += 2

	// Capability flags (upper)
	binary.LittleEndian.PutUint16(packet[offset:], 0x81FF)
	offset += 2

	packet[offset] = 21 // auth plugin data length
	offset++

	// Reserved (10 bytes)
	offset += 10

	// Auth plugin data part 2
	copy(packet[offset:], authData[8:])
	offset += 12
	packet[offset] = 0 // null terminator
	offset++

	// Auth plugin name
	copy(packet[offset:], authPluginName)
	offset += len(authPluginName)
	packet[offset] = 0

	return packet
}

// generatePostgresResponse creates a PostgreSQL response.
func (h *HealthCheckHandler) generatePostgresResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 8 {
		return nil
	}

	// Check for SSL request (8 bytes: length + 80877103)
	if len(request) >= 8 {
		msgLen := binary.BigEndian.Uint32(request[:4])

		sslCode := binary.BigEndian.Uint32(request[4:8])

		if msgLen == 8 && sslCode == 80877103 {
			// SSL request - respond with 'N' (no SSL)
			return []byte{'N'}
		}
	}

	// For startup message, send AuthenticationOk followed by ReadyForQuery
	// AuthenticationOk: 'R' + length(8) + authType(0)
	// ReadyForQuery: 'Z' + length(5) + status('I')
	response := []byte{
		'R', 0, 0, 0, 8, 0, 0, 0, 0, // AuthenticationOk
		'Z', 0, 0, 0, 5, 'I', // ReadyForQuery (Idle)
	}

	return response
}

// generateMSSQLResponse creates an MS SQL TDS response.
func (h *HealthCheckHandler) generateMSSQLResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 8 {
		return nil
	}

	// TDS packet format:
	// 1 byte: type (0x04 = response)
	// 1 byte: status (0x01 = EOM)
	// 2 bytes: length (big endian)
	// 2 bytes: SPID
	// 1 byte: packet ID
	// 1 byte: window

	// Send a simple LOGINACK token response
	serverName := []byte("NIAC-MSSQL")
	if len(devices) > 0 {
		serverName = []byte(devices[0].Name)
	}

	tokenData := make([]byte, 0, 32)
	tokenData = append(tokenData, 0xAD)                     // LOGINACK token
	tokenData = append(tokenData, byte(10+len(serverName))) // length (low byte)
	tokenData = append(tokenData, 0x00)                     // length (high byte)
	tokenData = append(tokenData, 0x01)                     // interface (SQL Server)
	tokenData = append(tokenData, 0x74, 0x00, 0x00, 0x04)   // TDS version 7.4
	tokenData = append(tokenData, byte(len(serverName)))
	tokenData = append(tokenData, serverName...)
	tokenData = append(tokenData, 0x00, 0x00, 0x00, 0x00) // program version

	packetLen := 8 + len(tokenData)
	response := make([]byte, packetLen)
	response[0] = 0x04 // type: response
	response[1] = 0x01 // status: EOM
	binary.BigEndian.PutUint16(
		response[2:],
		uint16(packetLen),
	)
	response[4] = 0x00 // SPID
	response[5] = 0x00
	response[6] = 0x01 // packet ID
	response[7] = 0x00 // window
	copy(response[8:], tokenData)

	return response
}

// generateModbusResponse creates a Modbus TCP response.
func (h *HealthCheckHandler) generateModbusResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 8 {
		return nil
	}

	// Modbus TCP format:
	// 2 bytes: Transaction ID
	// 2 bytes: Protocol ID (0x0000)
	// 2 bytes: Length
	// 1 byte: Unit ID
	// 1 byte: Function code
	// n bytes: Data

	transactionID := binary.BigEndian.Uint16(request[0:2])
	protocolID := binary.BigEndian.Uint16(request[2:4])
	unitID := request[6]
	functionCode := request[7]

	// Build response
	response := make([]byte, 9)
	binary.BigEndian.PutUint16(response[0:], transactionID)
	binary.BigEndian.PutUint16(response[2:], protocolID)
	binary.BigEndian.PutUint16(response[4:], 3) // length
	response[6] = unitID
	response[7] = functionCode
	response[8] = 0x00 // Success (no exception)

	return response
}

// generateDICOMResponse creates a DICOM A-ASSOCIATE-AC response.
func (h *HealthCheckHandler) generateDICOMResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 6 {
		return nil
	}

	// DICOM PDU format:
	// 1 byte: PDU type (0x01 = A-ASSOCIATE-RQ, 0x02 = A-ASSOCIATE-AC)
	// 1 byte: reserved
	// 4 bytes: PDU length (big endian)

	pduType := request[0]
	if pduType != 0x01 { // Only respond to A-ASSOCIATE-RQ
		return nil
	}

	// Build A-ASSOCIATE-AC response (simplified)
	calledAE := "NIAC-DICOM       " // 16 chars, space padded
	callingAE := "ANY-SCU          "

	if len(devices) > 0 {
		name := devices[0].Name
		if len(name) > 16 {
			name = name[:16]
		}

		calledAE = fmt.Sprintf("%-16s", name)
	}

	// Minimal A-ASSOCIATE-AC
	response := make([]byte, 74)
	response[0] = 0x02 // A-ASSOCIATE-AC
	response[1] = 0x00 // reserved

	// PDU length (68 bytes for minimal response)
	binary.BigEndian.PutUint32(response[2:], 68)

	// Protocol version
	binary.BigEndian.PutUint16(response[6:], 1)

	// Reserved
	response[8] = 0x00
	response[9] = 0x00

	// Called AE Title (16 bytes)
	copy(response[10:26], []byte(calledAE))

	// Calling AE Title (16 bytes)
	copy(response[26:42], []byte(callingAE))

	// Reserved (32 bytes)
	// Leave as zeros

	return response
}

// generateHL7Response creates an HL7 ACK message wrapped in MLLP.
func (h *HealthCheckHandler) generateHL7Response(request []byte, devices []*config.Device) []byte {
	// MLLP envelope: <VT>(0x0B) message <FS>(0x1C) <CR>(0x0D)
	if len(request) < 3 || request[0] != 0x0B {
		return nil
	}

	// Find message ID from MSH segment
	msgID := "NIAC001"

	msgStr := string(request)
	if idx := strings.Index(msgStr, "MSH|"); idx >= 0 {
		// Extract control ID from MSH-10
		segments := strings.Split(msgStr[idx:], "|")
		if len(segments) > 9 {
			msgID = segments[9]
		}
	}

	sendingApp := "NIAC"
	if len(devices) > 0 {
		sendingApp = devices[0].Name
	}

	// Build ACK message
	timestamp := time.Now().Format("20060102150405")
	ack := fmt.Sprintf("MSH|^~\\&|%s|FACILITY|SENDER|FACILITY|%s||ACK|%s|P|2.5\rMSA|AA|%s\r",
		sendingApp, timestamp, msgID, msgID)

	// Wrap in MLLP envelope
	response := make([]byte, 0, len(ack)+3)
	response = append(response, 0x0B) // Start block
	response = append(response, []byte(ack)...)
	response = append(response, 0x1C, 0x0D) // End block + CR

	return response
}

// generateOPCUAResponse creates an OPC UA Hello acknowledgment.
func (h *HealthCheckHandler) generateOPCUAResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 8 {
		return nil
	}

	// OPC UA message format:
	// 3 bytes: message type ("HEL", "ACK", "OPN", etc.)
	// 1 byte: chunk type ('F' = final)
	// 4 bytes: message size (little endian)

	msgType := string(request[0:3])
	if msgType != "HEL" {
		return nil
	}

	// Build ACK response
	response := make([]byte, 28)
	copy(response[0:3], "ACK")
	response[3] = 'F' // final chunk

	// Message size (28 bytes)
	binary.LittleEndian.PutUint32(response[4:8], 28)

	// Protocol version
	binary.LittleEndian.PutUint32(response[8:12], 0)

	// Receive buffer size
	binary.LittleEndian.PutUint32(response[12:16], 65535)

	// Send buffer size
	binary.LittleEndian.PutUint32(response[16:20], 65535)

	// Max message size
	binary.LittleEndian.PutUint32(response[20:24], 0) // 0 = no limit

	// Max chunk count
	binary.LittleEndian.PutUint32(response[24:28], 0) // 0 = no limit

	return response
}

// generateSMBResponse creates an SMB negotiate response.
func (h *HealthCheckHandler) generateSMBResponse(request []byte, devices []*config.Device) []byte {
	if len(request) < 4 {
		return nil
	}

	// NetBIOS session header (4 bytes) + SMB header
	// Check for SMB signature (0xFF 'S' 'M' 'B')
	if len(request) < 36 {
		return nil
	}

	// Skip NetBIOS header
	smbOffset := 4
	if request[smbOffset] != 0xFF || request[smbOffset+1] != 'S' || request[smbOffset+2] != 'M' ||
		request[smbOffset+3] != 'B' {
		// Check for SMB2
		if request[smbOffset] == 0xFE && request[smbOffset+1] == 'S' && request[smbOffset+2] == 'M' &&
			request[smbOffset+3] == 'B' {
			return h.generateSMB2Response(request, devices)
		}

		return nil
	}

	// Build SMB1 negotiate response (simplified)
	response := make([]byte, 39)

	// NetBIOS header
	response[0] = 0x00 // session message
	response[1] = 0x00
	response[2] = 0x00
	response[3] = 35 // length

	// SMB header
	copy(response[4:8], []byte{0xFF, 'S', 'M', 'B'}) // signature
	response[8] = 0x72                               // Negotiate command
	// Status: SUCCESS (4 bytes)
	response[9] = 0x00
	response[10] = 0x00
	response[11] = 0x00
	response[12] = 0x00
	// Flags
	response[13] = 0x88 // case insensitive, canonicalized paths
	// Flags2
	response[14] = 0x43
	response[15] = 0xC8
	// Reserved (12 bytes)
	// TID, PID, UID, MID
	// ...remaining fields

	return response
}

// generateSMB2Response creates an SMB2 negotiate response.
func (h *HealthCheckHandler) generateSMB2Response(request []byte, devices []*config.Device) []byte {
	// Build minimal SMB2 negotiate response
	response := make([]byte, 68)

	// NetBIOS header
	response[0] = 0x00
	response[1] = 0x00
	response[2] = 0x00
	response[3] = 64

	// SMB2 header
	copy(response[4:8], []byte{0xFE, 'S', 'M', 'B'})

	// Structure size (64)
	binary.LittleEndian.PutUint16(response[8:], 64)

	// Credit charge
	binary.LittleEndian.PutUint16(response[10:], 0)

	// Status: SUCCESS
	binary.LittleEndian.PutUint32(response[12:], 0)

	// Command: NEGOTIATE (0x0000)
	binary.LittleEndian.PutUint16(response[16:], 0)

	// Credits granted
	binary.LittleEndian.PutUint16(response[18:], 1)

	// Flags: Server response
	binary.LittleEndian.PutUint32(response[20:], 1)

	return response
}
