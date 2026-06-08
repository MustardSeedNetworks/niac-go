package protocols

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

type dnsResolveContext struct {
	answers      []layers.DNSResourceRecord
	responseCode layers.DNSResponseCode
	debugLevel   int
	serial       int
}

func (h *DNSHandler) resolveQuestions(
	questions []layers.DNSQuestion,
	set *dnsRecordSet,
	debugLevel int,
	serial int,
) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	ctx := &dnsResolveContext{
		answers:      make([]layers.DNSResourceRecord, 0, len(questions)),
		responseCode: layers.DNSResponseCodeNoErr,
		debugLevel:   debugLevel,
		serial:       serial,
	}

	for _, q := range questions {
		if !h.validateAndResolveQuestion(q, set, ctx) {
			continue
		}
	}

	if len(ctx.answers) == 0 && ctx.responseCode == layers.DNSResponseCodeNoErr {
		ctx.responseCode = layers.DNSResponseCodeNXDomain
	}

	return ctx.answers, ctx.responseCode
}

// validateAndResolveQuestion validates and resolves a single DNS question.
// Returns false if the question should be skipped.
func (h *DNSHandler) validateAndResolveQuestion(
	q layers.DNSQuestion,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) bool {
	if !isValidDNSName(q.Name) {
		h.logInvalidDNSName(q.Name, ctx)
		return false
	}

	hostname := strings.ToLower(strings.TrimSuffix(string(q.Name), "."))
	h.resolveByType(q, hostname, set, ctx)

	return true
}

// resolveByType dispatches to the appropriate resolver based on DNS query type.
func (h *DNSHandler) resolveByType(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	//exhaustive:ignore
	switch q.Type {
	case layers.DNSTypeA:
		h.resolveARecord(q, hostname, set, ctx)
	case layers.DNSTypeAAAA:
		h.resolveAAAARecord(q, hostname, set, ctx)
	case layers.DNSTypePTR:
		h.resolvePTRRecord(q, set, ctx)
	case layers.DNSType(dnsTypeNBSTAT):
		// NBSTAT handled separately in HandleQuery.
	}
}

// lookupHost looks up IP addresses for a hostname, falling back to the
// default domain for single-label lookups. Used by A/AAAA resolution.
func (h *DNSHandler) lookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	if set != nil {
		if recs, ok := set.forward[hostname]; ok {
			return recs
		}

		if !strings.Contains(hostname, ".") {
			fullname := hostname + "." + h.domain
			if recs, ok := set.forward[fullname]; ok {
				return recs
			}
		}

		return nil
	}

	if recs, ok := h.records[hostname]; ok {
		return recs
	}

	if !strings.Contains(hostname, ".") {
		fullname := hostname + "." + h.domain
		if recs, ok := h.records[fullname]; ok {
			return recs
		}
	}

	return nil
}

// resolveARecord resolves DNS A record queries.
func (h *DNSHandler) resolveARecord(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	for _, rec := range h.lookupHost(hostname, set) {
		if rec.ip.To4() == nil {
			continue
		}

		ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
			Name:  q.Name,
			Type:  layers.DNSTypeA,
			Class: layers.DNSClassIN,
			TTL:   rec.ttl,
			IP:    rec.ip,
		})

		if rec.rcode != layers.DNSResponseCodeNoErr {
			ctx.responseCode = rec.rcode
		}

		h.logDNSRecord("A", hostname, rec.ip.String(), ctx)
	}
}

// resolveAAAARecord resolves DNS AAAA record queries.
func (h *DNSHandler) resolveAAAARecord(
	q layers.DNSQuestion,
	hostname string,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	for _, rec := range h.lookupHost(hostname, set) {
		if rec.ip.To4() != nil || rec.ip.To16() == nil {
			continue
		}

		ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
			Name:  q.Name,
			Type:  layers.DNSTypeAAAA,
			Class: layers.DNSClassIN,
			TTL:   rec.ttl,
			IP:    rec.ip,
		})

		if rec.rcode != layers.DNSResponseCodeNoErr {
			ctx.responseCode = rec.rcode
		}

		h.logDNSRecord("AAAA", hostname, rec.ip.String(), ctx)
	}
}

// resolvePTRRecord resolves DNS PTR record queries.
func (h *DNSHandler) resolvePTRRecord(
	q layers.DNSQuestion,
	set *dnsRecordSet,
	ctx *dnsResolveContext,
) {
	ip, ok, isV6 := parsePTRName(q.Name)
	if !ok {
		h.logPTRParseFailure(q.Name, ctx)
		return
	}

	host, found := h.lookupPTR(ip, set)
	if !found || host.name == "" {
		if isV6 {
			ctx.responseCode = layers.DNSResponseCodeNXDomain
		}

		return
	}

	ptr := host.name
	if !strings.HasSuffix(ptr, ".") {
		ptr += "."
	}

	ctx.answers = append(ctx.answers, layers.DNSResourceRecord{
		Name:  q.Name,
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
		TTL:   host.ttl,
		PTR:   []byte(ptr),
	})

	if host.rcode != layers.DNSResponseCodeNoErr {
		ctx.responseCode = host.rcode
	}

	h.logDNSRecord("PTR", string(q.Name), ptr, ctx)
}

