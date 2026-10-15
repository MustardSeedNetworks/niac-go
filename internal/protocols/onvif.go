package protocols

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// UDPPortWSDiscovery is where a WS-Discovery target service listens.
const UDPPortWSDiscovery = 3702

// The namespaces of ONVIF's WS-Discovery profile: SOAP 1.2, the 2004/08
// addressing and 2005/04 discovery drafts ONVIF mandates, and the two types an
// ONVIF video transmitter advertises.
const (
	wsdSOAPNamespace = "http://www.w3.org/2003/05/soap-envelope"
	wsdAddressing    = "http://schemas.xmlsoap.org/ws/2004/08/addressing"
	wsdDiscovery     = "http://schemas.xmlsoap.org/ws/2005/04/discovery"
	wsdNetworkWSDL   = "http://www.onvif.org/ver10/network/wsdl"
	wsdDeviceWSDL    = "http://www.onvif.org/ver10/device/wsdl"
	wsdProbeAction   = wsdDiscovery + "/Probe"
	wsdMatchRFC3986  = wsdDiscovery + "/rfc3986"
	wsdMatchStrcmp0  = wsdDiscovery + "/strcmp0"
	wsdOnvifScope    = "onvif://www.onvif.org/"
	wsdMaxProbeBytes = 16 << 10
)

// RFC 9562 layout: 16 bytes, the version in the high nibble of byte 6, the
// variant in the top two bits of byte 8, and a 6-byte node field.
const (
	uuidBytes       = 16
	uuidVersionMask = 0x0f
	uuidVariantMask = 0x3f
	uuidVariant     = 0x80
	uuidVersion4    = 0x40
	uuidVersion8    = 0x80
	uuidNodeOffset  = 10
	uuidMACBytes    = 6
)

// wsdProbe is the part of a Probe a ProbeMatch depends on.
type wsdProbe struct {
	messageID string
	types     [][2]string
	scopes    []string
	matchBy   string
}

// handleONVIF answers a Probe sent to an ONVIF-enabled device and reports
// whether one owned the destination. A false return leaves the datagram to the
// closed-port path, so 3702 on every other device behaves as before.
func (h *UDPHandler) handleONVIF(pkt *Packet, ipLayer *layers.IPv4, udp *layers.UDP, devices []*config.Device) bool {
	device := h.onvifDeviceForIP(devices, ipLayer.DstIP)
	if device == nil {
		return false
	}

	response, ok := onvifProbeMatch(udp.Payload, device, ipLayer.DstIP.To4(), h.wsdMessages.Add(1))
	if !ok {
		return true
	}
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return true
	}
	err := h.SendUDP(
		device,
		ipLayer.DstIP.To4(), ipLayer.SrcIP.To4(),
		UDPPortWSDiscovery, uint16(udp.SrcPort),
		response,
		identity.source, identity.destination, identity.vlan,
	)
	if err != nil && h.stack.GetDebugLevel() >= DebugLevelInfo {
		logging.Debugf("ONVIF: ProbeMatch from %s failed sn=%d err=%v", device.Name, pkt.SerialNumber, err)
	}

	return true
}

func (h *UDPHandler) onvifDeviceForIP(devices []*config.Device, ip net.IP) *config.Device {
	for _, device := range devices {
		if device.ONVIFConfig != nil && device.ONVIFConfig.Enabled && h.stack.deviceOwnsIPv4(device, ip) {
			return device
		}
	}

	return nil
}

