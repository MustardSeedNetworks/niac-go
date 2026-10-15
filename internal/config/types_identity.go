package config

// The services below exist for identity: each answers the one query a scanner
// or a management system classifies a device by, and stops there.

// SIPConfig answers SIP OPTIONS with UserAgent, identifying a phone to a
// scanner. Calls and registrations are not simulated.
type SIPConfig struct {
	Enabled   bool
	UserAgent string
}

// ONVIFConfig answers a WS-Discovery Probe as an ONVIF network video
// transmitter named by Name and Hardware. The camera's SOAP services behind
// the advertised address are not simulated.
type ONVIFConfig struct {
	Enabled  bool
	Name     string
	Hardware string
}

// JetDirectConfig accepts raw print jobs on TCP 9100. A non-empty PJLID also
// answers PJL INFO ID and INFO STATUS, the way an HP JetDirect does; printers
// that speak no PJL leave it empty. Jobs are discarded, never rendered.
type JetDirectConfig struct {
	Enabled bool
	PJLID   string
}

// DICOMConfig accepts associations for the Verification SOP class on TCP 104
// and 11112 and answers C-ECHO, which is how a PACS and the modalities that
// send to it check each other. Storage, worklist and query are refused at
// association: they are the clinical application and are not simulated.
type DICOMConfig struct {
	Enabled bool
}

// HL7Config acknowledges every message framed in MLLP on TCP 2575, which is
// how an interface engine checks that a clinical system is listening. The
// message is not interpreted: patient, order and result logic are the clinical
// application and are not simulated.
type HL7Config struct {
	Enabled bool
}

// ModbusConfig answers Modbus/TCP on TCP 502: Read Holding Registers and Read
// Input Registers over the first HoldingRegisters and InputRegisters
// addresses, with the exception codes a server returns for anything else.
// Every register reads zero: the process values are the control program and
// are not simulated.
type ModbusConfig struct {
	Enabled          bool
	HoldingRegisters int
	InputRegisters   int
}

// MaxModbusRegisters is the whole Modbus address space of one register type,
// addresses 0 through 65535.
const MaxModbusRegisters = 1 << 16

// EtherNetIPConfig answers the EtherNet/IP List Identity command on TCP and
// UDP 44818 with the controller's CIP Identity object, which is how a
// commissioning tool or an asset inventory names a device on the plant floor.
// Sessions, explicit messaging and implicit I/O are the control system and
// are not simulated.
type EtherNetIPConfig struct {
	Enabled       bool
	VendorID      uint16
	DeviceType    uint16
	ProductCode   uint16
	RevisionMajor uint8
	RevisionMinor uint8
	SerialNumber  uint32
	ProductName   string
}

// MaxEtherNetIPProductName is the longest product name the Identity object
// carries, a SHORT_STRING of at most 32 characters (CIP Vol. 1 §5-2.2.1.8).
const MaxEtherNetIPProductName = 32