// logInvalidDNSName logs an invalid DNS name error.
func (h *DNSHandler) logInvalidDNSName(name []byte, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: Invalid domain name length (> 255 or label > 63): %s sn=%d\n",
			name,
			ctx.serial,
		)
	}
}

// logDNSRecord logs a resolved DNS record.
func (h *DNSHandler) logDNSRecord(recordType, name, value string, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: %s -> %s (%s record) sn=%d\n",
			name,
			value,
			recordType,
			ctx.serial,
		)
	}
}

// logPTRParseFailure logs a PTR query parse failure.
func (h *DNSHandler) logPTRParseFailure(name []byte, ctx *dnsResolveContext) {
	if ctx.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"DNS: PTR query %s could not be parsed sn=%d\n",
			name,
			ctx.serial,
		)
	}
}

func (h *DNSHandler) selectServerDevice(
	devices []*config.Device,
	wantIPv6 bool,
) (*config.Device, net.IP) {
	for _, dev := range devices {
		if !h.deviceHasDNSRecords(dev) {
			continue
		}

		ip := pickIPAddressForDNS(dev, wantIPv6)
		if ip == nil {
			continue
		}

		if len(dev.MACAddress) == 0 {
			continue
		}

		return dev, ip
	}

	return nil, nil
}

func (h *DNSHandler) deviceHasDNSRecords(dev *config.Device) bool {
	if dev == nil {
		return false
	}

	h.mu.RLock()
	_, hasSet := h.deviceRecords[dev]
	hasGlobal := len(h.records) > 0
	h.mu.RUnlock()

	if hasSet {
		return true
	}

	if dev.DNSConfig != nil {
		return true
	}
	// If no device-specific record set exists, fall back to global records
	return dev.DNSConfig == nil && hasGlobal
}

func (h *DNSHandler) getRecordSetForDevice(dev *config.Device) *dnsRecordSet {
	if dev == nil {
		return nil
	}

	h.mu.RLock()
	set := h.deviceRecords[dev]
	h.mu.RUnlock()

	return set
}

func pickIPAddressForDNS(device *config.Device, wantIPv6 bool) net.IP {
	for _, ip := range device.IPAddresses {
		if wantIPv6 {
			if ip.To4() == nil && ip.To16() != nil {
				return ip
			}
		} else if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

func (h *DNSHandler) lookupPTR(ip net.IP, set *dnsRecordSet) (dnsPTR, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if set != nil {
		rec, ok := set.reverse[ip.String()]

		return rec, ok
	}

	rec, ok := h.ptrRecords[ip.String()]

	return rec, ok
}

func parsePTRName(name []byte) (net.IP, bool, bool) {
	ptrName := strings.ToLower(strings.TrimSuffix(string(name), "."))

	switch {
	case strings.HasSuffix(ptrName, ".in-addr.arpa"):
		ip, ok := parseIPv4PTRName(ptrName)

		return ip, ok, false
	case strings.HasSuffix(ptrName, ".ip6.arpa"):
		ip, ok := parseIPv6PTRName(ptrName)

		return ip, ok, true
	default:
		return nil, false, false
	}
}

func parseIPv4PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".in-addr.arpa")

	parts := strings.Split(strings.Trim(base, "."), ".")
	if len(parts) != dnsIPv4Octets {
		return nil, false
	}

	ip := net.IPv4(0, 0, 0, 0).To4()

	for i := range dnsIPv4Octets {
		val, err := strconv.Atoi(parts[len(parts)-1-i])
		if err != nil || val < 0 || val > dnsMaxByteValue {
			return nil, false
		}

		ip[i] = byte(val)
	}

	return ip, true
}

func parseIPv6PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".ip6.arpa")

	nibbles := strings.Split(strings.Trim(base, "."), ".")
	if len(nibbles) != dnsIPv6NibbleLen {
		return nil, false
	}

	var builder strings.Builder

	builder.Grow(dnsIPv6NibbleLen)

	for _, v := range slices.Backward(nibbles) {
		if len(v) != 1 {
			return nil, false
		}

		builder.WriteString(v)
	}

	data, err := hex.DecodeString(builder.String())
	if err != nil || len(data) != net.IPv6len {
		return nil, false
	}

	return net.IP(data), true
}

func isValidDNSName(name []byte) bool {
	// RFC 1035: Maximum domain name length is 255 bytes
	if len(name) > dnsMaxNameLen {
		return false
	}

	// Validate individual label lengths (max 63 bytes per label)
	labels := strings.SplitSeq(string(name), ".")
	for label := range labels {
		if len(label) > dnsMaxLabelLen {
			return false
		}
	}

	return true
}
