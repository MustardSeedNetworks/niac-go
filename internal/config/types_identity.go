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
