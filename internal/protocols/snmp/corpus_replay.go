package snmp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/pathconfine"
)

// Replay statuses. Only ReplayCompleted means every probe ran; the others name
// the stage that stopped the replay.
const (
	ReplayCompleted   = "completed"
	ReplayPanic       = "panic"
	ReplayLoadError   = "load_error"
	ReplayReadError   = "read_error"
	ReplayScanError   = "scan_error"
	ReplayEmpty       = "empty"
	ReplayUDPSmokeErr = "udp_smoke_error"
)

const (
	replaySysDescrOID    = "1.3.6.1.2.1.1.1.0"
	replaySysObjectIDOID = "1.3.6.1.2.1.1.2.0"

	// replayDeviceName is the harness device the recording is loaded into.
	// The agent derives the simulated uptime base (and a default sysName)
	// from the device name, and GetBulk stops at MaxBulkResponseBytes, so a
	// different name changes encoded sizes and can move a bulk response's
	// last binding across the budget. It must stay the name the catalog's
	// qualification receipts were recorded with, or bulk_checks drifts by
	// one for recordings that sit on that boundary.
	replayDeviceName = "corpus-audit"

	// replayGetBatch is how many recorded bindings one GetRequest carries.
	// A batch whose response fails to encode is retried one binding at a time
	// so a single bad value is counted once instead of failing its neighbours.
	replayGetBatch = 64
)

// ReplayReceipt is what replaying one recording through an agent proved. The
// counters are deterministic for a given recording and reader; only
// Milliseconds varies between runs.
//
// The field set and JSON names match the qualification columns in
// niac-demo-catalog's runtime/niac/qualified.csv, so a receipt can be compared
// field for field with the row that qualified the recording.
type ReplayReceipt struct {
	Status              string `json:"status"`
	SourceEntries       int    `json:"source_entries"`
	RejectedLines       int    `json:"rejected_lines"`
	InvalidOIDs         int    `json:"invalid_oids"`
	ProbeBindings       int    `json:"probe_bindings"`
	GetMissing          int    `json:"get_missing"`
	ContractOmitted     int    `json:"contract_omitted"`
	OpaqueWirePreserved int    `json:"opaque_wire_preserved"`
	BERFailures         int    `json:"ber_failures"`
	BERTypeChanges      int    `json:"ber_type_changes"`
	BERValueChanges     int    `json:"ber_value_changes"`
	ContractKept        int    `json:"contract_kept"`
	ContractSubstituted int    `json:"contract_substituted"`
	KeptTypeChanges     int    `json:"kept_type_changes"`
	KeptValueChanges    int    `json:"kept_value_changes"`
	Identity            bool   `json:"source_identity"`
	NextChecks          int    `json:"next_checks"`
	BulkChecks          int    `json:"bulk_checks"`
	OrderingFailures    int    `json:"ordering_failures"`
	Smoke               bool   `json:"udp_smoke"`
	Milliseconds        int64  `json:"milliseconds,omitempty"`
	// Error explains a load, read or scan failure.
	Error        string         `json:"error,omitempty"`
	FailureTypes map[string]int `json:"failure_types"`
}

// ReplayOptions selects the optional stages of ReplayRecording.
type ReplayOptions struct {
	// UDPSmoke serves the loaded agent on an ephemeral loopback port and
	// drives one Get, GetNext and GetBulk through a real SNMP client.
	UDPSmoke bool
}

// replayRun carries one replay's state between its stages.
type replayRun struct {
	agent   *Agent
	receipt *ReplayReceipt
	pending *WalkEntry
	batch   []WalkEntry
	descr   bool
	object  bool
}

// ReplayRecording loads a net-snmp walk or snmpsim recording into a fresh
// agent and checks what that agent serves against the recording itself.
//
// Every recorded binding is asked for with GetRequest and the response is
// BER-encoded and decoded again, so a value the reader accepted but the wire
// cannot carry is counted rather than crashing a later request. Values the
// walk contract keeps must come back with the recorded type and value; values
// it substitutes are counted separately. GetNext and GetBulk walks from fixed
// roots check lexicographic ordering. Expected values come from a reader
// independent of the production one (see corpusRecordScanner), so a decoder
// bug that changes both sides equally still shows as a kept-value change.
//
// A panic anywhere in the agent is caught and reported as ReplayPanic: the
// harness exists to find recordings that crash the reader.
func ReplayRecording(path string, opts ReplayOptions) ReplayReceipt {
	start := time.Now()
	receipt := ReplayReceipt{Status: ReplayCompleted, FailureTypes: map[string]int{}}
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				receipt.Status = ReplayPanic
				receipt.Error = fmt.Sprint(recovered)
			}
		}()
		replayInto(path, opts, &receipt)
	}()
	receipt.Milliseconds = time.Since(start).Milliseconds()

	return receipt
}

