package snmp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/gosnmp/gosnmp"
)

type CorpusReplayReceipt struct {
	Status              string         `json:"status"`
	SourceEntries       int            `json:"source_entries"`
	RejectedLines       int            `json:"rejected_lines"`
	InvalidOIDs         int            `json:"invalid_oids"`
	ProbeBindings       int            `json:"probe_bindings"`
	GetMissing          int            `json:"get_missing"`
	ContractOmitted     int            `json:"contract_omitted"`
	OpaqueWirePreserved int            `json:"opaque_wire_preserved"`
	BERFailures         int            `json:"ber_failures"`
	BERTypeChanges      int            `json:"ber_type_changes"`
	BERValueChanges     int            `json:"ber_value_changes"`
	ContractKept        int            `json:"contract_kept"`
	ContractSubstituted int            `json:"contract_substituted"`
	KeptTypeChanges     int            `json:"kept_type_changes"`
	KeptValueChanges    int            `json:"kept_value_changes"`
	Identity            bool           `json:"source_identity"`
	NextChecks          int            `json:"next_checks"`
	BulkChecks          int            `json:"bulk_checks"`
	OrderingFailures    int            `json:"ordering_failures"`
	Smoke               bool           `json:"udp_smoke"`
	NetSNMPClient       bool           `json:"net_snmp_client"`
	Milliseconds        int64          `json:"milliseconds"`
	FailureTypes        map[string]int `json:"failure_types"`
}

func RunCorpusReplayLocal(path string, smoke bool) (r CorpusReplayReceipt) {
	start := time.Now()
	r.Status = "completed"
	r.FailureTypes = map[string]int{}
	defer func() {
		r.Milliseconds = time.Since(start).Milliseconds()
		if recover() != nil {
			r.Status = "panic"
		}
	}()
	debug.SetMemoryLimit(768 << 20)
	logging.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	device := &config.Device{Name: "corpus-audit", Type: "router", MACAddress: net.HardwareAddr{0, 17, 34, 51, 68, 85}, Properties: map[string]string{}}
	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(path); err != nil {
		r.Status = "load_error"
		return r
	}
	file, err := os.Open(path)
	if err != nil {
		r.Status = "read_error"
		return r
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	snmprec := contentIsSnmprec(reader)
	physicalScanner := bufio.NewScanner(reader)
	physicalScanner.Buffer(make([]byte, 65536), 4<<20)
	scanner := &corpusRecordsLocal{scanner:physicalScanner,recording:snmprec}
	var pending *WalkEntry
	var batch []WalkEntry
	var descr, object, lastWasHex bool
	flush := func() {
		if len(batch) > 0 {
			corpusGetBatch(agent, batch, &r)
			batch = batch[:0]
		}
	}
	accept := func(e *WalkEntry) {
		if e == nil {
			return
		}
		r.SourceEntries++
		if !IsValidOID(e.OID) {
			r.InvalidOIDs++
			return
		}
		if trimOID(e.OID) == "1.3.6.1.2.1.1.1.0" {
			v, ok := octets(e.Value)
			descr = ok && len(v) > 0 && e.Type == gosnmp.OctetString
		}
		if trimOID(e.OID) == "1.3.6.1.2.1.1.2.0" {
			object = e.Type == gosnmp.ObjectIdentifier && validWireOID(fmt.Sprint(e.Value))
		}
		batch = append(batch, *e)
		if len(batch) == 64 {
			flush()
		}
	}
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || isEndOfMIBLine(line) {
			continue
		}
		if !snmprec && lastWasHex && pending != nil && isHexContinuation(line) {
			appendHexContinuation(pending, line)
			continue
		}
		var e *WalkEntry
		var err error
		if snmprec {
			e, err = parseSnmprecLine(rawLine)
		} else {
			e, err = parseWalkLine(line)
		}
		if err != nil {
			r.RejectedLines++
			continue
		}
		if !snmprec { if literal,ok:=corpusLiteralStringLocal(rawLine);ok { e.Value=literal } }
		accept(pending)
		pending = e
		lastWasHex = isHexStringLine(line)
	}
	accept(pending)
	flush()
	r.Identity = descr && object
	if scanner.Err() != nil {
		r.Status = "scan_error"
		return r
	}
	if r.SourceEntries == 0 {
		r.Status = "empty"
		return r
	}
	corpusWalkChecks(agent, &r)
	if smoke {
		r.NetSNMPClient = corpusExternalClient
		r.Smoke = corpusUDPSmoke(agent)
		if !r.Smoke {
			r.Status = "udp_smoke_error"
		}
	}
	return r
}

