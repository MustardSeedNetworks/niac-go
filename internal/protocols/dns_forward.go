package protocols

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A server holding no record for a name asks its forwarders in order, the way
// a site resolver forwards to its upstream. A forwarder that is itself a
// simulated DNS server answers from its own records, and its own forwarders,
// inside the simulation: a site resolver pointed at the lab edge returns the
// simulated internet that the rest of the scenario routes to. Any other
// address is a real resolver, asked over the host's network off the packet
// loop, and only when the simulated chain has no answer.
const (
	dnsUpstreamTimeout     = 2 * time.Second
	dnsUpstreamMaxInFlight = 64
	dnsUpstreamBufferSize  = 65535
)

func dnsUnanswered(response *layers.DNS) bool {
	return len(response.Answers) == 0 && response.ResponseCode == layers.DNSResponseCodeNXDomain
}

// resolveForwarded walks the simulated forwarder chain behind set, nearest
// first, and returns the first answer found. When the chain has none it
// returns the real upstream resolvers met on the way, for the caller to ask.
func (h *DNSHandler) resolveForwarded(
	questions []layers.DNSQuestion, set *dnsRecordSet, debugLevel, serial int,
) ([]layers.DNSResourceRecord, layers.DNSResponseCode, []net.IP) {
	visited := map[*dnsRecordSet]bool{set: true}
	pending := slices.Clone(set.forwarders)
	var upstreams []net.IP

	for len(pending) > 0 {
		forwarder := pending[0]
		pending = pending[1:]

		next := h.simulatedResolver(forwarder)
		if next == nil {
			upstreams = append(upstreams, forwarder)
			continue
		}
		if visited[next] {
			continue
		}
		visited[next] = true

		answers, rcode := h.resolveQuestions(questions, next, debugLevel, serial)
		if len(answers) > 0 || rcode != layers.DNSResponseCodeNXDomain {
			return answers, rcode, nil
		}
		pending = append(pending, next.forwarders...)
	}

	return nil, layers.DNSResponseCodeNXDomain, upstreams
}

// simulatedResolver returns the record set of the simulated DNS server that
// owns address, or nil when no simulated server does.
func (h *DNSHandler) simulatedResolver(address net.IP) *dnsRecordSet {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for device, set := range h.deviceRecords {
		if slices.ContainsFunc(device.IPAddresses, address.Equal) {
			return set
		}
	}

	return nil
}

// askUpstream answers response from the real resolvers in upstreams and hands
// it to reply. The exchange runs off the packet loop; a query beyond the
// in-flight cap, or one no resolver answers, is answered SERVFAIL, which is
// what a resolver whose upstream is unreachable returns.
func (h *DNSHandler) askUpstream(
	response *layers.DNS, upstreams []net.IP, debugLevel, serial int, reply func(*layers.DNS),
) {
	response.AA = false
	// The question names alias the received frame, which the packet loop is
	// free to reuse once this handler returns.
	response.Questions = cloneDNSQuestions(response.Questions)

	select {
	case h.upstreamSlots <- struct{}{}:
	default:
		response.ResponseCode = layers.DNSResponseCodeServFail
		reply(response)
		return
	}

	h.stack.wg.Go(func() {
		defer func() { <-h.upstreamSlots }()

		answer, err := h.queryUpstreams(upstreams, response.Questions)
		if err != nil {
			if debugLevel >= DebugLevelInfo {
				logging.Debugf("DNS: upstream forwarders failed: %v sn=%d", err, serial)
			}
			response.ResponseCode = layers.DNSResponseCodeServFail
		} else {
			response.Answers = encodableAnswers(answer.Answers)
			response.ResponseCode = answer.ResponseCode
		}
		reply(response)
	})
}

// queryUpstreams asks each resolver in turn within one shared deadline and
// returns the first reply.
func (h *DNSHandler) queryUpstreams(upstreams []net.IP, questions []layers.DNSQuestion) (*layers.DNS, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dnsUpstreamTimeout)
	defer cancel()

	var failures []error
	for _, upstream := range upstreams {
		answer, err := h.queryUpstream(ctx, upstream, questions)
		if err == nil {
			return answer, nil
		}
		failures = append(failures, fmt.Errorf("%s: %w", upstream, err))
	}

	return nil, errors.Join(failures...)
}

