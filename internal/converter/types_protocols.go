package converter

// TTLConfig configures ICMP TTL timeout behavior (traceroute simulation).
// It is an object, not a bare integer: `ttl: 64` on a device is rejected.
type TTLConfig struct {
	// TTL is the hop count at which this device answers with ICMP time
	// exceeded, placing it at that position in a traceroute.
	TTL int `yaml:"ttl,omitempty"`

	// IP is the source address the time-exceeded message comes from.
	IP string `yaml:"ip,omitempty"`

	// Mask is the netmask paired with IP when the hop answers for a range.
	Mask string `yaml:"mask,omitempty"`
}

// DhcpServer represents DHCP server configuration.
type DhcpServer struct {
	// ClientLeases are fixed reservations handed to matching clients.
	ClientLeases []DhcpLease `yaml:"client_leases,omitempty" validate:"omitempty,dive"`

	// SubnetMask is option 1, the mask offered to clients.
	SubnetMask string `yaml:"subnet_mask,omitempty"`

	// Router is option 3, the default gateway offered to clients.
	Router string `yaml:"router,omitempty"`

	// DomainNameServer is option 6, the DNS server offered to clients.
	DomainNameServer string `yaml:"domain_name_server,omitempty"`

	// NextServerIP is the siaddr field, the TFTP server a booting client
	// should fetch its image from.
	NextServerIP string `yaml:"next_server_ip,omitempty"`

	// ServerIdentifier is option 54, this server's own address.
	ServerIdentifier string `yaml:"server_identifier,omitempty"`

	// PoolStart is the first address of the dynamic pool. The pool must sit
	// inside a routed network this config declares, or preflight rejects it.
	PoolStart string `yaml:"pool_start,omitempty"`

	// PoolEnd is the last address of the dynamic pool.
	PoolEnd string `yaml:"pool_end,omitempty"`

	// NTPServers is option 42, the NTP servers offered to clients.
	NTPServers []string `yaml:"ntp_servers,omitempty"`

	// DomainSearch is option 119, the domain search list.
	DomainSearch []string `yaml:"domain_search,omitempty"`

	// TFTPServerName is option 66, the boot server name.
	TFTPServerName string `yaml:"tftp_server_name,omitempty"`

	// BootfileName is option 67, the boot file a client should request.
	BootfileName string `yaml:"bootfile_name,omitempty"`

	// VendorSpecific is option 43, vendor-specific information as a hex
	// string.
	VendorSpecific string `yaml:"vendor_specific,omitempty"`

	// SNTPServersV6 is DHCPv6 option 31, the SNTP servers offered.
	SNTPServersV6 []string `yaml:"sntp_servers_v6,omitempty"`

	// NTPServersV6 is DHCPv6 option 56, the NTP servers offered.
	NTPServersV6 []string `yaml:"ntp_servers_v6,omitempty"`

	// SIPServersV6 is DHCPv6 option 22, the SIP servers offered.
	SIPServersV6 []string `yaml:"sip_servers_v6,omitempty"`

	// SIPDomainsV6 is DHCPv6 option 21, the SIP domain list offered.
	SIPDomainsV6 []string `yaml:"sip_domains_v6,omitempty"`
}

// DhcpLease represents a DHCP client lease.
type DhcpLease struct {
	// ClientIP is the address reserved for the matching client.
	ClientIP string `yaml:"client_ip" validate:"required,ip"`

	// MacAddrValue is the client MAC the reservation matches.
	MacAddrValue string `yaml:"mac_addr_value,omitempty"`

	// MacAddrMask masks MacAddrValue so a reservation can match a range of
	// MACs, such as a whole vendor OUI.
	MacAddrMask string `yaml:"mac_addr_mask,omitempty"`
}

