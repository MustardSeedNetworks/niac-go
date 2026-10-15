package config

// Validation of the identity services in types_identity.go.

// maxSIPUserAgentLength bounds the header a response carries; real handsets
// send a few dozen bytes.
const maxSIPUserAgentLength = 128

// validateSIP requires an enabled listener to name itself. The User-Agent is
// the whole point of the service, and it is written into a response header,
// so it must be one line of printable US-ASCII.
func (v *Validator) validateSIP(device *Device, prefix string) {
	sip := device.SIPConfig
	if sip == nil || !sip.Enabled {
		return
	}
	if !validSIPUserAgent(sip.UserAgent) {
		v.addError(prefix+".sip.user_agent",
			"SIP user_agent must be printable US-ASCII of at most 128 characters, such as Cisco-CP8841/12.8.1")
	}
}

func validSIPUserAgent(agent string) bool {
	if agent == "" || agent[0] == ' ' || len(agent) > maxSIPUserAgentLength {
		return false
	}
	for index := range len(agent) {
		if agent[index] < ' ' || agent[index] > '~' {
			return false
		}
	}
	return true
}

// maxONVIFScopeLength bounds an authored scope value; a camera's name and
// model are a few dozen bytes.
const maxONVIFScopeLength = 64

// validateONVIF requires an enabled camera to name itself: the name and
// hardware scopes are what a discovery tool lists it by.
func (v *Validator) validateONVIF(device *Device, prefix string) {
	onvif := device.ONVIFConfig
	if onvif == nil || !onvif.Enabled {
		return
	}
	if !validONVIFScope(onvif.Name) {
		v.addError(prefix+".onvif.name",
			"ONVIF name must be printable US-ASCII of at most 64 characters, such as AXIS P3265-LVE")
	}
	if !validONVIFScope(onvif.Hardware) {
		v.addError(prefix+".onvif.hardware",
			"ONVIF hardware must be printable US-ASCII of at most 64 characters, such as P3265-LVE")
	}
}

func validONVIFScope(value string) bool {
	return len(value) <= maxONVIFScopeLength && validSIPUserAgent(value)
}

// maxPJLIDLength bounds the model string; real ones run to about thirty
// characters ("HP LaserJet MFP M634").
const maxPJLIDLength = 64

// validateJetDirect keeps the PJL ID inside the quoted string the INFO ID
// answer carries: one line of printable US-ASCII with no double quote.
func (v *Validator) validateJetDirect(device *Device, prefix string) {
	jetDirect := device.JetDirectConfig
	if jetDirect == nil || jetDirect.PJLID == "" {
		return
	}
	if !validPJLID(jetDirect.PJLID) {
		v.addError(
			prefix+".jetdirect.pjl_id",
			"PJL pjl_id must be printable US-ASCII without double quotes, at most 64 characters, such as HP LaserJet MFP M634",
		)
	}
}

func validPJLID(id string) bool {
	if id[0] == ' ' || len(id) > maxPJLIDLength {
		return false
	}
	for index := range len(id) {
		if id[index] < ' ' || id[index] > '~' || id[index] == '"' {
			return false
		}
	}
	return true
}

// validateModbus keeps each register count inside the Modbus address space.
func (v *Validator) validateModbus(device *Device, prefix string) {
	modbus := device.ModbusConfig
	if modbus == nil {
		return
	}
	for _, registers := range []struct {
		field string
		count int
	}{
		{"holding_registers", modbus.HoldingRegisters},
		{"input_registers", modbus.InputRegisters},
	} {
		if registers.count < 0 || registers.count > MaxModbusRegisters {
			v.addError(prefix+".modbus."+registers.field, "Modbus "+registers.field+" must be between 0 and 65536")
		}
	}
}

// validateEtherNetIP requires a product name the Identity object can carry
// when the device answers List Identity.
func (v *Validator) validateEtherNetIP(device *Device, prefix string) {
	enip := device.EtherNetIPConfig
	if enip == nil || !enip.Enabled {
		return
	}
	if !validEtherNetIPProductName(enip.ProductName) {
		v.addError(prefix+".ethernet_ip.product_name",
			"EtherNet/IP product_name must be 1 to 32 printable ASCII characters")
	}
}

func validEtherNetIPProductName(name string) bool {
	if name == "" || len(name) > MaxEtherNetIPProductName {
		return false
	}
	for index := range len(name) {
		if name[index] < ' ' || name[index] > '~' {
			return false
		}
	}
	return true
}
