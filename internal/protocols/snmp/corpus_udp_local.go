package snmp

import (
	"github.com/gosnmp/gosnmp"
	"io"
	"net"
	"os/exec"
	"time"
)

var corpusExternalClient bool

func corpusUDPSmoke(agent *Agent) bool {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return false
	}
	defer conn.Close()
	done := make(chan bool, 1)
	go func() {
		buf := make([]byte, 65535)
		for range 3 {
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, remote, e := conn.ReadFromUDP(buf)
			if e != nil {
				done <- false
				return
			}
			decoder := &gosnmp.GoSNMP{MaxOids: 65535}
			request, e := decoder.SnmpDecodePacket(buf[:n])
			if e != nil {
				done <- false
				return
			}
			vars, status, index := agent.ProcessRequest(request)
			response := &gosnmp.SnmpPacket{Version: request.Version, Community: request.Community, PDUType: gosnmp.GetResponse, RequestID: request.RequestID, Variables: vars, Error: status, ErrorIndex: index}
			body, e := response.MarshalMsg()
			if e != nil {
				done <- false
				return
			}
			if _, e = conn.WriteToUDP(body, remote); e != nil {
				done <- false
				return
			}
		}
		done <- true
	}()
	if corpusExternalClient {
		address := conn.LocalAddr().String()
		for i, tool := range []string{"snmpget", "snmpgetnext", "snmpbulkget"} {
			args := []string{"-v2c", "-c", "public", "-t", "1", "-r", "0", "-On"}
			if i == 2 {
				args = append(args, "-Cn0", "-Cr4")
			}
			oid := ".1.3.6.1.2.1.1"
			if i == 0 {
				oid += ".1.0"
			}
			args = append(args, address, oid)
			cmd := exec.Command("/usr/bin/"+tool, args...)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			if cmd.Run() != nil {
				return false
			}
		}
		return <-done
	}
	client := &gosnmp.GoSNMP{Target: "127.0.0.1", Port: uint16(conn.LocalAddr().(*net.UDPAddr).Port), Community: "public", Version: gosnmp.Version2c, Timeout: time.Second, Retries: 0, MaxOids: 60}
	if client.Connect() != nil {
		return false
	}
	defer client.Conn.Close()
	a, e := client.Get([]string{"1.3.6.1.2.1.1.1.0"})
	if e != nil || len(a.Variables) != 1 {
		return false
	}
	a, e = client.GetNext([]string{"1.3.6.1.2.1.1"})
	if e != nil || len(a.Variables) != 1 {
		return false
	}
	a, e = client.GetBulk([]string{"1.3.6.1.2.1.1"}, 0, 4)
	if e != nil || len(a.Variables) == 0 {
		return false
	}
	return <-done
}
