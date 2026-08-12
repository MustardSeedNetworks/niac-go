package snmp

import (
	"fmt"
	"sync/atomic"

	"github.com/gosnmp/gosnmp"
)

const (
	snmpGroup = "1.3.6.1.2.1.11"
	// enableAuthenTrapsOID is snmpEnableAuthenTraps, the one object this agent
	// implements read-write. Everything else it serves is read-only.
	enableAuthenTrapsOID   = snmpGroup + ".30.0"
	authenticationTrapsOn  = 1
	authenticationTrapsOff = 2
)

type snmpStats struct {
	inPkts, outPkts                                                    atomic.Uint32
	inBadVersions, inBadCommunityNames, inBadCommunityUses             atomic.Uint32
	inASNParseErrs, inBadTypes                                         atomic.Uint32
	inTooBigs, inNoSuchNames, inBadValues, inReadOnlys, inGenErrs      atomic.Uint32
	inTotalReqVars, inTotalSetVars                                     atomic.Uint32
	inGetRequests, inGetNexts, inSetRequests, inGetResponses, inTraps  atomic.Uint32
	outTooBigs, outNoSuchNames, outBadValues, outReadOnlys, outGenErrs atomic.Uint32
	outGetRequests, outGetNexts, outSetRequests, outGetResponses       atomic.Uint32
	outTraps, enableAuthenTraps                                        atomic.Uint32
}

func (a *Agent) initializeSNMPMIB() {
	a.stats.enableAuthenTraps.Store(authenticationTrapsOff)
	counters := []struct {
		suffix int
		value  *atomic.Uint32
	}{
		{1, &a.stats.inPkts},
		{2, &a.stats.outPkts},
		{3, &a.stats.inBadVersions},
		{4, &a.stats.inBadCommunityNames},
		{5, &a.stats.inBadCommunityUses},
		{6, &a.stats.inASNParseErrs},
		{7, &a.stats.inBadTypes},
		{8, &a.stats.inTooBigs},
		{9, &a.stats.inNoSuchNames},
		{10, &a.stats.inBadValues},
		{11, &a.stats.inReadOnlys},
		{12, &a.stats.inGenErrs},
		{13, &a.stats.inTotalReqVars},
		{14, &a.stats.inTotalSetVars},
		{15, &a.stats.inGetRequests},
		{16, &a.stats.inGetNexts},
		{17, &a.stats.inSetRequests},
		{18, &a.stats.inGetResponses},
		{19, &a.stats.inTraps},
		{20, &a.stats.outTooBigs},
		{21, &a.stats.outNoSuchNames},
		{22, &a.stats.outBadValues},
		{23, &a.stats.outReadOnlys},
		{24, &a.stats.outGenErrs},
		{25, &a.stats.outGetRequests},
		{26, &a.stats.outGetNexts},
		{27, &a.stats.outSetRequests},
		{28, &a.stats.outGetResponses},
		{29, &a.stats.outTraps},
	}
	for _, counter := range counters {
		a.mib.SetDynamic(fmt.Sprintf("%s.%d.0", snmpGroup, counter.suffix), func() *OIDValue {
			return &OIDValue{Type: gosnmp.Counter32, Value: counter.value.Load()}
		})
	}
	a.mib.SetDynamic(snmpGroup+".30.0", func() *OIDValue {
		return &OIDValue{Type: gosnmp.Integer, Value: int(a.stats.enableAuthenTraps.Load())}
	})
}