// DNSServer represents DNS server configuration.
type DNSServer struct {
	// ForwardRecords are name-to-address (A) records this server answers.
	ForwardRecords []DNSRecord `yaml:"forward_records,omitempty" validate:"omitempty,dive"`

	// ReverseRecords are address-to-name (PTR) records this server answers.
	ReverseRecords []DNSRecord `yaml:"reverse_records,omitempty" validate:"omitempty,dive"`
}

// DNSRecord represents a DNS A or PTR record.
type DNSRecord struct {
	// Name is the hostname, for example clinic-rtr-01.clinic.local.
	Name string `yaml:"name" validate:"required"`

	// IP is the record's address. The key is `ip`, not `address` — an
	// `address` key is rejected by the strict loader.
	IP string `yaml:"ip" validate:"required,ip"`

	// TTL is the record's time to live in seconds.
	TTL int `yaml:"ttl,omitempty" validate:"omitempty,gte=0"`

	// RCode answers this record with a DNS response code instead of the
	// address, 0..15 — 3 is NXDOMAIN, which is how a lookup failure is
	// authored.
	RCode int `yaml:"rcode,omitempty" validate:"omitempty,gte=0,lte=15"`
}

// LldpConfig represents LLDP discovery protocol configuration.
type LldpConfig struct {
	// Enabled advertises LLDP from this device, overriding the fleet-wide
	// `discovery_protocols.lldp`.
	Enabled bool `yaml:"enabled,omitempty"`

	// AdvertiseInterval is the seconds between advertisements. Omit for 30.
	AdvertiseInterval int `yaml:"advertise_interval,omitempty"`

	// TTL is the seconds a neighbour should hold this advertisement.
	TTL int `yaml:"ttl,omitempty"`

	// SystemDescription is the advertised system description TLV.
	SystemDescription string `yaml:"system_description,omitempty"`

	// PortDescription is the advertised port description TLV.
	PortDescription string `yaml:"port_description,omitempty"`

	// ChassisIDType selects which chassis ID subtype is advertised, for
	// example mac or local.
	ChassisIDType string `yaml:"chassis_id_type,omitempty"`
}

// CdpConfig represents CDP discovery protocol configuration.
type CdpConfig struct {
	// Enabled advertises CDP from this device, overriding the fleet-wide
	// `discovery_protocols.cdp`.
	Enabled bool `yaml:"enabled,omitempty"`

	// AdvertiseInterval is the seconds between advertisements. Omit for 60.
	AdvertiseInterval int `yaml:"advertise_interval,omitempty"`

	// Holdtime is the seconds a neighbour should hold this advertisement.
	Holdtime int `yaml:"holdtime,omitempty"`

	// Version is the CDP protocol version advertised, 1 or 2.
	Version int `yaml:"version,omitempty"`

	// SoftwareVersion is the advertised software version string.
	SoftwareVersion string `yaml:"software_version,omitempty"`

	// Platform is the advertised platform string, for example
	// "cisco WS-C3750X-48P".
	Platform string `yaml:"platform,omitempty"`

	// PortID is the advertised port identifier.
	PortID string `yaml:"port_id,omitempty"`
}

// EdpConfig represents EDP discovery protocol configuration.
type EdpConfig struct {
	// Enabled advertises EDP (Extreme) from this device.
	Enabled bool `yaml:"enabled,omitempty"`

	// AdvertiseInterval is the seconds between advertisements.
	AdvertiseInterval int `yaml:"advertise_interval,omitempty"`

	// VersionString is the advertised software version.
	VersionString string `yaml:"version_string,omitempty"`

	// DisplayString is the advertised system display name.
	DisplayString string `yaml:"display_string,omitempty"`
}

