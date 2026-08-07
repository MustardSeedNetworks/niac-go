package protocols

import (
	"fmt"
	"os"
	"strings"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

const (
	// MDNSPort carries multicast DNS.
	MDNSPort = 5353
	// mdnsDomain suffixes every multicast DNS name.
	mdnsDomain = ".local"
	// serviceEnumerationName lists the service types a host offers.
	serviceEnumerationName = "_services._dns-sd._udp.local"
	// mdnsCacheFlush marks a record as authoritative, replacing any cached
	// copy rather than adding to it.
	mdnsCacheFlush = 0x8000
)

// MDNSHandler answers multicast DNS queries for simulated devices.
//
// Apple hardware, printers and most IoT gear announce themselves this way
// rather than through NetBIOS, so without it those devices are unnamed on a
// discovery map even when they answer everything else.
type MDNSHandler struct {
	stack      *Stack
	debugLevel int
}

// NewMDNSHandler creates a multicast DNS responder.
func NewMDNSHandler(stack *Stack, debugLevel int) *MDNSHandler {
	return &MDNSHandler{stack: stack, debugLevel: debugLevel}
}

// SetDebugLevel adjusts responder logging.
func (h *MDNSHandler) SetDebugLevel(level int) {
	h.debugLevel = level
}

// HandleQuery answers a multicast DNS query.
//
// Queries arrive addressed to the multicast group, so the responding device
// cannot be identified from the destination address the way a unicast service
// identifies it. Each advertising device is matched against the question
// instead, and every device holding a matching name answers.
func (h *MDNSHandler) HandleQuery(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udp *layers.UDP,
	devices []*config.Device,
	packet gopacket.Packet,
) {
	var query layers.DNS
	if err := query.DecodeFromBytes(udp.Payload, gopacket.NilDecodeFeedback); err != nil {
		return
	}
	if query.QR || len(query.Questions) == 0 {
		return
	}

	eth, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if eth == nil {
		return
	}

	for _, device := range devices {
		if device == nil || device.MDNSConfig == nil || !device.MDNSConfig.Enabled {
			continue
		}
		answers := mdnsAnswers(device, query.Questions)
		if len(answers) == 0 {
			continue
		}
		h.respond(pkt, ipLayer, udp, eth, device, query.ID, answers)
	}
}

// respond sends one device's answers back to the asking host.
//
// The reply is unicast to the querier rather than multicast: it reaches the
// tool that asked without every other listener on the segment having to
// process it, and a querier accepts either.
func (h *MDNSHandler) respond(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udp *layers.UDP,
	eth *layers.Ethernet,
	device *config.Device,
	id uint16,
	answers []layers.DNSResourceRecord,
) {
	response := &layers.DNS{
		ID: id, QR: true, AA: true,
		ANCount: safeconv.Uint16(len(answers)),
		Answers: answers,
	}

	buf := gopacket.NewSerializeBuffer()
	if err := response.SerializeTo(buf, gopacket.SerializeOptions{FixLengths: true}); err != nil {
		return
	}

	deviceIP := getFirstIPv4(device.IPAddresses)
	if deviceIP == nil {
		return
	}

	err := h.stack.udpHandler.SendUDP(
		deviceIP.To4(), ipLayer.SrcIP.To4(),
		MDNSPort, uint16(udp.SrcPort),
		buf.Bytes(), device.MACAddress, eth.SrcMAC, pkt.VLAN,
	)
	if err != nil {
		return
	}

	if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "mDNS: %s answered %d record(s) sn=%d\n",
			device.Name, len(answers), pkt.SerialNumber)
	}
}

