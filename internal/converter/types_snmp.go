package converter

// SnmpAgent represents SNMP agent configuration.
type SnmpAgent struct {
	// Enabled serves SNMP for this device. Omit to inherit the default,
	// which is enabled whenever the block is present.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Community is the v1/v2c read community string. Omit for "public".
	Community string `yaml:"community,omitempty"`

	// SysName is sysName.0, the device's SNMP name. It overrides whatever the
	// walk file carries.
	SysName string `yaml:"sysname,omitempty"`

	// SysDescr is sysDescr.0, the description string a scanner reads to
	// identify the platform and OS version.
	SysDescr string `yaml:"sysdescr,omitempty"`

	// SysContact is sysContact.0. Starter walks carry a placeholder here;
	// keep real customer contacts out of shipped content.
	SysContact string `yaml:"syscontact,omitempty"`

	// SysLocation is sysLocation.0. Starter walks carry a placeholder here;
	// keep real customer locations out of shipped content.
	SysLocation string `yaml:"syslocation,omitempty"`

	// WalkFile is the recorded SNMP walk this device replays, resolved
	// against `include_path` when relative. It supplies every OID not
	// overridden by the fields above or by `add_mibs`.
	WalkFile string `yaml:"walk_file,omitempty"`

	// WalkFiles is several walk files merged in order, later entries winning.
	WalkFiles []string `yaml:"walk_files,omitempty"`

	// AddMibs sets or overrides individual OIDs on top of the walk.
	AddMibs []AddMib `yaml:"add_mibs,omitempty" validate:"omitempty,dive"`

	// CommunityIncludes serve a different walk file per community string, so
	// one device can answer differently to different pollers.
	CommunityIncludes []CommunityInclude `yaml:"community_includes,omitempty" validate:"omitempty,dive"`

	// AccessList restricts which source addresses may poll this agent.
	AccessList []string `yaml:"access_list,omitempty"`

	// SnmpAddr binds the agent to a specific address when the device has
	// more than one.
	SnmpAddr string `yaml:"snmp_addr,omitempty"`

	// Dot1DFdbTable injects a bridge forwarding-database entry, so a scanner
	// walking dot1dTpFdbTable sees this device's neighbours.
	Dot1DFdbTable *FdbTableConfig `yaml:"dot1d_fdb_table,omitempty"`

	// Dot1QFdbTable injects a VLAN-aware forwarding-database entry
	// (dot1qTpFdbTable).
	Dot1QFdbTable *FdbTableConfig `yaml:"dot1q_fdb_table,omitempty"`

	// Traps sends SNMP notifications to receivers on cold start and on link
	// state changes.
	Traps *TrapsConfig `yaml:"traps,omitempty"`
}

// AddMib represents a MIB override or addition.
type AddMib struct {
	// OID is the numeric object identifier, for example 1.3.6.1.2.1.1.5.0.
	OID string `yaml:"oid" validate:"required"`

	// Type is the SNMP value type, for example STRING, INTEGER, OID,
	// Counter32, Gauge32 or Hex-STRING.
	Type string `yaml:"type" validate:"required"`

	// Value is the value served for the OID, interpreted per `type`.
	Value string `yaml:"value" validate:"required"`
}

// CommunityInclude represents a community-specific walk include.
type CommunityInclude struct {
	// Community is the community string that selects this walk.
	Community string `yaml:"community" validate:"required"`

	// WalkFile is the walk served to pollers using that community.
	WalkFile string `yaml:"walk_file" validate:"required"`
}

// FdbTableConfig configures SNMP forwarding database table injection.
type FdbTableConfig struct {
	// Port is the bridge port number the entry is learned on.
	Port int `yaml:"port,omitempty"`

	// VLAN is the VLAN id the entry belongs to.
	VLAN int `yaml:"vlan,omitempty"`
}

// Snmpv3Config represents SNMPv3 user / auth / priv configuration.
//
// Added 2026-05-27. NOT license-gated — SNMPv3 is the only safe SNMP
// version (v1/v2c send credentials in cleartext) and is free for all
// NIAC tiers. The actual SNMPv3 packet path lives in
// internal/protocols/snmp/.
type Snmpv3Config struct {
	// Enabled answers SNMPv3 requests. Independent of `snmp_agent`, which
	// serves v1/v2c.
	Enabled bool `yaml:"enabled,omitempty"`

	// EngineID is the USM engine identifier as a hex string. Omit to derive
	// one from the device's MAC.
	EngineID string `yaml:"engine_id,omitempty"`

	// Users are the USM accounts this agent accepts.
	Users []Snmpv3User `yaml:"users,omitempty"`
}

// Snmpv3User represents one SNMPv3 USM user record.
type Snmpv3User struct {
	// Username is the USM security name.
	Username string `yaml:"username" validate:"required"`

	// AuthProtocol is the authentication digest. "none" is noAuthNoPriv.
	AuthProtocol string `yaml:"auth_protocol,omitempty" validate:"omitempty,oneof=none md5 sha sha256 sha512"`

	// AuthPassword is the authentication passphrase, at least 8 characters.
	// Simulated credentials only.
	AuthPassword string `yaml:"auth_password,omitempty"`

	// PrivProtocol is the privacy cipher. Requires an auth protocol —
	// there is no privNoAuth mode in USM.
	PrivProtocol string `yaml:"priv_protocol,omitempty" validate:"omitempty,oneof=none des aes aes192 aes256"`

	// PrivPassword is the privacy passphrase, at least 8 characters.
	// Simulated credentials only.
	PrivPassword string `yaml:"priv_password,omitempty"`
}

// TrapsConfig represents SNMP trap configuration (v1.6.0).
type TrapsConfig struct {
	// Enabled sends SNMP notifications from this device.
	Enabled bool `yaml:"enabled,omitempty"`

	// Receivers are trap destinations as "host:port", conventionally port
	// 162.
	Receivers []string `yaml:"receivers,omitempty"`

	// Community is the community string sent with v2c traps.
	Community string `yaml:"community,omitempty"`

	// ColdStart sends a coldStart trap when the device comes up.
	ColdStart *TrapTriggerConfig `yaml:"cold_start,omitempty"`

	// LinkState sends linkUp and linkDown traps as interfaces change, which
	// is how an injected link fault reaches an NMS.
	LinkState *LinkStateTrapConfig `yaml:"link_state,omitempty"`
}

// TrapTriggerConfig configures a simple trap trigger.
type TrapTriggerConfig struct {
	// Enabled arms this trigger.
	Enabled bool `yaml:"enabled,omitempty"`

	// OnStartup fires the trap once when the session starts.
	OnStartup bool `yaml:"on_startup,omitempty"`
}

// LinkStateTrapConfig configures link up/down traps.
type LinkStateTrapConfig struct {
	// Enabled arms link-state traps.
	Enabled bool `yaml:"enabled,omitempty"`

	// LinkDown sends a trap when an interface goes down.
	LinkDown bool `yaml:"link_down,omitempty"`

	// LinkUp sends a trap when an interface comes up.
	LinkUp bool `yaml:"link_up,omitempty"`
}