// FdpConfig represents FDP discovery protocol configuration.
type FdpConfig struct {
	// Enabled advertises FDP (Foundry) from this device.
	Enabled bool `yaml:"enabled,omitempty"`

	// AdvertiseInterval is the seconds between advertisements.
	AdvertiseInterval int `yaml:"advertise_interval,omitempty"`

	// Holdtime is the seconds a neighbour should hold this advertisement.
	Holdtime int `yaml:"holdtime,omitempty"`

	// SoftwareVersion is the advertised software version string.
	SoftwareVersion string `yaml:"software_version,omitempty"`

	// Platform is the advertised platform string.
	Platform string `yaml:"platform,omitempty"`

	// PortID is the advertised port identifier.
	PortID string `yaml:"port_id,omitempty"`
}

// StpConfig represents STP/RSTP/MSTP configuration.
type StpConfig struct {
	// Enabled makes the device participate in spanning tree.
	Enabled bool `yaml:"enabled,omitempty"`

	// BridgePriority is the bridge priority; the lowest value in the segment
	// wins the root election. Multiples of 4096.
	BridgePriority uint16 `yaml:"bridge_priority,omitempty"`

	// HelloTime is the seconds between BPDUs.
	HelloTime uint16 `yaml:"hello_time,omitempty"`

	// MaxAge is the seconds a BPDU stays valid.
	MaxAge uint16 `yaml:"max_age,omitempty"`

	// ForwardDelay is the seconds a port spends in listening and learning.
	ForwardDelay uint16 `yaml:"forward_delay,omitempty"`

	// Version selects the spanning-tree flavour, for example stp, rstp
	// or mstp.
	Version string `yaml:"version,omitempty"`
}

