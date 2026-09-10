package config

import "strings"

// SNMPv2Enabled reports whether an authored device serves a community.
func SNMPv2Enabled(cfg SNMPConfig) bool {
	return (cfg.Enabled == nil || *cfg.Enabled) && strings.TrimSpace(cfg.Community) != ""
}

// SNMPv3Enabled reports whether an authored device serves configured USM users.
func SNMPv3Enabled(cfg *SNMPv3Config) bool {
	return cfg != nil && cfg.Enabled && len(cfg.Users) > 0
}
