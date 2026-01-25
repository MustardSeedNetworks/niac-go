package protocols

import (
	"fmt"
	"os"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
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

// Health check protocol encoding constants.
const (
	// maxUint32Val is the maximum uint32 value for bounds checking.
	maxUint32Val = 0xFFFFFFFF

	// maxUint16Val is the maximum uint16 value for TCP window and other fields.
	healthCheckMaxUint16 = 65535

	// Byte shift constants.
	hcByteShift8  = 8
	hcByteShift16 = 16

	// IP protocol constants for healthcheck responses.
	hcIPv4Version   = 4     // IPv4 version field
	hcIPv4IHL       = 5     // IPv4 header length (5 = 20 bytes)
	hcIPv4TTL       = 64    // IPv4 default TTL
	hcTCPWindowSize = 65535 // Default TCP window size
	hcTCPInitialSeq = 1000  // Initial TCP sequence number for SYN-ACK

	// TDS header size.
	tdsHeaderSize = 8 // TDS packet header size

	// MySQL protocol constants.
	mysqlPacketHeader = 4  // Packet header size
	mysqlProtocolVer  = 10 // Protocol version 10

	// MySQL capability flags.
	mysqlCapFlagsLower = 0xF7FF // Lower capability flags
	mysqlStatusFlags   = 0x0002 // Status flags (autocommit)
	mysqlCapFlagsUpper = 0x81FF // Upper capability flags

	// MSSQL/TDS protocol constants.
	mssqlTokenDataCap   = 32   // Token data capacity
	mssqlLoginAckToken  = 0xAD // LOGINACK token type
	mssqlLengthExtra    = 10   // Extra bytes for length calculation
	mssqlLengthHigh     = 0x00 // Length high byte
	mssqlInterface      = 0x01 // Interface (SQL Server)
	mssqlTDSVersion1    = 0x74 // TDS version byte 1
	mssqlProgramVersion = 0x00 // Program version (zero)

	// Modbus protocol constants.
	modbusResponseLen = 9 // Basic response length
	modbusDataLen     = 3 // Data length field value

	// DICOM protocol constants.
	dicomResponseLen = 74 // A-ASSOCIATE-AC response length
	dicomPDUDataLen  = 68 // PDU data length

	// OPC-UA protocol constants.
	opcuaAckLen = 28 // OPC-UA ACK length

	// PostgreSQL protocol constants.
	postgresResponseLen = 39 // AuthenticationOk + ReadyForQuery length

	// SMB protocol constants.
	smbResponseLen  = 68 // SMB2 response length
	smbHeaderLen    = 64 // SMB2 header/structure size
	smb1ResponseLen = 39 // SMB1 response length

	// HL7 MLLP constants.
	hl7MLLPEnvelopeExtra = 3    // Extra bytes for MLLP envelope (start + end + CR)
	hl7StartBlock        = 0x0B // MLLP start block character
	hl7EndBlock          = 0x1C // MLLP end block character
	hl7CarriageReturn    = 0x0D // Carriage return character

	// Protocol minimum header sizes.
	modbusMinHeader      = 8    // Modbus TCP header + function code
	dicomMinHeader       = 6    // DICOM PDU header minimum
	mllpMinEnvelope      = 3    // MLLP envelope minimum (start + end + CR)
	opcuaMinHeader       = 8    // OPC UA header minimum
	netbiosSessionHeader = 4    // NetBIOS session header minimum
	ldapMinRequest       = 5    // LDAP minimum request size
	ldapSequenceTag      = 0x30 // LDAP SEQUENCE tag
	ldapIntegerTag       = 0x02 // LDAP INTEGER tag for message ID

	// Protocol response/parsing sizes.
	mysqlMinHeader       = 8    // MySQL handshake minimum header
	postgresMinHeader    = 8    // PostgreSQL minimum header
	smbMinHeader         = 8    // SMB minimum header
	smbSignature         = 16   // SMB signature size
	mssqlMinHeader       = 8    // TDS minimum header
	rtspMinHeader        = 9    // RTSP minimum request size ("OPTIONS ")
	modbusMinResponse    = 4    // Modbus minimum response header
	dicomAssocItemLen    = 36   // DICOM Association Context Item length
	macAddrSizeHC        = 6    // MAC address size for healthcheck
	dicomAssocRQ         = 0x01 // DICOM A-ASSOCIATE-RQ PDU type
	dicomAETitleLen      = 16   // DICOM AE title max length
	hl7MSHControlIDIndex = 9    // HL7 MSH Control ID field index (MSH-10)
	smbMinWithNetbiosHdr = 36   // SMB minimum size with NetBIOS header
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
	_ *Packet,
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

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Health check TCP SYN on port %d from %s (devices: %v)\n",
			port, ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Send SYN-ACK to indicate service is available
	h.sendSYNACK(ipLayer, tcpLayer, devices)
}