func (a *Agent) recordInboundPDU(pduType gosnmp.PDUType, variables int) {
	switch pduType {
	case gosnmp.GetRequest:
		a.stats.inGetRequests.Add(1)
		a.stats.inTotalReqVars.Add(safeUint32FromInt(variables))
	case gosnmp.GetNextRequest, gosnmp.GetBulkRequest:
		a.stats.inGetNexts.Add(1)
		a.stats.inTotalReqVars.Add(safeUint32FromInt(variables))
	case gosnmp.SetRequest:
		a.stats.inSetRequests.Add(1)
		a.stats.inTotalSetVars.Add(safeUint32FromInt(variables))
	case gosnmp.GetResponse:
		a.stats.inGetResponses.Add(1)
	case gosnmp.Trap, gosnmp.InformRequest, gosnmp.SNMPv2Trap:
		a.stats.inTraps.Add(1)
	case gosnmp.Sequence, gosnmp.Report:
		a.stats.inBadTypes.Add(1)
	}
}

// RecordInboundPacket records a datagram accepted by the agent transport.
func (a *Agent) RecordInboundPacket() { a.stats.inPkts.Add(1) }

// RecordBadVersion records a datagram carrying an unsupported SNMP version.
func (a *Agent) RecordBadVersion() { a.stats.inPkts.Add(1); a.stats.inBadVersions.Add(1) }

// RecordBadCommunityName records a request using an unknown community.
func (a *Agent) RecordBadCommunityName() {
	a.stats.inPkts.Add(1)
	a.stats.inBadCommunityNames.Add(1)
}

// RecordBadCommunityUse records a known community rejected by access policy.
func (a *Agent) RecordBadCommunityUse() {
	a.stats.inPkts.Add(1)
	a.stats.inBadCommunityUses.Add(1)
}

// RecordASNParseError records a datagram rejected by BER decoding.
func (a *Agent) RecordASNParseError() { a.stats.inPkts.Add(1); a.stats.inASNParseErrs.Add(1) }

// RecordResponse records a response handed to the transport and its SNMP error status.
func (a *Agent) RecordResponse(status gosnmp.SNMPError) {
	a.stats.outPkts.Add(1)
	a.stats.outGetResponses.Add(1)
	switch status {
	case gosnmp.TooBig:
		a.stats.outTooBigs.Add(1)
	case gosnmp.NoSuchName:
		a.stats.outNoSuchNames.Add(1)
	case gosnmp.BadValue:
		a.stats.outBadValues.Add(1)
	case gosnmp.ReadOnly:
		a.stats.outReadOnlys.Add(1)
	case gosnmp.GenErr:
		a.stats.outGenErrs.Add(1)
	case gosnmp.WrongType, gosnmp.WrongLength, gosnmp.WrongEncoding,
		gosnmp.WrongValue, gosnmp.InconsistentValue:
		a.stats.outBadValues.Add(1)
	case gosnmp.NoAccess, gosnmp.NoCreation, gosnmp.ResourceUnavailable, gosnmp.CommitFailed,
		gosnmp.UndoFailed, gosnmp.AuthorizationError, gosnmp.NotWritable, gosnmp.InconsistentName:
		a.stats.outGenErrs.Add(1)
	case gosnmp.NoError:
	}
}

// RecordInboundError accounts for the error status carried by an inbound PDU.
func (a *Agent) RecordInboundError(status gosnmp.SNMPError) {
	switch status {
	case gosnmp.TooBig:
		a.stats.inTooBigs.Add(1)
	case gosnmp.NoSuchName:
		a.stats.inNoSuchNames.Add(1)
	case gosnmp.BadValue:
		a.stats.inBadValues.Add(1)
	case gosnmp.ReadOnly:
		a.stats.inReadOnlys.Add(1)
	case gosnmp.GenErr:
		a.stats.inGenErrs.Add(1)
	case gosnmp.WrongType, gosnmp.WrongLength, gosnmp.WrongEncoding,
		gosnmp.WrongValue, gosnmp.InconsistentValue:
		a.stats.inBadValues.Add(1)
	case gosnmp.NoAccess, gosnmp.NoCreation, gosnmp.ResourceUnavailable, gosnmp.CommitFailed,
		gosnmp.UndoFailed, gosnmp.AuthorizationError, gosnmp.NotWritable, gosnmp.InconsistentName:
		a.stats.inGenErrs.Add(1)
	case gosnmp.NoError:
	}
}
