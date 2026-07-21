package snmp

import (
	"strconv"
	"sync/atomic"

	"github.com/gosnmp/gosnmp"
)

const icmpMIBRoot = "1.3.6.1.2.1.5"

const (
	icmpTypeEchoReply         = 0
	icmpTypeDestUnreachable   = 3
	icmpTypeSourceQuench      = 4
	icmpTypeRedirect          = 5
	icmpTypeEchoRequest       = 8
	icmpTypeTimeExceeded      = 11
	icmpTypeParameterProblem  = 12
	icmpTypeTimestamp         = 13
	icmpTypeTimestampReply    = 14
	icmpTypeAddressMask       = 17
	icmpTypeAddressMaskReply  = 18
	icmpOutCounterOffset      = 13
	icmpInDestUnreachableOID  = 3
	icmpInTimeExceededOID     = 4
	icmpInParameterProblemOID = 5
	icmpInSourceQuenchOID     = 6
	icmpInRedirectOID         = 7
	icmpInEchoRequestOID      = 8
	icmpInEchoReplyOID        = 9
	icmpInTimestampOID        = 10
	icmpInTimestampReplyOID   = 11
	icmpInAddressMaskOID      = 12
	icmpInAddressMaskReplyOID = 13
)

// initializeICMPMIB registers the mandatory RFC 1213 ICMP counters against the
// device's shared packet telemetry.
func (a *Agent) initializeICMPMIB() {
	for id := 1; id <= 26; id++ {
		a.mib.Set(icmpMIBRoot+"."+strconv.Itoa(id)+".0", &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	}
	a.registerLiveICMPCounters()
}

func (a *Agent) registerLiveICMPCounters() {
	a.setProtocolCounter(icmpMIBRoot+".1.0", &a.protocolStats.icmpInMsgs)
	a.setProtocolCounter(icmpMIBRoot+".14.0", &a.protocolStats.icmpOutMsgs)
	for icmpType, oid := range map[uint8]int{
		icmpTypeDestUnreachable:  icmpInDestUnreachableOID,
		icmpTypeTimeExceeded:     icmpInTimeExceededOID,
		icmpTypeParameterProblem: icmpInParameterProblemOID,
		icmpTypeSourceQuench:     icmpInSourceQuenchOID,
		icmpTypeRedirect:         icmpInRedirectOID,
		icmpTypeEchoRequest:      icmpInEchoRequestOID,
		icmpTypeEchoReply:        icmpInEchoReplyOID,
		icmpTypeTimestamp:        icmpInTimestampOID,
		icmpTypeTimestampReply:   icmpInTimestampReplyOID,
		icmpTypeAddressMask:      icmpInAddressMaskOID,
		icmpTypeAddressMaskReply: icmpInAddressMaskReplyOID,
	} {
		a.setProtocolCounter(icmpMIBRoot+"."+strconv.Itoa(oid)+".0", &a.protocolStats.icmpInTypes[icmpType])
		outOID := icmpMIBRoot + "." + strconv.Itoa(oid+icmpOutCounterOffset) + ".0"
		a.setProtocolCounter(outOID, &a.protocolStats.icmpOutTypes[icmpType])
	}
}

func (a *Agent) setProtocolCounter(oid string, counter *atomic.Uint32) {
	a.mib.SetDynamic(oid, func() *OIDValue {
		return &OIDValue{Type: gosnmp.Counter32, Value: counter.Load()}
	})
}
