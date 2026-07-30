package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp/synth"

func networkProfiles() []DeviceProfile {
	return []DeviceProfile{
		newProfile("lab", "router", "cisco", "Catalyst C8500L-8S4X", "Cisco Catalyst C8500L-8S4X",
			"IOS XE 17.15", synth.VendorCiscoIOS, synth.TypeRouter),
		newProfile("wan", "router", "cisco", "Catalyst C8500-12X", "Cisco Catalyst C8500-12X",
			"IOS XE 17.15", synth.VendorCiscoIOS, synth.TypeRouter),
		newProfile("firewall", "firewall", "palo alto", "PA-5450", "Palo Alto PA-5450",
			"PAN-OS 11.2", synth.VendorPaloAlto, synth.TypeFirewall),
		newProfile("core", "layer3-switch", "cisco", "Nexus 9508", "Cisco Nexus 9508",
			"NX-OS 10.5", synth.VendorCiscoIOS, synth.TypeRouter),
	}
}

func campusProfiles() []DeviceProfile {
	return []DeviceProfile{
		newProfile("distribution", "switch", "cisco", "Catalyst C9606R", "Cisco Catalyst C9606R",
			"IOS XE 17.15", synth.VendorCiscoIOS, synth.TypeSwitch),
		newProfile("access", "switch", "cisco", "Catalyst C9350-48HX", "Cisco Catalyst C9350-48HX",
			"IOS XE 17.18", synth.VendorCiscoIOS, synth.TypeSwitch),
		newProfile("server-switch", "switch", "cisco", "Nexus 93180YC-FX3", "Cisco Nexus 93180YC-FX3",
			"NX-OS 10.5", synth.VendorCiscoIOS, synth.TypeSwitch),
		newProfile("ap", "access-point", "cisco", "Wireless CW9178I", "Cisco Wireless CW9178I",
			"IOS XE 17.15.3", synth.VendorCiscoIOS, synth.TypeAccessPoint),
	}
}

func endpointProfiles() []DeviceProfile {
	return []DeviceProfile{
		newProfile("workstation", "host", "dell", "OptiPlex 7020", "Dell OptiPlex 7020",
			"Windows 11 Enterprise", synth.VendorGeneric, synth.TypeHost),
		newProfile("server", "server", "dell", "PowerEdge R660", "Dell PowerEdge R660",
			"Ubuntu Server 26.04", synth.VendorGeneric, synth.TypeServer),
		newProfile("controller", "server", "cisco", "Catalyst 9800-L", "Cisco Catalyst 9800-L",
			"IOS XE 17.15.3", synth.VendorCiscoIOS, synth.TypeRouter),
	}
}
