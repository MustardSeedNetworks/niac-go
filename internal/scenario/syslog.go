package scenario

import (
	"net"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// syslogPort is the RFC 5424 UDP port the emission path already uses.
const syslogPort = 514

// snmpTrapPort is the RFC 3417 notification receiver port.
const snmpTrapPort = 162

// nmsServiceRole is the site service that collects. It is excluded from its own
// receiver list: a collector logging its own faults to itself tells an operator
// nothing they cannot already see.
const nmsServiceRole = "NMS"

// siteSyslog points one managed device at its own site's collector.
//
// Without this the emission added by P2-3 is unreachable. It works and it is
// wire-tested, but no shipped scenario ever turned it on, so an operator who
// installed a release and injected a fault on any pack saw nothing (#2105).
//
// Devices with no site are the shared WAN core, which belongs to no site's
// collector and so stays quiet.
func siteSyslog(spec deviceSpec) *converter.SyslogConfig {
	if !reportsToSiteCollector(spec) {
		return nil
	}

	return &converter.SyslogConfig{
		Enabled: true,
		Receivers: []string{
			net.JoinHostPort(siteNMSAddress(*spec.site), strconv.Itoa(syslogPort)),
		},
	}
}

// siteTraps points one managed device's SNMP notifications at the same
// collector as its syslog (#2472). Link changes trap as they happen, which is how
// an injected link fault reaches the NMS. coldStart fires on a reboot only: on
// session start every device at once would bury the collector's log.
func siteTraps(spec deviceSpec, community string) *converter.TrapsConfig {
	if !reportsToSiteCollector(spec) {
		return nil
	}

	return &converter.TrapsConfig{
		Enabled: true,
		Receivers: []string{
			net.JoinHostPort(siteNMSAddress(*spec.site), strconv.Itoa(snmpTrapPort)),
		},
		Community: community,
		ColdStart: &converter.TrapTriggerConfig{Enabled: true},
		LinkState: &converter.LinkStateTrapConfig{Enabled: true, LinkDown: true, LinkUp: true},
	}
}

// reportsToSiteCollector is every site device but the collector itself.
func reportsToSiteCollector(spec deviceSpec) bool {
	return spec.site != nil && spec.name != spec.site.Code+"-"+nmsServiceRole+"01"
}

// siteNMSAddress derives the collector address from the service ordering rather
// than repeating a host number, so reordering serviceRoles cannot silently
// point every device at the wrong server.
func siteNMSAddress(site Site) string {
	for index, role := range serviceRoles() {
		if role == nmsServiceRole {
			return siteIP(site, vlanServers, serviceHostOffset+index+1)
		}
	}

	panic("serviceRoles() no longer contains " + nmsServiceRole)
}
