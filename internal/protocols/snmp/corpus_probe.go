package snmp

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

const (
	// replayNextSteps bounds the GetNext walk from the MIB root.
	replayNextSteps = 16
	// replayBulkRepetitions is max-repetitions for each GetBulk ordering probe.
	replayBulkRepetitions = 32
	// replayMissingPrefixArcs and replayFailurePrefixArcs truncate OIDs in
	// FailureTypes keys so they group by table instead of by row, and so a
	// key never carries an instance index that could identify a host.
	replayMissingPrefixArcs = 7
	replayFailurePrefixArcs = 9
	// replayMaxOIDs lets the decoder accept every binding a batch can hold.
	replayMaxOIDs = 65535
)

// replayBulkRoots are the GetBulk starting points: the whole tree, ifTable,
// ifXTable and the enterprise subtree, where ordering bugs have shown up.
func replayBulkRoots() []string {
	return []string{"1", "1.3.6.1.2.1.2", "1.3.6.1.2.1.31", "1.3.6.1.4.1"}
}

// replayGetBatchInto asks the agent for every entry in one GetRequest and
// checks the BER round trip of the response against the agent's own answer
// and against the recorded source.
func replayGetBatchInto(agent *Agent, entries []WalkEntry, receipt *ReplayReceipt) {
	vars := make([]gosnmp.SnmpPDU, len(entries))
	for i, entry := range entries {
		vars[i] = gosnmp.SnmpPDU{Name: entry.OID, Type: gosnmp.Null}
	}
	request := &gosnmp.SnmpPacket{Version: gosnmp.Version2c, PDUType: gosnmp.GetRequest, Variables: vars}
	response, status, index := agent.ProcessRequest(request)
	decoded, err := replayRoundTrip(&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse,
		RequestID: 1, Error: status, ErrorIndex: index, Variables: response,
	})
	if err != nil {
		if len(entries) > 1 {
			for _, entry := range entries {
				replayGetBatchInto(agent, []WalkEntry{entry}, receipt)
			}

			return
		}
		receipt.BERFailures++
		if len(response) > 0 {
			receipt.FailureTypes[strconv.Itoa(int(response[0].Type))]++
		}

		return
	}

	contract := agent.WalkContract()
	for i := range min(len(decoded.Variables), len(entries), len(response)) {
		receipt.ProbeBindings++
		kept := contract.Classify(entries[i].OID) == BucketKept
		replayCheckBinding(kept, entries[i], response[i], decoded.Variables[i], receipt)
	}
}

// replayCheckBinding compares one decoded binding with what the agent meant
// to send (BER fidelity) and, for contract-kept OIDs, with the recording.
func replayCheckBinding(kept bool, source WalkEntry, expected, got gosnmp.SnmpPDU, receipt *ReplayReceipt) {
	if got.Type == gosnmp.NoSuchObject || got.Type == gosnmp.NoSuchInstance {
		if !kept {
			receipt.ContractOmitted++
			receipt.ContractSubstituted++

			return
		}
		receipt.GetMissing++
		receipt.FailureTypes["missing:"+replayOIDPrefix(source.OID, replayMissingPrefixArcs)]++

		return
	}

	opaqueEqual := replayOpaqueEqual(expected, got)
	if opaqueEqual {
		receipt.OpaqueWirePreserved++
	}
	if got.Type != expected.Type && !opaqueEqual {
		receipt.BERTypeChanges++
	}
	if !opaqueEqual && !replaySameValue(got.Type, expected.Value, got.Value) {
		receipt.BERValueChanges++
	}
	if !kept {
		receipt.ContractSubstituted++

		return
	}

	receipt.ContractKept++
	sourcePDU := gosnmp.SnmpPDU{Name: source.OID, Type: source.Type, Value: source.Value}
	sourceOpaqueEqual := replayOpaqueEqual(sourcePDU, got)
	prefix := replayOIDPrefix(source.OID, replayFailurePrefixArcs)
	if source.Type != got.Type && !sourceOpaqueEqual {
		receipt.KeptTypeChanges++
		receipt.FailureTypes[fmt.Sprintf("kept_type:%s:%d>%d", prefix, source.Type, got.Type)]++
	}
	if !sourceOpaqueEqual && !replaySameValue(got.Type, source.Value, got.Value) {
		receipt.KeptValueChanges++
		receipt.FailureTypes["kept_value:"+prefix]++
	}
}