// IcmpConfig represents ICMP/ICMPv4 configuration.
type IcmpConfig struct {
	// Enabled answers ICMP echo requests, which is what makes the device
	// pingable.
	Enabled bool `yaml:"enabled,omitempty"`

	// TTL is the TTL stamped on ICMP replies.
	TTL uint8 `yaml:"ttl,omitempty"`

	// RateLimit caps replies per second; further requests are dropped.
	RateLimit int `yaml:"rate_limit,omitempty"`

	// AddressMaskReply is the mask answered to an ICMP address mask request.
	AddressMaskReply string `yaml:"address_mask_reply,omitempty"`

	// RouterAdvertisement makes the device announce itself as an IPv4 router.
	RouterAdvertisement *IcmpRouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// IcmpRouterAdvertisement configures IPv4 router advertisements.
type IcmpRouterAdvertisement struct {
	// Period is the seconds between advertisements.
	Period int `yaml:"period,omitempty"`

	// Lifetime is the seconds a client should consider the router valid.
	Lifetime int `yaml:"lifetime,omitempty"`

	// Routers are the advertised router entries.
	Routers []IcmpRouter `yaml:"routers,omitempty"`
}

// IcmpRouter represents an advertised router entry.
type IcmpRouter struct {
	// Address is the advertised router's IPv4 address.
	Address string `yaml:"address,omitempty"`

	// Preference is the advertised preference level; higher wins.
	Preference int `yaml:"preference,omitempty"`
}

// Icmpv6Config represents ICMPv6 configuration.
type Icmpv6Config struct {
	// Enabled answers ICMPv6 echo and neighbour solicitation.
	Enabled bool `yaml:"enabled,omitempty"`

	// HopLimit is the hop limit stamped on ICMPv6 replies.
	HopLimit uint8 `yaml:"hop_limit,omitempty"`

	// RateLimit caps replies per second.
	RateLimit int `yaml:"rate_limit,omitempty"`

	// RouterAdvertisement makes the device announce itself as an IPv6 router.
	RouterAdvertisement *Icmpv6RouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// Icmpv6RouterAdvertisement configures IPv6 router advertisements.
type Icmpv6RouterAdvertisement struct {
	// Period is the seconds between advertisements.
	Period int `yaml:"period,omitempty"`

	// CurHopLimit is the hop limit clients should adopt.
	CurHopLimit int `yaml:"cur_hop_limit,omitempty"`

	// Managed sets the M flag: clients should use DHCPv6 for addresses.
	Managed int `yaml:"managed,omitempty"`

	// Other sets the O flag: clients should use DHCPv6 for other
	// configuration only.
	Other int `yaml:"other,omitempty"`

	// Lifetime is the seconds this router stays a valid default.
	Lifetime int `yaml:"lifetime,omitempty"`

	// ReachableTime is the milliseconds a neighbour is assumed reachable.
	ReachableTime int `yaml:"reachable_time,omitempty"`

	// RetransTimer is the milliseconds between neighbour solicitations.
	RetransTimer int `yaml:"retrans_timer,omitempty"`

	// MTU is the link MTU advertised to clients.
	MTU int `yaml:"mtu,omitempty"`

	// PrefixInfo are the advertised prefixes clients autoconfigure from.
	PrefixInfo []Icmpv6PrefixInfo `yaml:"prefix_info,omitempty"`
}

// Icmpv6PrefixInfo represents IPv6 prefix info options.
type Icmpv6PrefixInfo struct {
	// PrefixLength is the advertised prefix length, typically 64.
	PrefixLength int `yaml:"prefix_length,omitempty"`

	// Onlink sets the L flag: the prefix is on-link.
	Onlink int `yaml:"onlink,omitempty"`

	// Auto sets the A flag: clients may autoconfigure from this prefix.
	Auto int `yaml:"auto,omitempty"`

	// ValidLifetime is the seconds the prefix stays valid.
	ValidLifetime int `yaml:"valid_lifetime,omitempty"`

	// PreferredLifetime is the seconds the prefix stays preferred.
	PreferredLifetime int `yaml:"preferred_lifetime,omitempty"`

	// Prefix is the advertised IPv6 prefix, for example 2001:db8:1::.
	Prefix string `yaml:"prefix,omitempty"`
}

// Dhcpv6Config represents DHCPv6 server configuration.
//
// Omit the block entirely when the device should not serve DHCPv6. An empty
// `dhcpv6: {}` is a configured server: it picks up the default lifetimes on
// the next load and the device acquires a DHCPv6 service it never authored.
type Dhcpv6Config struct {
	// Enabled serves DHCPv6.
	Enabled bool `yaml:"enabled,omitempty"`

	// Pools are the IPv6 address ranges handed out.
	Pools []Dhcpv6Pool `yaml:"pools,omitempty"`

	// PreferredLifetime is the seconds a lease stays preferred.
	PreferredLifetime uint32 `yaml:"preferred_lifetime,omitempty"`

	// ValidLifetime is the seconds a lease stays valid.
	ValidLifetime uint32 `yaml:"valid_lifetime,omitempty"`

	// Preference is the server preference, 0..255; a client prefers the
	// highest.
	Preference uint8 `yaml:"preference,omitempty"`

	// DNSServers are the DNS servers offered to clients.
	DNSServers []string `yaml:"dns_servers,omitempty"`

	// DomainList is the domain search list offered to clients.
	DomainList []string `yaml:"domain_list,omitempty"`

	// SNTPServers are the SNTP servers offered to clients.
	SNTPServers []string `yaml:"sntp_servers,omitempty"`

	// NTPServers are the NTP servers offered to clients.
	NTPServers []string `yaml:"ntp_servers,omitempty"`

	// SIPServers are the SIP servers offered to clients.
	SIPServers []string `yaml:"sip_servers,omitempty"`

	// SIPDomains is the SIP domain list offered to clients.
	SIPDomains []string `yaml:"sip_domains,omitempty"`
}

// Dhcpv6Pool represents an IPv6 address pool.
type Dhcpv6Pool struct {
	// Network is the pool's prefix, for example 2001:db8:1::/64.
	Network string `yaml:"network,omitempty"`

	// RangeStart is the first address handed out.
	RangeStart string `yaml:"range_start,omitempty"`

	// RangeEnd is the last address handed out.
	RangeEnd string `yaml:"range_end,omitempty"`
}
