package linklive

// FindingKind identifies one acceptance mismatch.
type FindingKind string

const (
	FindingMissingDevice    FindingKind = "missing-device"
	FindingUnexpectedDevice FindingKind = "unexpected-device"
	FindingNameConflict     FindingKind = "name-conflict"
	FindingTypeConflict     FindingKind = "type-conflict"
	FindingAddressConflict  FindingKind = "address-conflict"
	FindingMissingLink      FindingKind = "missing-link"
	FindingUnexpectedLink   FindingKind = "unexpected-link"
	FindingPortConflict     FindingKind = "port-conflict"
	FindingDuplexConflict   FindingKind = "duplex-conflict"
	FindingSpeedConflict    FindingKind = "speed-conflict"
	FindingVLANConflict     FindingKind = "vlan-conflict"
	FindingProblemConflict  FindingKind = "problem-conflict"
)

// Finding describes a deterministic difference between authored and observed truth.
type Finding struct {
	Kind     FindingKind `json:"kind"`
	Device   string      `json:"device"`
	Peer     string      `json:"peer,omitempty"`
	Expected string      `json:"expected,omitempty"`
	Observed string      `json:"observed,omitempty"`
}

// AuthoredSnapshot is the stable NIAC-side acceptance model.
type AuthoredSnapshot struct {
	Devices []AuthoredDevice
	Links   []AuthoredLink
}

// AuthoredDevice contains stable identity and addressing fields.
type AuthoredDevice struct {
	Name string
	Type string
	MAC  string
	IPv4 []string
}

// AuthoredLink contains the fields Link-Live renders for an inferred edge.
type AuthoredLink struct {
	Source     string
	Target     string
	SourceMAC  string
	TargetMAC  string
	SourcePort string
	TargetPort string
	SpeedMbps  int
	Duplex     string
	NativeVLAN int
}

// ObservedSnapshot is a sanitized Link-Live topology response.
type ObservedSnapshot struct {
	Hosts []ObservedHost `json:"hosts"`
}

// ObservedHost contains only topology acceptance fields.
type ObservedHost struct {
	HostID       int                  `json:"hostId"`
	Name         string               `json:"bestNameFormatted"`
	Type         string               `json:"displayedDeviceType"`
	MAC          string               `json:"longMfrMac"`
	IPv4         string               `json:"-"`
	WorstProblem string               `json:"worstProblem"`
	DefaultAddr  observedDefaultAddr  `json:"defaultAddr"`
	Connections  []ObservedConnection `json:"connectedHosts"`
}

type observedDefaultAddr struct {
	IPv4 string `json:"ipV4Address"`
}

// ObservedConnection is one Link-Live-inferred peer relationship.
type ObservedConnection struct {
	HostID int          `json:"connectedHostId"`
	Name   string       `json:"name"`
	MAC    string       `json:"mac"`
	IPv4   string       `json:"ipV4Address"`
	Port   string       `json:"connectedPort"`
	Edge   ObservedEdge `json:"connectedEdge"`
}

// ObservedEdge contains the rendered interface telemetry for a connection.
type ObservedEdge struct {
	Port   string `json:"compiledPort"`
	Speed  string `json:"compiledSpeed"`
	Duplex string `json:"compiledDuplex"`
	VLAN   string `json:"compiledVlan"`
}
