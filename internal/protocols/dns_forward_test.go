package protocols

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// answerNow runs answerQuery and waits for the reply, which arrives at once
// from the simulation and later from a real upstream.
func answerNow(h *DNSHandler, query *layers.DNS, server *config.Device, serial int) *layers.DNS {
	replies := make(chan *layers.DNS, 1)
	h.answerQuery(query, server, 0, serial, func(response *layers.DNS) { replies <- response })
	select {
	case response := <-replies:
		return response
	case <-time.After(2 * dnsUpstreamTimeout):
		return nil
	}
}

func forwardingStack(t *testing.T, yaml string) *Stack {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	return NewStack(nil, cfg, logging.NewDebugConfig(0))
}

func aQuery(name string) *layers.DNS {
	return &layers.DNS{ID: 9, RD: true, Questions: []layers.DNSQuestion{
		{Name: []byte(name), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}}
}

func deviceNamed(stack *Stack, name string) *config.Device {
	for _, device := range stack.AllDevices() {
		if device.Name == name {
			return device
		}
	}
	return nil
}

const simulatedChain = `devices:
  - name: site-dns
    mac: "02:00:00:00:00:01"
    ips: [192.0.2.10]
    dns:
      forwarders: [192.0.2.1]
      forward_records:
        - name: printer.site.example
          ip: 192.0.2.50
  - name: edge
    mac: "02:00:00:00:00:02"
    ips: [192.0.2.1]
    dns:
      forwarders: [192.0.2.10]
      forward_records:
        - name: www.google.com
          ip: 8.8.8.8
        - name: printer.site.example
          ip: 198.51.100.50
`

func TestDNSForwardersResolveInsideTheSimulation(t *testing.T) {
	stack := forwardingStack(t, simulatedChain)
	site := deviceNamed(stack, "site-dns")
	tests := []struct {
		name   string
		query  string
		want   string
		rcode  layers.DNSResponseCode
		author bool
	}{
		{"miss forwards to the edge", "www.google.com", "8.8.8.8", layers.DNSResponseCodeNoErr, false},
		{
			"own record wins over the forwarder", "printer.site.example", "192.0.2.50",
			layers.DNSResponseCodeNoErr, true,
		},
		{"forwarding loop ends in not-found", "nowhere.example", "", layers.DNSResponseCodeNXDomain, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := answerNow(stack.dnsHandler, aQuery(tc.query), site, 1)
			if response == nil {
				t.Fatal("no reply")
			}
			if response.ResponseCode != tc.rcode || response.AA != tc.author {
				t.Fatalf("rcode %v AA %v, want %v AA %v", response.ResponseCode, response.AA, tc.rcode, tc.author)
			}
			if tc.want == "" {
				if len(response.Answers) != 0 {
					t.Fatalf("answers = %v, want none", response.Answers)
				}
				return
			}
			if len(response.Answers) != 1 || !response.Answers[0].IP.Equal(net.ParseIP(tc.want)) {
				t.Fatalf("answers = %v, want %s", response.Answers, tc.want)
			}
		})
	}
}

// fakeUpstream is a real resolver on loopback that answers every A query with
// address.
func fakeUpstream(t *testing.T, address string) int {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	go func() {
		buffer := make([]byte, dnsUpstreamBufferSize)
		for {
			length, peer, readErr := conn.ReadFrom(buffer)
			if readErr != nil {
				return
			}
			query := &layers.DNS{}
			if query.DecodeFromBytes(buffer[:length], gopacket.NilDecodeFeedback) != nil {
				continue
			}
			reply := &layers.DNS{
				ID: query.ID, QR: true, RD: query.RD, RA: true, Questions: query.Questions,
				Answers: []layers.DNSResourceRecord{{
					Name: query.Questions[0].Name, Type: layers.DNSTypeA, Class: layers.DNSClassIN,
					TTL: 60, IP: net.ParseIP(address),
				}},
			}
			out := gopacket.NewSerializeBuffer()
			if reply.SerializeTo(out, gopacket.SerializeOptions{FixLengths: true}) == nil {
				_, _ = conn.WriteTo(out.Bytes(), peer)
			}
		}
	}()
	_, port, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return number
}

const upstreamChain = `devices:
  - name: site-dns
    mac: "02:00:00:00:00:01"
    ips: [192.0.2.10]
    dns:
      forwarders: [192.0.2.1]
  - name: edge
    mac: "02:00:00:00:00:02"
    ips: [192.0.2.1]
    dns:
      forwarders: [127.0.0.1]
      forward_records:
        - name: www.google.com
          ip: 8.8.8.8
`

func TestDNSForwardersAskRealUpstreamWhenTheSimulationHasNoAnswer(t *testing.T) {
	stack := forwardingStack(t, upstreamChain)
	stack.dnsHandler.upstreamPort = fakeUpstream(t, "203.0.113.80")
	site := deviceNamed(stack, "site-dns")

	simulated := answerNow(stack.dnsHandler, aQuery("www.google.com"), site, 1)
	if simulated == nil || len(simulated.Answers) != 1 || !simulated.Answers[0].IP.Equal(net.ParseIP("8.8.8.8")) {
		t.Fatalf("simulated internet name = %+v, want the edge's 8.8.8.8", simulated)
	}

	upstream := answerNow(stack.dnsHandler, aQuery("example.org"), site, 2)
	if upstream == nil {
		t.Fatal("no reply from the upstream path")
	}
	if upstream.ResponseCode != layers.DNSResponseCodeNoErr || upstream.AA || upstream.ID != 9 ||
		len(upstream.Answers) != 1 || !upstream.Answers[0].IP.Equal(net.ParseIP("203.0.113.80")) {
		t.Fatalf("upstream answer = rcode %v AA %v id %d answers %v",
			upstream.ResponseCode, upstream.AA, upstream.ID, upstream.Answers)
	}
}

func TestDNSForwardersAnswerServFailWhenNoUpstreamReplies(t *testing.T) {
	stack := forwardingStack(t, upstreamChain)
	silent, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = silent.Close() })
	_, port, _ := net.SplitHostPort(silent.LocalAddr().String())
	stack.dnsHandler.upstreamPort, _ = strconv.Atoi(port)

	response := answerNow(stack.dnsHandler, aQuery("example.org"), deviceNamed(stack, "site-dns"), 1)
	if response == nil || response.ResponseCode != layers.DNSResponseCodeServFail || len(response.Answers) != 0 {
		t.Fatalf("unreachable upstream = %+v, want SERVFAIL", response)
	}
}

func TestDNSUpstreamAnswersTheReplyCannotCarryAreDropped(t *testing.T) {
	carried := layers.DNSResourceRecord{
		Name: []byte("example.org"), Type: layers.DNSTypeA, Class: layers.DNSClassIN, IP: net.ParseIP("203.0.113.80"),
	}
	caa := layers.DNSResourceRecord{
		Name: []byte("example.org"), Type: layers.DNSType(257), Class: layers.DNSClassIN, Data: []byte{0, 5, 'i'},
	}
	kept := encodableAnswers([]layers.DNSResourceRecord{caa, carried})
	if len(kept) != 1 || kept[0].Type != layers.DNSTypeA {
		t.Fatalf("kept = %v, want only the A record", kept)
	}
}
