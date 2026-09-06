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
	ap := newProfile("ap", "access-point", "cisco", "Wireless CW9178I", "Cisco Wireless CW9178I",
		"IOS XE 17.15.3", synth.VendorCiscoIOS, synth.TypeAccessPoint)
	ap.SysObjectID = "1.3.6.1.4.1.9.1.525"
	return []DeviceProfile{
		newProfile("distribution", "switch", "cisco", "Catalyst C9606R", "Cisco Catalyst C9606R",
			"IOS XE 17.15", synth.VendorCiscoIOS, synth.TypeSwitch),
		newProfile("access", "switch", "cisco", "Catalyst C9350-48HX", "Cisco Catalyst C9350-48HX",
			"IOS XE 17.18", synth.VendorCiscoIOS, synth.TypeSwitch),
		newProfile(
			"server-switch",
			"switch",
			"cisco",
			"Nexus 93180YC-FX3",
			"Cisco Nexus 93180YC-FX3",
			"NX-OS 10.5",
			synth.VendorCiscoIOS,
			synth.TypeSwitch,
		),
		ap,
	}
}

func endpointProfiles() []DeviceProfile {
	return []DeviceProfile{
		newProfile("workstation", "host", "dell", "OptiPlex 7020", "Dell OptiPlex 7020",
			"Windows 11 Enterprise", synth.VendorGeneric, synth.TypeHost),
		newProfile("windows-laptop", "host", "dell", "Latitude 7450", "Dell Latitude 7450",
			"Windows 11 Enterprise", synth.VendorGeneric, synth.TypeHost),
		newProfile("macbook", "host", "apple", "MacBook Pro", "Apple MacBook Pro",
			"macOS", synth.VendorGeneric, synth.TypeHost),
		newProfile("nurse-station", "host", "dell", "OptiPlex 7020 Micro", "Clinical nurse station",
			"Windows 11 Enterprise", synth.VendorGeneric, synth.TypeHost),
		newProfile("infusion-pump", "iot", "baxter", "Sigma Spectrum", "Baxter infusion pump",
			"Embedded clinical software", synth.VendorGeneric, synth.TypeHost),
		// A phone and a camera are what LLDP-MED exists to identify. Without
		// them the packs had nothing that advertised an endpoint class, so a
		// discovery tool had nothing to classify and G3 could not be seen
		// working anywhere but on an access point.
		newProfile("voip-phone", "voip-phone", "cisco", "CP-8841", "Cisco IP Phone 8841",
			"SIP88xx.12-8-1", synth.VendorCiscoIOS, synth.TypeHost),
		newProfile("ip-camera", "iot", "axis", "P3265-LVE", "Axis P3265-LVE network camera",
			"AXIS OS 11.11", synth.VendorGeneric, synth.TypeHost),
		newProfile(
			"mr-system",
			"iot",
			"siemens",
			"MAGNETOM Vida",
			"Siemens Healthineers MR imaging system",
			"Embedded clinical software",
			synth.VendorGeneric,
			synth.TypeHost,
		),
		newProfile("rugged-handheld", "host", "zebra", "TC58", "Zebra rugged mobile computer",
			"Android Enterprise", synth.VendorGeneric, synth.TypeHost),
		newProfile("barcode-printer", "printer", "zebra", "ZT411", "Zebra industrial printer",
			"Link-OS", synth.VendorGeneric, synth.TypePrinter),
		newProfile(
			"plc",
			"iot",
			"rockwell automation",
			"ControlLogix 5580",
			"Industrial controller",
			"ControlLogix firmware",
			synth.VendorGeneric,
			synth.TypeHost,
		),
		newProfile("hmi", "iot", "rockwell automation", "PanelView 5510", "Operator terminal",
			"FactoryTalk View ME", synth.VendorGeneric, synth.TypeHost),
		newProfile(
			"robot-controller",
			"iot",
			"fanuc robotics",
			"R-30iB Plus",
			"FANUC industrial robot controller",
			"Embedded controller software",
			synth.VendorGeneric,
			synth.TypeHost,
		),
		newProfile(
			"point-of-sale",
			"host",
			"hewlett packard",
			"Engage One Pro",
			"Retail point-of-sale terminal",
			"Windows 11 IoT Enterprise",
			synth.VendorGeneric,
			synth.TypeHost,
		),
		newProfile(
			"receipt-printer",
			"printer",
			"seiko epson",
			"TM-T88VII",
			"Retail receipt printer",
			"Embedded printer software",
			synth.VendorGeneric,
			synth.TypePrinter,
		),
		newProfile("digital-signage", "iot", "samsung", "QMC", "Commercial signage display",
			"Tizen", synth.VendorGeneric, synth.TypeHost),
		newProfile(
			"noc-workstation",
			"host",
			"dell",
			"Precision 3680",
			"Network operations workstation",
			"Windows 11 Enterprise",
			synth.VendorGeneric,
			synth.TypeHost,
		),
		newProfile("server", "server", "dell", "PowerEdge R660", "Dell PowerEdge R660",
			"Ubuntu Server 26.04", synth.VendorGeneric, synth.TypeServer),
		newProfile("controller", "server", "cisco", "Catalyst 9800-L", "Cisco Catalyst 9800-L",
			"IOS XE 17.15.3", synth.VendorCiscoIOS, synth.TypeRouter),
	}
}