// onvifProbeMatch builds the ProbeMatches answering payload from device at
// address, numbered messageNumber in its AppSequence. A message that is not a Probe, or a Probe whose types or scopes
// the camera does not satisfy, gets no reply (WS-Discovery 2005/04 §5.3).
func onvifProbeMatch(payload []byte, device *config.Device, address net.IP, messageNumber uint64) ([]byte, bool) {
	if len(payload) > wsdMaxProbeBytes {
		return nil, false
	}
	probe, ok := parseWSDProbe(payload)
	if !ok {
		return nil, false
	}
	scopes := onvifScopes(device.ONVIFConfig)
	if !wsdTypesMatch(probe.types) || !wsdScopesMatch(probe.scopes, probe.matchBy, scopes) {
		return nil, false
	}

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString(`<SOAP-ENV:Envelope xmlns:SOAP-ENV="` + wsdSOAPNamespace + `" xmlns:wsa="` + wsdAddressing +
		`" xmlns:d="` + wsdDiscovery + `" xmlns:dn="` + wsdNetworkWSDL + `" xmlns:tds="` + wsdDeviceWSDL + `">`)
	body.WriteString(`<SOAP-ENV:Header>`)
	body.WriteString(`<wsa:MessageID>urn:uuid:` + wsdRandomUUID() + `</wsa:MessageID>`)
	body.WriteString(`<wsa:RelatesTo>` + xmlText(probe.messageID) + `</wsa:RelatesTo>`)
	body.WriteString(`<wsa:To>` + wsdAddressing + `/role/anonymous</wsa:To>`)
	body.WriteString(`<wsa:Action>` + wsdDiscovery + `/ProbeMatches</wsa:Action>`)
	fmt.Fprintf(&body, `<d:AppSequence InstanceId="1" MessageNumber="%d"/>`, messageNumber)
	body.WriteString(`</SOAP-ENV:Header>`)
	body.WriteString(`<SOAP-ENV:Body><d:ProbeMatches><d:ProbeMatch>`)
	body.WriteString(`<wsa:EndpointReference><wsa:Address>urn:uuid:` + onvifEndpointUUID(device) +
		`</wsa:Address></wsa:EndpointReference>`)
	body.WriteString(`<d:Types>dn:NetworkVideoTransmitter tds:Device</d:Types>`)
	body.WriteString(`<d:Scopes>` + xmlText(strings.Join(scopes, " ")) + `</d:Scopes>`)
	body.WriteString(`<d:XAddrs>http://` + address.String() + `/onvif/device_service</d:XAddrs>`)
	body.WriteString(`<d:MetadataVersion>1</d:MetadataVersion>`)
	body.WriteString(`</d:ProbeMatch></d:ProbeMatches></SOAP-ENV:Body></SOAP-ENV:Envelope>`)

	return []byte(body.String()), true
}

// onvifScopes are the scopes the camera is listed by: the Profile S baseline,
// then its authored name and model.
func onvifScopes(cfg *config.ONVIFConfig) []string {
	return []string{
		wsdOnvifScope + "type/video_encoder",
		wsdOnvifScope + "Profile/Streaming",
		wsdOnvifScope + "hardware/" + url.PathEscape(cfg.Hardware),
		wsdOnvifScope + "name/" + url.PathEscape(cfg.Name),
	}
}

// The element paths a Probe is read from, each step written {namespace}local.
const (
	wsdEnvelopePath  = "{" + wsdSOAPNamespace + "}Envelope"
	wsdHeaderPath    = wsdEnvelopePath + "/{" + wsdSOAPNamespace + "}Header"
	wsdMessageIDPath = wsdHeaderPath + "/{" + wsdAddressing + "}MessageID"
	wsdActionPath    = wsdHeaderPath + "/{" + wsdAddressing + "}Action"
	wsdProbePath     = wsdEnvelopePath + "/{" + wsdSOAPNamespace + "}Body/{" + wsdDiscovery + "}Probe"
	wsdTypesPath     = wsdProbePath + "/{" + wsdDiscovery + "}Types"
	wsdScopesPath    = wsdProbePath + "/{" + wsdDiscovery + "}Scopes"
)

// wsdParser walks a Probe's tokens, tracking the element path and the
// namespace declarations in scope.
type wsdParser struct {
	probe    wsdProbe
	action   string
	path     []string
	scopes   []map[string]string
	text     strings.Builder
	sawProbe bool
}

// parseWSDProbe reads a SOAP 1.2 Probe. QNames in Types are resolved against
// the namespace declarations in scope, so dn:NetworkVideoTransmitter matches
// whatever prefix the client bound. A Probe without a MessageID cannot be
// answered: the ProbeMatch has nothing to relate to.
func parseWSDProbe(payload []byte) (wsdProbe, bool) {
	decoder := xml.NewDecoder(strings.NewReader(string(payload)))
	var parser wsdParser
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return wsdProbe{}, false
		}
		switch element := token.(type) {
		case xml.StartElement:
			parser.start(element)
		case xml.CharData:
			parser.text.Write(element)
		case xml.EndElement:
			if !parser.end() {
				return wsdProbe{}, false
			}
		}
	}

	if !parser.sawProbe || parser.action != wsdProbeAction || parser.probe.messageID == "" {
		return wsdProbe{}, false
	}

	return parser.probe, true
}

func (p *wsdParser) start(element xml.StartElement) {
	p.scopes = append(p.scopes, wsdNamespaceScope(p.scopes, element.Attr))
	p.path = append(p.path, "{"+element.Name.Space+"}"+element.Name.Local)
	p.text.Reset()
	switch strings.Join(p.path, "/") {
	case wsdProbePath:
		p.sawProbe = true
	case wsdScopesPath:
		for _, attr := range element.Attr {
			if attr.Name == (xml.Name{Local: "MatchBy"}) {
				p.probe.matchBy = strings.TrimSpace(attr.Value)
			}
		}
	}
}

