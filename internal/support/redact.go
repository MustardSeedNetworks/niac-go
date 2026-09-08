package support

import (
	"regexp"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Placeholder replaces every credential a bundle would otherwise carry. It is
// a single fixed string rather than a length-preserving mask so a reader can
// see at a glance that a value was removed, not merely obscured.
const Placeholder = "[REDACTED]"

// Redact zeroes every credential a scenario carries, in place.
//
// Redaction is structural, over the typed fields, not textual: a scan for
// "public" would both miss an operator's real community string and rewrite
// every unrelated occurrence of the word.
func Redact(cfg *config.Config) {
	if cfg == nil {
		return
	}
	for i := range cfg.Devices {
		redactDevice(&cfg.Devices[i])
	}
}

func redactDevice(device *config.Device) {
	redactSNMP(&device.SNMPConfig)
	if device.SNMPv3Config != nil {
		for i := range device.SNMPv3Config.Users {
			user := &device.SNMPv3Config.Users[i]
			redactIfSet(&user.AuthPassword)
			redactIfSet(&user.PrivPassword)
		}
	}
	if device.FTPConfig != nil {
		for i := range device.FTPConfig.Users {
			redactIfSet(&device.FTPConfig.Users[i].Password)
		}
	}
}

func redactSNMP(snmp *config.SNMPConfig) {
	redactIfSet(&snmp.Community)
	for i := range snmp.CommunityIncludes {
		redactIfSet(&snmp.CommunityIncludes[i].Community)
	}
	if snmp.Traps != nil {
		redactIfSet(&snmp.Traps.Community)
	}
}

// redactIfSet leaves an unset field unset: writing a placeholder into a field
// the operator never configured would make the bundle claim a credential
// exists where none does.
func redactIfSet(value *string) {
	if *value != "" {
		*value = Placeholder
	}
}

// Secrets returns every credential value a scenario carries, so free text the
// structural redactor cannot reach -- log lines, command output -- can be
// scrubbed of the same values.
func Secrets(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	appendIfSet := func(value string) {
		if value != "" {
			out = append(out, value)
		}
	}
	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		appendIfSet(device.SNMPConfig.Community)
		for _, include := range device.SNMPConfig.CommunityIncludes {
			appendIfSet(include.Community)
		}
		if device.SNMPConfig.Traps != nil {
			appendIfSet(device.SNMPConfig.Traps.Community)
		}
		if device.SNMPv3Config != nil {
			for _, user := range device.SNMPv3Config.Users {
				appendIfSet(user.AuthPassword)
				appendIfSet(user.PrivPassword)
			}
		}
		if device.FTPConfig != nil {
			for _, user := range device.FTPConfig.Users {
				appendIfSet(user.Password)
			}
		}
	}
	return out
}

// bearerPattern and credentialAssignment catch the two shapes a secret takes
// in free text when its value is not one the caller knew to pass: an
// Authorization header, and a key=value or "key": "value" pair whose key names
// a credential.
var (
	bearerPattern = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`)

	credentialAssignment = regexp.MustCompile(
		`(?i)\b(token|community|password|passphrase|secret|api[_-]?key)\b(\s*[:=]\s*)"?[^"\s,}]+"?`)
)

// ScrubText removes known secret values from free text and masks anything
// still shaped like a credential.
//
// The known values come first because they are exact: a community string that
// happens to look like a word is removed by value, not by pattern. The
// patterns are the backstop for a secret this package was never told about.
func ScrubText(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, Placeholder)
	}
	text = bearerPattern.ReplaceAllString(text, "Bearer "+Placeholder)
	return credentialAssignment.ReplaceAllString(text, "$1$2"+Placeholder)
}
