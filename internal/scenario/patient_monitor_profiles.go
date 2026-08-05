package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp/synth"

func patientMonitorProfiles() []DeviceProfile {
	return []DeviceProfile{philipsPatientMonitorProfile(), gePatientMonitorProfile()}
}

func philipsPatientMonitorProfile() DeviceProfile {
	return newProfile(
		"philips-patient-monitor", "iot", "philips healthcare", "IntelliVue MX850",
		"Philips IntelliVue MX850 bedside patient monitor", "Embedded clinical software",
		synth.VendorGeneric, synth.TypeHost,
	)
}

func gePatientMonitorProfile() DeviceProfile {
	return newProfile(
		"ge-patient-monitor", "iot", "ge healthcare", "CARESCAPE B850",
		"GE HealthCare CARESCAPE B850 patient monitor", "Embedded clinical software",
		synth.VendorGeneric, synth.TypeHost,
	)
}
