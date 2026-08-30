package api

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// validateAlertConfig validates alert configuration fields
// SECURITY FIX MEDIUM-3: Comprehensive input validation to prevent injection attacks.
func validateAlertConfig(cfg AlertConfig) []ErrorDetail {
	var errs []ErrorDetail

	// Validate packet threshold (must be reasonable)
	if cfg.PacketsThreshold > 0 && cfg.PacketsThreshold > 1000000000 {
		errs = append(errs, ErrorDetail{
			Field: "packets_threshold",
			Issue: "threshold exceeds maximum allowed value of 1 billion",
			Value: strconv.FormatUint(cfg.PacketsThreshold, 10),
		})
	}

	// Validate webhook URL if provided
	// SECURITY FIX #158: Enhanced SSRF protection for webhook URLs
	if cfg.WebhookURL != "" {
		if len(cfg.WebhookURL) > maxURLLength {
			errs = append(errs, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL exceeds maximum length of 2048 characters",
				Value: "[too long]",
			})
		}
		// Basic URL format validation
		if !strings.HasPrefix(cfg.WebhookURL, "http://") &&
			!strings.HasPrefix(cfg.WebhookURL, "https://") {
			errs = append(errs, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL must start with http:// or https://",
				Value: cfg.WebhookURL[:min(truncateErrorValue, len(cfg.WebhookURL))],
			})
		}
		// SSRF protection: check for internal/private addresses
		err := validateWebhookURLSSRF(cfg.WebhookURL)
		if err != nil {
			errs = append(errs, ErrorDetail{
				Field: "webhook_url",
				Issue: err.Error(),
				Value: "[redacted]",
			})
		}
	}

	return errs
}

// normalizeAndParseIP parses an IP address string, handling various edge cases.
// SECURITY FIX #168: Handle IPv6 zone identifiers, mapped addresses, and decimal notation.
func normalizeAndParseIP(host string) net.IP {
	// Strip IPv6 zone identifier (e.g., "::1%eth0" -> "::1")
	if idx := strings.Index(host, "%"); idx != -1 {
		host = host[:idx]
	}

	// Try standard parsing first
	ip := net.ParseIP(host)
	if ip != nil {
		return ip
	}

	// Try parsing as decimal IPv4 (e.g., "2130706433" = 127.0.0.1)
	if decimal, err := strconv.ParseUint(host, 10, 32); err == nil {
		return net.IPv4(
			safeconv.ByteFromUint64(decimal>>bitShift24),
			safeconv.ByteFromUint64(decimal>>bitShift16),
			safeconv.ByteFromUint64(decimal>>bitShift8),
			safeconv.ByteFromUint64(decimal),
		)
	}

	return nil
}

// isIPv6MappedLoopback checks if an IPv6 address is a mapped IPv4 loopback.
// SECURITY FIX #168: Detect ::ffff:127.0.0.1 and similar mapped loopback addresses.
func isIPv6MappedLoopback(ip net.IP) bool {
	// Check if it's an IPv6-mapped IPv4 address
	if ip4 := ip.To4(); ip4 != nil && len(ip) == net.IPv6len {
		// This is an IPv6-mapped IPv4 address, check the IPv4 part
		return ip4.IsLoopback()
	}
	return false
}

// isBlockedHostname checks if a hostname is in the blocked list.
func isBlockedHostname(host string) bool {
	// Blocked hosts for SSRF protection.
	blockedHosts := []string{
		"localhost",
		"localhost.localdomain",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"0",
		"[::1]",
	}

	// Metadata service hosts to block.
	metadataHosts := []string{
		"169.254.169.254", // AWS/GCP/Azure metadata service
		"metadata.google.internal",
		"metadata.goog",
	}

	lowerHost := strings.ToLower(host)

	if slices.Contains(blockedHosts, lowerHost) {
		return true
	}

	// Check for short-form localhost variations
	if strings.HasPrefix(lowerHost, "127.") || lowerHost == "127" {
		return true
	}

	return slices.Contains(metadataHosts, lowerHost)
}

// isBlockedIP checks if an IP address should be blocked for SSRF.
func isBlockedIP(ip net.IP) error {
	if ip.IsLoopback() {
		return errors.New("loopback addresses not allowed")
	}

	if isIPv6MappedLoopback(ip) {
		return errors.New("loopback addresses not allowed")
	}

	if ip.IsPrivate() {
		return errors.New("private network addresses not allowed")
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return errors.New("link-local addresses not allowed")
	}

	if ip.IsUnspecified() {
		return errors.New("unspecified addresses not allowed")
	}

	// Block 169.254.0.0/16 (link-local) explicitly
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
		return errors.New("link-local addresses not allowed")
	}

	return nil
}