// replayWalkChecks walks the agent with GetNext from the root and GetBulk from
// a few fixed roots, counting any binding that does not advance the OID.
func replayWalkChecks(agent *Agent, receipt *ReplayReceipt) {
	current := "1"
	for range replayNextSteps {
		vars := agent.ProcessPDU(gosnmp.GetNextRequest, []gosnmp.SnmpPDU{{Name: current, Type: gosnmp.Null}}, 0, 0)
		if len(vars) == 0 || isEndOfMIB(vars[0]) {
			break
		}
		next := vars[0]
		receipt.NextChecks++
		if fidelityCompareOIDs(next.Name, current) <= 0 {
			receipt.OrderingFailures++

			break
		}
		if _, err := replayRoundTrip(replayResponse(vars)); err != nil {
			receipt.BERFailures++
		}
		current = next.Name
	}

	for _, root := range replayBulkRoots() {
		vars := agent.ProcessPDU(
			gosnmp.GetBulkRequest, []gosnmp.SnmpPDU{{Name: root, Type: gosnmp.Null}}, 0, replayBulkRepetitions,
		)
		previous := root
		for _, binding := range vars {
			if isEndOfMIB(binding) {
				break
			}
			receipt.BulkChecks++
			if fidelityCompareOIDs(binding.Name, previous) <= 0 {
				receipt.OrderingFailures++
			}
			previous = binding.Name
		}
		if _, err := replayRoundTrip(replayResponse(vars)); err != nil {
			receipt.BERFailures++
		}
	}
}

func replayResponse(vars []gosnmp.SnmpPDU) *gosnmp.SnmpPacket {
	return &gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, RequestID: 1, Variables: vars,
	}
}

// replayRoundTrip encodes a packet as it would go on the wire and decodes it
// again with a client-side decoder.
func replayRoundTrip(packet *gosnmp.SnmpPacket) (*gosnmp.SnmpPacket, error) {
	body, err := packet.MarshalMsg()
	if err != nil {
		return nil, fmt.Errorf("encode response: %w", err)
	}
	decoder := &gosnmp.GoSNMP{MaxOids: replayMaxOIDs}
	decoded, err := decoder.SnmpDecodePacket(body)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return decoded, nil
}

// replaySameValue compares two values of the decoded type. Octet-like values
// compare as bytes, OIDs without a leading dot, addresses as IPs.
func replaySameValue(kind gosnmp.Asn1BER, left, right any) bool {
	//nolint:exhaustive // every other type compares by its printed value below
	switch kind {
	case gosnmp.Null:
		return left == nil && right == nil
	case gosnmp.OctetString, gosnmp.Opaque, gosnmp.BitString:
		leftBytes, leftOK := octets(left)
		rightBytes, rightOK := octets(right)

		return leftOK && rightOK && bytes.Equal(leftBytes, rightBytes)
	case gosnmp.ObjectIdentifier:
		return trimOID(fmt.Sprint(left)) == trimOID(fmt.Sprint(right))
	case gosnmp.IPAddress:
		return net.ParseIP(fmt.Sprint(left)).Equal(net.ParseIP(fmt.Sprint(right)))
	default:
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
}

// replayOpaqueEqual reports a recorded Opaque that gosnmp decoded as an
// OpaqueFloat or OpaqueDouble. The decoded type differs but the encoded bytes
// are identical, so the wire carried the recording faithfully.
func replayOpaqueEqual(source, decoded gosnmp.SnmpPDU) bool {
	if source.Type != gosnmp.Opaque || (decoded.Type != gosnmp.OpaqueFloat && decoded.Type != gosnmp.OpaqueDouble) {
		return false
	}
	left, err := replayResponse([]gosnmp.SnmpPDU{source}).MarshalMsg()
	if err != nil {
		return false
	}
	right, err := replayResponse([]gosnmp.SnmpPDU{decoded}).MarshalMsg()

	return err == nil && bytes.Equal(left, right)
}

func replayOIDPrefix(oid string, arcs int) string {
	parts := strings.Split(trimOID(oid), ".")
	if len(parts) > arcs {
		parts = parts[:arcs]
	}

	return strings.Join(parts, ".")
}