func (h *DNSHandler) queryUpstream(
	ctx context.Context, upstream net.IP, questions []layers.DNSQuestion,
) (*layers.DNS, error) {
	var idBytes [2]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return nil, fmt.Errorf("query id: %w", err)
	}
	query := &layers.DNS{ID: binary.BigEndian.Uint16(idBytes[:]), RD: true, Questions: questions}

	buffer := gopacket.NewSerializeBuffer()
	if err := query.SerializeTo(buffer, gopacket.SerializeOptions{FixLengths: true}); err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp", net.JoinHostPort(upstream.String(), strconv.Itoa(h.upstreamPort)))
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("deadline: %w", err)
		}
	}
	if _, err = conn.Write(buffer.Bytes()); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	reply := make([]byte, dnsUpstreamBufferSize)
	for {
		length, readErr := conn.Read(reply)
		if readErr != nil {
			return nil, fmt.Errorf("receive: %w", readErr)
		}

		answer := &layers.DNS{}
		if answer.DecodeFromBytes(reply[:length], gopacket.NilDecodeFeedback) != nil ||
			!answer.QR || answer.ID != query.ID {
			// A stray or malformed datagram is not the reply; keep waiting
			// for it until the deadline.
			continue
		}

		return answer, nil
	}
}

// answerQuery builds the response to query from serverDevice's records and
// forwarders and hands it to reply: at once when the simulation answers it,
// later when a real upstream resolver has to be asked.
func (h *DNSHandler) answerQuery(
	query *layers.DNS, serverDevice *config.Device, debugLevel, serial int, reply func(*layers.DNS),
) {
	response := &layers.DNS{
		ID:           query.ID,
		QR:           true,
		OpCode:       query.OpCode,
		AA:           true,
		RD:           query.RD,
		RA:           true,
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    query.Questions,
		Answers:      []layers.DNSResourceRecord{},
	}

	set := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(query.Questions, set, debugLevel, serial)
	response = h.applyNXDomainFault(response, serverDevice)

	if !dnsUnanswered(response) || set == nil || len(set.forwarders) == 0 ||
		h.stack.deviceFaultActive(serverDevice, devicestate.FaultDNSNXDomain) {
		h.logUnanswered(response, serial, debugLevel)
		reply(response)
		return
	}

	answers, rcode, upstreams := h.resolveForwarded(query.Questions, set, debugLevel, serial)
	if len(upstreams) == 0 {
		response.AA = len(answers) == 0
		response.Answers, response.ResponseCode = answers, rcode
		if response.Answers == nil {
			response.Answers = []layers.DNSResourceRecord{}
		}
		h.logUnanswered(response, serial, debugLevel)
		reply(response)
		return
	}

	h.askUpstream(response, upstreams, debugLevel, serial, reply)
}

// encodableAnswers keeps the upstream records the reply can carry. The
// serializer rejects record types it has no encoder for, and one such record
// would fail the whole reply, leaving the client to time out.
func encodableAnswers(answers []layers.DNSResourceRecord) []layers.DNSResourceRecord {
	kept := make([]layers.DNSResourceRecord, 0, len(answers))
	for _, answer := range answers {
		probe := &layers.DNS{Answers: []layers.DNSResourceRecord{answer}}
		if probe.SerializeTo(gopacket.NewSerializeBuffer(), gopacket.SerializeOptions{FixLengths: true}) == nil {
			kept = append(kept, answer)
		}
	}
	return kept
}

func cloneDNSQuestions(questions []layers.DNSQuestion) []layers.DNSQuestion {
	cloned := make([]layers.DNSQuestion, len(questions))
	for i, question := range questions {
		cloned[i] = layers.DNSQuestion{Name: slices.Clone(question.Name), Type: question.Type, Class: question.Class}
	}
	return cloned
}

func (h *DNSHandler) logUnanswered(response *layers.DNS, serial, debugLevel int) {
	if len(response.Answers) == 0 && debugLevel >= DebugLevelInfo {
		logging.Debugf("DNS: %s for queries sn=%d", response.ResponseCode, serial)
	}
}
