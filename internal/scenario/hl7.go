package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// packHL7 opens HL7 on a hospital's clinical systems. The EMR, the laboratory
// system and the pharmacy system each receive ADT, order and result messages
// over MLLP, so an interface engine pointed at any of them gets an ACK.
func packHL7(profile string) *converter.HL7Config {
	if profile != "hospital" {
		return nil
	}

	return &converter.HL7Config{Enabled: true}
}