// end closes the current element, recording its text when the Probe needs it.
// It reports false for a Types entry that names an unbound prefix.
func (p *wsdParser) end() bool {
	value := strings.TrimSpace(p.text.String())
	switch strings.Join(p.path, "/") {
	case wsdMessageIDPath:
		p.probe.messageID = value
	case wsdActionPath:
		p.action = value
	case wsdTypesPath:
		for qname := range strings.FieldsSeq(value) {
			resolved, ok := wsdResolveQName(p.scopes[len(p.scopes)-1], qname)
			if !ok {
				return false
			}
			p.probe.types = append(p.probe.types, resolved)
		}
	case wsdScopesPath:
		p.probe.scopes = strings.Fields(value)
	}
	p.path = p.path[:len(p.path)-1]
	p.scopes = p.scopes[:len(p.scopes)-1]
	p.text.Reset()

	return true
}

// wsdNamespaceScope is the prefix-to-namespace map in force inside an element
// carrying attrs, given the scopes of its ancestors.
func wsdNamespaceScope(scopes []map[string]string, attrs []xml.Attr) map[string]string {
	inherited := map[string]string{}
	if len(scopes) > 0 {
		inherited = scopes[len(scopes)-1]
	}
	scope := inherited
	copied := false
	for _, attr := range attrs {
		prefix, declares := "", false
		switch {
		case attr.Name.Space == "xmlns":
			prefix, declares = attr.Name.Local, true
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			declares = true
		}
		if !declares {
			continue
		}
		if !copied {
			scope, copied = maps.Clone(inherited), true
		}
		scope[prefix] = attr.Value
	}

	return scope
}

func wsdResolveQName(scope map[string]string, qname string) ([2]string, bool) {
	prefix, local, found := strings.Cut(qname, ":")
	if !found {
		prefix, local = "", qname
	}
	namespace, bound := scope[prefix]
	if !bound || local == "" {
		return [2]string{}, false
	}

	return [2]string{namespace, local}, true
}

// wsdTypesMatch reports whether the camera is every type the Probe asked for.
// A Probe naming no type matches any target.
func wsdTypesMatch(requested [][2]string) bool {
	types := [][2]string{
		{wsdNetworkWSDL, "NetworkVideoTransmitter"},
		{wsdDeviceWSDL, "Device"},
	}
	for _, want := range requested {
		if !slices.Contains(types, want) {
			return false
		}
	}

	return true
}

// wsdScopesMatch reports whether every requested scope matches one the camera
// carries under matchBy. The default rule is RFC 3986 segment-prefix matching;
// an unknown rule matches nothing, as the specification requires.
func wsdScopesMatch(requested []string, matchBy string, scopes []string) bool {
	var matches func(have, want string) bool
	switch matchBy {
	case "", wsdMatchRFC3986:
		matches = func(have, want string) bool {
			want = strings.TrimSuffix(want, "/")
			return strings.EqualFold(have, want) || strings.HasPrefix(strings.ToLower(have), strings.ToLower(want)+"/")
		}
	case wsdMatchStrcmp0:
		matches = func(have, want string) bool { return have == want }
	default:
		return false
	}
	for _, want := range requested {
		found := false
		for _, have := range scopes {
			if matches(have, want) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// onvifEndpointUUID is the camera's stable endpoint reference: a
// version 8 (RFC 9562) UUID hashed from the device name, whose node field is
// the device MAC, as camera firmware commonly derives it. A client that has
// seen the camera once recognises it on the next Probe.
func onvifEndpointUUID(device *config.Device) string {
	sum := sha256.Sum256([]byte(device.Name))
	id := sum[:uuidBytes]
	id[6] = id[6]&uuidVersionMask | uuidVersion8
	id[8] = id[8]&uuidVariantMask | uuidVariant
	if len(device.MACAddress) == uuidMACBytes {
		copy(id[uuidNodeOffset:], device.MACAddress)
	}

	return wsdFormatUUID(id)
}

func wsdRandomUUID() string {
	id := make([]byte, uuidBytes)
	_, _ = rand.Read(id)
	id[6] = id[6]&uuidVersionMask | uuidVersion4
	id[8] = id[8]&uuidVariantMask | uuidVariant

	return wsdFormatUUID(id)
}

func wsdFormatUUID(id []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}

func xmlText(value string) string {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))

	return escaped.String()
}