// isBlockedIPv6Mapped checks for IPv6-mapped private addresses.
func isBlockedIPv6Mapped(ip net.IP) error {
	ip4 := ip.To4()
	if ip4 == nil || len(ip) != net.IPv6len {
		return nil
	}

	if ip4.IsPrivate() || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
		return errors.New("private/loopback addresses not allowed (IPv6-mapped)")
	}

	return nil
}

// validateWebhookURLSSRF validates a webhook URL to prevent SSRF attacks.
// SECURITY FIX #158, #168: Prevents requests to internal/private networks.
func validateWebhookURLSSRF(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.New("invalid URL format")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return errors.New("URL must have a hostname")
	}

	if isBlockedHostname(host) {
		return errors.New("blocked hostname")
	}

	ip := normalizeAndParseIP(host)
	if ip == nil {
		return nil
	}

	if err = isBlockedIP(ip); err != nil {
		return err
	}

	return isBlockedIPv6Mapped(ip)
}

// isValidInterfaceChar checks if a character is valid in an interface name.
func isValidInterfaceChar(c rune) bool {
	isLowerLetter := c >= 'a' && c <= 'z'
	isUpperLetter := c >= 'A' && c <= 'Z'
	isDigit := c >= '0' && c <= '9'
	isAllowedSpecial := c == '-' || c == '_' || c == '.'

	return isLowerLetter || isUpperLetter || isDigit || isAllowedSpecial
}

// validateInterfaceName validates interface name characters and length.
// SECURITY FIX #182: Stricter validation to prevent path traversal and dangerous names.
func validateInterfaceName(iface string) *ErrorDetail {
	if iface == "" {
		return &ErrorDetail{
			Field: "interface",
			Issue: "interface name is required",
		}
	}

	if len(iface) > maxInterfaceNameLen {
		return &ErrorDetail{
			Field: "interface",
			Issue: fmt.Sprintf(
				"interface name exceeds %d characters (Linux IFNAMSIZ limit)",
				maxInterfaceNameLen,
			),
			Value: iface[:min(truncateErrorValue, len(iface))],
		}
	}

	// Must start with a letter (apply De Morgan's law for clarity)
	isLowerLetter := iface[0] >= 'a' && iface[0] <= 'z'
	isUpperLetter := iface[0] >= 'A' && iface[0] <= 'Z'
	if iface[0] == '-' || iface[0] == '.' || (!isLowerLetter && !isUpperLetter) {
		return &ErrorDetail{
			Field: "interface",
			Issue: "interface name must start with a letter",
			Value: iface[:min(truncateErrorValue, len(iface))],
		}
	}

	// Must not contain path traversal sequences
	if iface == ".." || strings.Contains(iface, "..") {
		return &ErrorDetail{
			Field: "interface",
			Issue: "interface name must not contain path traversal sequences",
			Value: iface[:min(truncateErrorValue, len(iface))],
		}
	}

	for _, c := range iface {
		if !isValidInterfaceChar(c) {
			return &ErrorDetail{
				Field: "interface",
				Issue: "interface name contains invalid characters (only alphanumeric, dash, underscore, dot allowed)",
				Value: iface[:min(truncateErrorValue, len(iface))],
			}
		}
	}

	return nil
}

// validateConfigPath checks for path traversal and length.
func validateConfigPath(path string) []ErrorDetail {
	if path == "" {
		return nil
	}

	var errs []ErrorDetail

	if strings.Contains(path, "..") {
		errs = append(errs, ErrorDetail{
			Field: "config_path",
			Issue: "path traversal detected (.. not allowed)",
			Value: "[redacted]",
		})
	}

	if len(path) > maxPathLength {
		errs = append(errs, ErrorDetail{
			Field: "config_path",
			Issue: "path exceeds maximum length of 4096 characters",
			Value: "[too long]",
		})
	}

	return errs
}

