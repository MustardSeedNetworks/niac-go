package protocols

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// AddRecord adds a DNS A/AAAA record.
func (h *DNSHandler) AddRecord(hostname string, ip net.IP) {
	h.AddRecordWithTTL(hostname, ip, dnsDefaultTTL, layers.DNSResponseCodeNoErr)
}

// AddRecordWithTTL adds a DNS A/AAAA record with TTL and response code.
func (h *DNSHandler) AddRecordWithTTL(
	hostname string,
	ip net.IP,
	ttl uint32,
	rcode layers.DNSResponseCode,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Normalize hostname
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Add forward record
	h.records[hostname] = append(h.records[hostname], dnsRecord{
		ip:    ip,
		ttl:   ttl,
		rcode: rcode,
	})

	// Add reverse record (PTR)
	h.ptrRecords[ip.String()] = dnsPTR{
		name:  hostname,
		ttl:   ttl,
		rcode: rcode,
	}
}

// AddPTRRecord adds a PTR record with TTL and response code.
func (h *DNSHandler) AddPTRRecord(
	ip net.IP,
	hostname string,
	ttl uint32,
	rcode layers.DNSResponseCode,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))
	h.ptrRecords[ip.String()] = dnsPTR{
		name:  hostname,
		ttl:   ttl,
		rcode: rcode,
	}
}

// SetDomain sets the default DNS domain.
func (h *DNSHandler) SetDomain(domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.domain = domain
}

// LoadDeviceRecords loads DNS records from configured devices.
func (h *DNSHandler) LoadDeviceRecords(devices []*config.Device) {
	for _, device := range devices {
		hostname := device.SNMPConfig.SysName
		if hostname == "" {
			hostname = device.Name
		}

		for _, ip := range device.IPAddresses {
			h.AddRecord(hostname, ip)
		}
	}
}

// LoadDeviceDNSConfig loads device-specific DNS records.
func (h *DNSHandler) LoadDeviceDNSConfig(device *config.Device) {
	if device == nil || device.DNSConfig == nil {
		return
	}

	set := &dnsRecordSet{
		forward: make(map[string][]dnsRecord),
		reverse: make(map[string]dnsPTR),
	}

	for _, rec := range device.DNSConfig.ForwardRecords {
		name := strings.ToLower(strings.TrimSuffix(rec.Name, "."))
		set.forward[name] = append(set.forward[name], dnsRecord{
			ip:    rec.IP,
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(safeconv.DNSRCode(rec.RCode)),
		})
	}

	for _, rec := range device.DNSConfig.ReverseRecords {
		set.reverse[rec.IP.String()] = dnsPTR{
			name:  strings.ToLower(strings.TrimSuffix(rec.Name, ".")),
			ttl:   rec.TTL,
			rcode: layers.DNSResponseCode(safeconv.DNSRCode(rec.RCode)),
		}
	}

	h.mu.Lock()
	h.deviceRecords[device] = set
	h.mu.Unlock()
}

// LookupHost looks up IP addresses for a hostname (exported for testing).
func (h *DNSHandler) LookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	return h.lookupHost(hostname, set)
}

// lookupHost looks up IP addresses for a hostname.
func (h *DNSHandler) lookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Device-specific records
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

	// Global records
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

// ResolveQuestions resolves DNS questions (exported for testing).
func (h *DNSHandler) ResolveQuestions(
	questions []layers.DNSQuestion,
	set *dnsRecordSet,
	debugLevel int,
	serial int,
) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	return h.resolveQuestions(questions, set, debugLevel, serial)
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