// mdnsAnswers builds the records a device owes for the questions asked.
func mdnsAnswers(device *config.Device, questions []layers.DNSQuestion) []layers.DNSResourceRecord {
	cfg := device.MDNSConfig
	host := mdnsHostname(cfg)
	deviceIP := getFirstIPv4(device.IPAddresses)
	if host == "" || deviceIP == nil {
		return nil
	}

	var answers []layers.DNSResourceRecord
	for _, question := range questions {
		name := strings.ToLower(strings.TrimSuffix(string(question.Name), "."))
		switch {
		case question.Type == layers.DNSTypeA && name == host:
			answers = append(answers, mdnsRecord(host, layers.DNSTypeA, cfg.TTL, func(r *layers.DNSResourceRecord) {
				r.IP = deviceIP.To4()
			}))
		case question.Type == layers.DNSTypePTR && name == serviceEnumerationName:
			answers = append(answers, mdnsServiceEnumeration(cfg)...)
		case question.Type == layers.DNSTypePTR:
			answers = append(answers, mdnsServiceInstances(cfg, host, name)...)
		}
	}

	return answers
}

// mdnsServiceEnumeration lists the service types this device offers, the reply
// to the "what services exist here" question a browser asks first.
func mdnsServiceEnumeration(cfg *config.MDNSConfig) []layers.DNSResourceRecord {
	records := make([]layers.DNSResourceRecord, 0, len(cfg.Services))
	for _, svc := range cfg.Services {
		serviceType := mdnsServiceType(svc.Type)
		records = append(records, mdnsRecord(serviceEnumerationName, layers.DNSTypePTR, cfg.TTL,
			func(r *layers.DNSResourceRecord) { r.PTR = []byte(serviceType) }))
	}

	return records
}

// mdnsServiceInstances answers a browse for one service type with this
// device's instance of it.
func mdnsServiceInstances(
	cfg *config.MDNSConfig,
	host, question string,
) []layers.DNSResourceRecord {
	var records []layers.DNSResourceRecord
	for _, svc := range cfg.Services {
		serviceType := mdnsServiceType(svc.Type)
		if question != serviceType {
			continue
		}
		instance := mdnsInstanceName(cfg, serviceType)
		records = append(records,
			mdnsRecord(serviceType, layers.DNSTypePTR, cfg.TTL,
				func(r *layers.DNSResourceRecord) { r.PTR = []byte(instance) }),
			mdnsRecord(instance, layers.DNSTypeSRV, cfg.TTL,
				func(r *layers.DNSResourceRecord) {
					r.SRV = layers.DNSSRV{Port: svc.Port, Name: []byte(host)}
				}),
			mdnsRecord(instance, layers.DNSTypeTXT, cfg.TTL,
				func(r *layers.DNSResourceRecord) { r.TXTs = mdnsTXT(svc.TXT) }),
		)
	}

	return records
}

func mdnsRecord(
	name string,
	recordType layers.DNSType,
	ttl uint32,
	fill func(*layers.DNSResourceRecord),
) layers.DNSResourceRecord {
	record := layers.DNSResourceRecord{
		Name:  []byte(name),
		Type:  recordType,
		Class: layers.DNSClass(mdnsCacheFlush | uint16(layers.DNSClassIN)),
		TTL:   ttl,
	}
	fill(&record)

	return record
}

func mdnsHostname(cfg *config.MDNSConfig) string {
	name := strings.ToLower(strings.TrimSpace(cfg.Hostname))
	if name == "" {
		return ""
	}
	if strings.HasSuffix(name, mdnsDomain) {
		return name
	}

	return name + mdnsDomain
}

// mdnsServiceType normalises "_ipp._tcp" into a fully qualified service name.
func mdnsServiceType(serviceType string) string {
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(serviceType), "."))
	if name == "" {
		return ""
	}
	if strings.HasSuffix(name, mdnsDomain) {
		return name
	}

	return name + mdnsDomain
}

// mdnsInstanceName labels this device's instance of a service, the
// "<instance>.<service>.local" form a browser displays.
func mdnsInstanceName(cfg *config.MDNSConfig, serviceType string) string {
	label := strings.TrimSuffix(strings.ToLower(cfg.Hostname), mdnsDomain)

	return label + "." + serviceType
}

func mdnsTXT(entries []string) [][]byte {
	if len(entries) == 0 {
		return [][]byte{[]byte("")}
	}
	txt := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		txt = append(txt, []byte(entry))
	}

	return txt
}
