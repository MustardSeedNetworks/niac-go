package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// TestPTRAnswerParsesBackOnTheWire guards the wire form of a PTR reply.
//
// gopacket appends the root label when it serializes a name, so handing it a
// name that already ends in "." adds an empty label and a second terminator.
// The record then declares one more byte of RDATA than the name occupies, and
// every resolver rejects the reply with "extra input data". That is why
// simulated endpoints rendered as bare IP addresses in Link-Live instead of
// their hostnames: the forward lookup worked, the reverse lookup was discarded.
func TestPTRAnswerParsesBackOnTheWire(t *testing.T) {
	const (
		wantName = "med-nurse-b01-f01-01.care.example"
		query    = "21.210.51.10.in-addr.arpa"
	)

	handler := NewDNSHandler(newTestStackInternal(t))
	set := &dnsRecordSet{
		forward: make(map[string][]dnsRecord),
		reverse: map[string]dnsPTR{
			net.ParseIP("10.51.210.21").String(): {name: wantName, ttl: 300},
		},
	}
	ctx := &dnsResolveContext{}

	handler.resolvePTRRecord(layers.DNSQuestion{
		Name:  []byte(query),
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
	}, set, ctx)

	if len(ctx.answers) != 1 {
		t.Fatalf("resolver produced %d answers, want 1", len(ctx.answers))
	}

	response := &layers.DNS{
		ID: 1, QR: true, AA: true,
		Questions: []layers.DNSQuestion{{
			Name: []byte(query), Type: layers.DNSTypePTR, Class: layers.DNSClassIN,
		}},
		Answers: ctx.answers,
	}
	buf := gopacket.NewSerializeBuffer()
	if err := response.SerializeTo(buf, gopacket.SerializeOptions{FixLengths: true}); err != nil {
		t.Fatalf("serialize PTR response: %v", err)
	}

	var decoded layers.DNS
	if err := decoded.DecodeFromBytes(buf.Bytes(), gopacket.NilDecodeFeedback); err != nil {
		t.Fatalf("a resolver could not parse our PTR reply: %v", err)
	}
	if len(decoded.Answers) != 1 {
		t.Fatalf("decoded %d answers, want 1", len(decoded.Answers))
	}
	if got := string(decoded.Answers[0].PTR); got != wantName {
		t.Errorf("PTR name = %q, want %q", got, wantName)
	}

	// gopacket's decoder tolerates trailing slack inside RDATA; dig and BIND do
	// not, so assert the declared length matches the name exactly. A wire name
	// costs one length byte per label plus a single root terminator, which for
	// a dot-separated name is len(name)+2.
	wantRDLength := len(wantName) + 2
	if got := rdLengthOfFirstAnswer(t, buf.Bytes()); got != wantRDLength {
		t.Errorf("PTR RDLENGTH = %d, want %d (extra bytes make resolvers reject the reply)",
			got, wantRDLength)
	}
}

// rdLengthOfFirstAnswer walks the header, question and answer name to read the
// first answer's declared RDLENGTH straight off the wire.
func rdLengthOfFirstAnswer(t *testing.T, packet []byte) int {
	t.Helper()
	const (
		headerLen    = 12
		typeClassLen = 4
		ttlLen       = 4
	)
	offset := headerLen
	skipName := func() {
		for offset < len(packet) && packet[offset] != 0 {
			offset += int(packet[offset]) + 1
		}
		offset++ // root terminator
	}
	skipName()             // question name
	offset += typeClassLen // question type + class
	skipName()             // answer name
	offset += typeClassLen // answer type + class
	offset += ttlLen       // answer TTL
	if offset+1 >= len(packet) {
		t.Fatalf("packet truncated before RDLENGTH at offset %d of %d", offset, len(packet))
	}
	return int(packet[offset])<<8 | int(packet[offset+1])
}