// validateSimulationRequest validates simulation request fields
// SECURITY FIX MEDIUM-3: Prevent injection and resource exhaustion.
func validateSimulationRequest(req SimulationRequest) []ErrorDetail {
	var errs []ErrorDetail
	if req.SessionID != "" && !ValidSessionID(req.SessionID) {
		errs = append(errs, ErrorDetail{
			Field: "sessionId",
			Issue: "session ID must contain 1 to 40 lowercase letters, numbers, or hyphens and must start and end with a letter or number",
			Value: req.SessionID,
		})
	}

	if err := validateInterfaceName(req.Interface); err != nil {
		errs = append(errs, *err)
	}

	errs = append(errs, validateConfigPath(req.ConfigPath)...)

	if req.ConfigData != "" && len(req.ConfigData) > MaxScenarioConfigSize {
		errs = append(errs, ErrorDetail{
			Field: "config_data",
			Issue: fmt.Sprintf(
				"config data exceeds maximum size of %d bytes",
				MaxScenarioConfigSize,
			),
			Value: "[too large]",
		})
	}

	return errs
}

// ValidSessionID reports whether value is a usable session identifier: 1-40
// characters of [a-z0-9-], not starting or ending with a hyphen.
//
// Exported because internal/daemon interpolates a session id into the inline
// config filename and must enforce the same rule at that sink -- the daemon's
// crash-recovery path calls StartSimulation with a request read back from disk
// and never passes through this package's validators.
func ValidSessionID(value string) bool {
	if len(value) == 0 || len(value) > 40 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

// validateReplayRequest validates replay request fields
// SECURITY FIX MEDIUM-3: Prevent PCAP injection and resource exhaustion.
func validateReplayRequest(req ReplayRequest) []ErrorDetail {
	var errs []ErrorDetail

	// Validate inline data size. Bug #1e: MaxPCAPBase64Len is a ceiling
	// (4*ceil(N/3)), unlike the previous floor-divided MaxPCAPUploadSize*4/
	// base64Ratio, which could reject an exactly-100MiB raw capture by a
	// few bytes.
	if len(req.InlineData) > MaxPCAPBase64Len {
		errs = append(errs, ErrorDetail{
			Field: "data",
			Issue: fmt.Sprintf(
				"inline data exceeds maximum size of %d bytes (100MB base64)",
				MaxPCAPBase64Len,
			),
			Value: "[too large]",
		})
	}

	// Validate file path (prevent path traversal)
	if req.File != "" {
		if strings.Contains(req.File, "..") {
			errs = append(errs, ErrorDetail{
				Field: "file",
				Issue: "path traversal detected (.. not allowed)",
				Value: "[redacted]",
			})
		}

		if len(req.File) > maxPathLength {
			errs = append(errs, ErrorDetail{
				Field: "file",
				Issue: "file path exceeds maximum length of 4096 characters",
				Value: "[too long]",
			})
		}
	}

	// Validate loop milliseconds (prevent extreme values)
	if req.LoopMs < 0 {
		errs = append(errs, ErrorDetail{
			Field: "loop_ms",
			Issue: "loop_ms cannot be negative",
			Value: strconv.Itoa(req.LoopMs),
		})
	}

	if req.LoopMs > maxLoopMs { // Max 24 hours
		errs = append(errs, ErrorDetail{
			Field: "loop_ms",
			Issue: "loop_ms exceeds maximum of 24 hours (86400000ms)",
			Value: strconv.Itoa(req.LoopMs),
		})
	}

	// Validate scale (prevent extreme values)
	if req.Scale < 0 {
		errs = append(errs, ErrorDetail{
			Field: "scale",
			Issue: "scale cannot be negative",
			Value: fmt.Sprintf("%f", req.Scale),
		})
	}

	if req.Scale > maxScaleFactor {
		errs = append(errs, ErrorDetail{
			Field: "scale",
			Issue: "scale exceeds maximum of 1000x",
			Value: fmt.Sprintf("%f", req.Scale),
		})
	}

	if req.LoopCount < 0 {
		errs = append(errs, ErrorDetail{
			Field: "loopCount",
			Issue: "loopCount cannot be negative",
			Value: strconv.Itoa(req.LoopCount),
		})
	}

	if req.LoopCount > maxReplayLoopCount {
		errs = append(errs, ErrorDetail{
			Field: "loopCount",
			Issue: "loopCount exceeds maximum of 1000000 passes",
			Value: strconv.Itoa(req.LoopCount),
		})
	}

	errs = append(errs, validateReplayRate(req)...)
	errs = append(errs, validateReplayFilter(req)...)

	return errs
}

// validateReplayFilter checks the replay BPF filter length and that it
// compiles, so the API rejects a malformed expression with a clean 400 before
// playback starts.
func validateReplayFilter(req ReplayRequest) []ErrorDetail {
	if req.BPFFilter == "" {
		return nil
	}

	if len(req.BPFFilter) > maxBPFFilterLen {
		return []ErrorDetail{{
			Field: "bpfFilter",
			Issue: "bpfFilter exceeds maximum length of 1024 characters",
			Value: "[too long]",
		}}
	}

	if err := capture.ValidateBPFExpr(req.BPFFilter); err != nil {
		return []ErrorDetail{{
			Field: "bpfFilter",
			Issue: err.Error(),
			Value: req.BPFFilter,
		}}
	}

	return nil
}

// validateReplayRate validates the rate mode and its mode-specific parameter.
// A pacing mode overrides the captured timing, so only the parameter that the
// selected mode consumes is checked.
func validateReplayRate(req ReplayRequest) []ErrorDetail {
	var errs []ErrorDetail

	switch config.RateMode(req.RateMode) {
	case "", config.RateTiming, config.RateTopspeed:
		// No rate parameter to validate.
	case config.RatePPS:
		if req.Pps <= 0 {
			errs = append(errs, ErrorDetail{
				Field: "pps",
				Issue: `pps must be greater than 0 when rateMode is "pps"`,
				Value: fmt.Sprintf("%f", req.Pps),
			})
		}
		if req.Pps > maxReplayPPS {
			errs = append(errs, ErrorDetail{
				Field: "pps",
				Issue: "pps exceeds maximum of 10000000 packets/sec",
				Value: fmt.Sprintf("%f", req.Pps),
			})
		}
	case config.RateMbps:
		if req.MbpsCap <= 0 {
			errs = append(errs, ErrorDetail{
				Field: "mbpsCap",
				Issue: `mbpsCap must be greater than 0 when rateMode is "mbps"`,
				Value: fmt.Sprintf("%f", req.MbpsCap),
			})
		}
		if req.MbpsCap > maxReplayMbps {
			errs = append(errs, ErrorDetail{
				Field: "mbpsCap",
				Issue: "mbpsCap exceeds maximum of 1000000 Mbps",
				Value: fmt.Sprintf("%f", req.MbpsCap),
			})
		}
	default:
		errs = append(errs, ErrorDetail{
			Field: "rateMode",
			Issue: "rateMode must be one of: timing, topspeed, pps, mbps",
			Value: req.RateMode,
		})
	}

	return errs
}

// validateQueryParam validates a query parameter
// SECURITY FIX MEDIUM-3: Prevent injection via query parameters.
func validateQueryParam(name, value string, allowedValues []string) *ErrorDetail {
	if value == "" {
		return nil
	}

	// Length check
	if len(value) > maxQueryParamLen {
		return &ErrorDetail{
			Field: name,
			Issue: "parameter value exceeds maximum length of 1024 characters",
			Value: value[:truncateErrorValue],
		}
	}

	// If allowed values specified, check against whitelist
	if len(allowedValues) > 0 {
		if !slices.Contains(allowedValues, value) {
			return &ErrorDetail{
				Field: name,
				Issue: fmt.Sprintf("invalid value (allowed: %v)", allowedValues),
				Value: value[:min(truncateErrorValue, len(value))],
			}
		}
	}

	return nil
}

// validateSimulationStartRequest validates required fields for simulation start.
func validateSimulationStartRequest(req SimulationRequest) []ErrorDetail {
	var errs []ErrorDetail
	if req.Interface == "" {
		errs = append(errs, ErrorDetail{Field: "interface", Issue: "interface is required"})
	}
	if req.ConfigPath == "" && req.ConfigData == "" && req.TemplateName == "" {
		errs = append(errs, ErrorDetail{
			Field: "config",
			Issue: "either config_path, config_data, or template_name must be provided",
		})
	}
	if req.AttachmentMode == fabric.ModeTrunk && req.SessionID == "" {
		errs = append(
			errs,
			ErrorDetail{Field: "sessionId", Issue: "session ID is required for trunk mode"},
		)
	}
	return errs
}

func validateSimulationForPreflight(req SimulationRequest) []ErrorDetail {
	errs := validateSimulationStartRequest(req)
	errs = append(errs, validateSimulationRequest(req)...)
	return errs
}
