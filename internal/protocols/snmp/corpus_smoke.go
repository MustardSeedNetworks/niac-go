package snmp

import (
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
)

const (
	// replaySmokeRequests is the Get, GetNext and GetBulk the client sends.
	replaySmokeRequests = 3
	replaySmokeDeadline = 3 * time.Second
	replaySmokeTimeout  = time.Second
	replaySmokeMaxOIDs  = 60
	replaySmokeBulkReps = 4
	replayUDPBuffer     = 65535
	replayLoopback      = "127.0.0.1"
)

// replayUDPSmoke serves agent on an ephemeral loopback port and drives a
// Get, GetNext and GetBulk through gosnmp, so the agent's answers survive a
// real socket and a real client's decoder, not only an in-process round trip.
func replayUDPSmoke(agent *Agent) bool {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(replayLoopback)})
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()

	served := make(chan bool, 1)
	go func() { served <- replayServe(agent, conn) }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return false
	}
	clientOK := replaySmokeClient(addr.Port)
	if !clientOK {
		// Unblock the server so its goroutine does not outlive the replay;
		// the deferred Close then reports an error nobody needs.
		_ = conn.Close()
	}

	return <-served && clientOK
}

func replayServe(agent *Agent, conn *net.UDPConn) bool {
	buf := make([]byte, replayUDPBuffer)
	for range replaySmokeRequests {
		if err := conn.SetReadDeadline(time.Now().Add(replaySmokeDeadline)); err != nil {
			return false
		}
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return false
		}
		decoder := &gosnmp.GoSNMP{MaxOids: replayMaxOIDs}
		request, err := decoder.SnmpDecodePacket(buf[:n])
		if err != nil {
			return false
		}
		vars, status, index := agent.ProcessRequest(request)
		response := &gosnmp.SnmpPacket{
			Version: request.Version, Community: request.Community, PDUType: gosnmp.GetResponse,
			RequestID: request.RequestID, Variables: vars, Error: status, ErrorIndex: index,
		}
		body, err := response.MarshalMsg()
		if err != nil {
			return false
		}
		if _, err = conn.WriteToUDP(body, remote); err != nil {
			return false
		}
	}

	return true
}

func replaySmokeClient(port int) bool {
	client := &gosnmp.GoSNMP{
		Target: replayLoopback, Port: uint16(port), Community: "public", //nolint:gosec // an OS-assigned UDP port fits
		Version: gosnmp.Version2c, Timeout: replaySmokeTimeout, Retries: 0, MaxOids: replaySmokeMaxOIDs,
	}
	if err := client.Connect(); err != nil {
		return false
	}
	defer func() { _ = client.Conn.Close() }()

	got, err := client.Get([]string{replaySysDescrOID})
	if err != nil || len(got.Variables) != 1 {
		return false
	}
	got, err = client.GetNext([]string{"1.3.6.1.2.1.1"})
	if err != nil || len(got.Variables) != 1 {
		return false
	}
	got, err = client.GetBulk([]string{"1.3.6.1.2.1.1"}, 0, replaySmokeBulkReps)

	return err == nil && len(got.Variables) > 0
}