func replayInto(path string, opts ReplayOptions, receipt *ReplayReceipt) {
	device := &config.Device{
		Name:       replayDeviceName,
		Type:       "router",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		Properties: map[string]string{},
	}
	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(path); err != nil {
		receipt.Status = ReplayLoadError
		receipt.Error = err.Error()

		return
	}
	file, err := pathconfine.Open(path)
	if err != nil {
		receipt.Status = ReplayReadError
		receipt.Error = err.Error()

		return
	}
	defer func() { _ = file.Close() }()

	run := &replayRun{agent: agent, receipt: receipt}
	if scanErr := run.probeRecording(bufio.NewReaderSize(file, snmprecSniffWindow)); scanErr != nil {
		receipt.Status = ReplayScanError
		receipt.Error = scanErr.Error()

		return
	}
	if receipt.SourceEntries == 0 {
		receipt.Status = ReplayEmpty

		return
	}
	replayWalkChecks(agent, receipt)
	if opts.UDPSmoke {
		receipt.Smoke = replayUDPSmoke(agent)
		if !receipt.Smoke {
			receipt.Status = ReplayUDPSmokeErr
		}
	}
}

// probeRecording reads the recording a second time, independently of
// LoadWalkFile, and asks the agent for every binding it holds.
func (r *replayRun) probeRecording(reader *bufio.Reader) error {
	recording := contentIsSnmprec(reader)
	physical := bufio.NewScanner(reader)
	physical.Buffer(make([]byte, 0, walkScanBufInitial), walkScanBufMax)
	scanner := &corpusRecordScanner{scanner: physical, recording: recording}

	lastWasHex := false
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || isEndOfMIBLine(line) {
			continue
		}
		if !recording && lastWasHex && r.pending != nil && isHexContinuation(line) {
			appendHexContinuation(r.pending, line)

			continue
		}
		entry, err := parseRecordLine(raw, line, recording)
		if err != nil {
			r.receipt.RejectedLines++

			continue
		}
		r.accept(r.pending)
		r.pending = entry
		lastWasHex = isHexStringLine(line)
	}
	r.accept(r.pending)
	r.flush()
	r.receipt.Identity = r.descr && r.object

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan recording: %w", err)
	}

	return nil
}

// parseRecordLine parses one logical record. For a net-snmp walk the string
// value is replaced by the literal bytes between its quotes, so the expected
// value does not pass through the production string decoder.
func parseRecordLine(raw, line string, recording bool) (*WalkEntry, error) {
	if recording {
		return parseSnmprecLine(raw)
	}
	entry, err := parseWalkLine(line)
	if err != nil {
		return nil, err
	}
	if literal, ok := corpusLiteralString(raw); ok {
		entry.Value = literal
	}

	return entry, nil
}

func (r *replayRun) accept(entry *WalkEntry) {
	if entry == nil {
		return
	}
	r.receipt.SourceEntries++
	if !IsValidOID(entry.OID) {
		r.receipt.InvalidOIDs++

		return
	}
	switch trimOID(entry.OID) {
	case replaySysDescrOID:
		value, ok := octets(entry.Value)
		r.descr = ok && len(value) > 0 && entry.Type == gosnmp.OctetString
	case replaySysObjectIDOID:
		r.object = entry.Type == gosnmp.ObjectIdentifier && validWireOID(fmt.Sprint(entry.Value))
	}
	r.batch = append(r.batch, *entry)
	if len(r.batch) == replayGetBatch {
		r.flush()
	}
}

func (r *replayRun) flush() {
	if len(r.batch) > 0 {
		replayGetBatchInto(r.agent, r.batch, r.receipt)
		r.batch = r.batch[:0]
	}
}