func corpusGetBatch(agent *Agent, entries []WalkEntry, r *CorpusReplayReceipt) {
	vars := make([]gosnmp.SnmpPDU, len(entries))
	for i, e := range entries {
		vars[i] = gosnmp.SnmpPDU{Name: e.OID, Type: gosnmp.Null}
	}
	request := &gosnmp.SnmpPacket{Version: gosnmp.Version2c, PDUType: gosnmp.GetRequest, Variables: vars}
	response, status, index := agent.ProcessRequest(request)
	packet := &gosnmp.SnmpPacket{Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, RequestID: 1, Error: status, ErrorIndex: index, Variables: response}
	decoded, err := corpusPacketRoundtrip(packet)
	if err != nil {
		if len(entries) > 1 {
			for _, e := range entries {
				corpusGetBatch(agent, []WalkEntry{e}, r)
			}
			return
		}
		r.BERFailures++
		r.FailureTypes[fmt.Sprintf("%d", response[0].Type)]++
		return
	}
	contract := agent.WalkContract()
	for i, got := range decoded.Variables {
		r.ProbeBindings++
		expected := response[i]
		source := entries[i]
		if got.Type == gosnmp.NoSuchObject || got.Type == gosnmp.NoSuchInstance {
			if contract.Classify(source.OID) != BucketKept {
				r.ContractOmitted++
				r.ContractSubstituted++
				continue
			}
			r.GetMissing++
			parts := strings.Split(trimOID(source.OID), ".")
			if len(parts) > 7 {
				parts = parts[:7]
			}
			r.FailureTypes["missing:"+strings.Join(parts, ".")]++
			continue
		}
		opaqueEqual := corpusOpaqueEqual(expected, got)
		if opaqueEqual {
			r.OpaqueWirePreserved++
		}
		if got.Type != expected.Type && !opaqueEqual {
			r.BERTypeChanges++
		}
		if !opaqueEqual && !corpusSameValue(got.Type, expected.Value, got.Value) {
			r.BERValueChanges++
		}
		if contract.Classify(source.OID) != BucketKept {
			r.ContractSubstituted++
			continue
		}
		r.ContractKept++
		sourceOpaqueEqual := corpusOpaqueEqual(gosnmp.SnmpPDU{Name: source.OID, Type: source.Type, Value: source.Value}, got)
		if source.Type != got.Type && !sourceOpaqueEqual {
			r.KeptTypeChanges++
			r.FailureTypes["kept_type:"+corpusSafePrefix(source.OID)+fmt.Sprintf(":%d>%d", source.Type, got.Type)]++
		}
		if !sourceOpaqueEqual && !corpusSameValue(got.Type, source.Value, got.Value) {
			r.KeptValueChanges++
			r.FailureTypes["kept_value:"+corpusSafePrefix(source.OID)]++
		}
	}
}

func corpusPacketRoundtrip(packet *gosnmp.SnmpPacket) (*gosnmp.SnmpPacket, error) {
	body, err := packet.MarshalMsg()
	if err != nil {
		return nil, err
	}
	decoder := &gosnmp.GoSNMP{MaxOids: 65535}
	return decoder.SnmpDecodePacket(body)
}

func corpusSameValue(kind gosnmp.Asn1BER, a, b any) bool {
	if kind == gosnmp.Null {
		return a == nil && b == nil
	}
	if kind == gosnmp.OctetString || kind == gosnmp.Opaque || kind == gosnmp.BitString {
		left, l := octets(a)
		right, r := octets(b)
		return l && r && string(left) == string(right)
	}
	if kind == gosnmp.ObjectIdentifier {
		return trimOID(fmt.Sprint(a)) == trimOID(fmt.Sprint(b))
	}
	if kind == gosnmp.IPAddress {
		return net.ParseIP(fmt.Sprint(a)).Equal(net.ParseIP(fmt.Sprint(b)))
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func corpusWalkChecks(agent *Agent, r *CorpusReplayReceipt) {
	current := "1"
	for range 16 {
		vars := agent.ProcessPDU(gosnmp.GetNextRequest, []gosnmp.SnmpPDU{{Name: current, Type: gosnmp.Null}}, 0, 0)
		if len(vars) == 0 || isEndOfMIB(vars[0]) {
			break
		}
		next := vars[0]
		r.NextChecks++
		if fidelityCompareOIDs(next.Name, current) <= 0 {
			r.OrderingFailures++
			break
		}
		if _, err := corpusPacketRoundtrip(&gosnmp.SnmpPacket{Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, Variables: vars}); err != nil {
			r.BERFailures++
		}
		current = next.Name
	}
	for _, root := range []string{"1", "1.3.6.1.2.1.2", "1.3.6.1.2.1.31", "1.3.6.1.4.1"} {
		vars := agent.ProcessPDU(gosnmp.GetBulkRequest, []gosnmp.SnmpPDU{{Name: root, Type: gosnmp.Null}}, 0, 32)
		previous := root
		for _, v := range vars {
			if isEndOfMIB(v) {
				break
			}
			r.BulkChecks++
			if fidelityCompareOIDs(v.Name, previous) <= 0 {
				r.OrderingFailures++
			}
			previous = v.Name
		}
		if _, err := corpusPacketRoundtrip(&gosnmp.SnmpPacket{Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, Variables: vars}); err != nil {
			r.BERFailures++
		}
	}
}

func WriteCorpusReplayLocal(path string, smoke bool) {
	r := RunCorpusReplayLocal(path, smoke)
	_ = json.NewEncoder(os.Stdout).Encode(r)
}

func WriteCorpusReplayNetSNMPLocal(path string) {
	corpusExternalClient = true
	WriteCorpusReplayLocal(path, true)
}

func corpusSafePrefix(oid string) string {
	parts := strings.Split(trimOID(oid), ".")
	if len(parts) > 9 {
		parts = parts[:9]
	}
	return strings.Join(parts, ".")
}

func corpusOpaqueEqual(source, decoded gosnmp.SnmpPDU) bool {
	if source.Type != gosnmp.Opaque || (decoded.Type != gosnmp.OpaqueFloat && decoded.Type != gosnmp.OpaqueDouble) {
		return false
	}
	packet := func(v gosnmp.SnmpPDU) *gosnmp.SnmpPacket {
		return &gosnmp.SnmpPacket{Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, RequestID: 1, Variables: []gosnmp.SnmpPDU{v}}
	}
	left, err := packet(source).MarshalMsg()
	if err != nil {
		return false
	}
	right, err := packet(decoded).MarshalMsg()
	return err == nil && bytes.Equal(left, right)
}
