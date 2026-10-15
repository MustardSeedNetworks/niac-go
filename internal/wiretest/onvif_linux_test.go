//go:build linux && integration

package wiretest_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// onvifPackCamera is a camera on the hospital pack's first site.
const onvifPackCamera = "MED-CAM-01"

// AI-2: a pack camera answers an unsolicited WS-Discovery Probe with a
// ProbeMatch naming the camera, from the address the Probe was sent to, back
// to the port it came from.
func TestPackCameraAnswersWSDiscoveryProbe(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	var host string
	for index := range authored.Devices {
		if device := &authored.Devices[index]; device.Name == onvifPackCamera && len(device.IPAddresses) > 0 {
			host = device.IPAddresses[0].String()
		}
	}
	if host == "" {
		t.Fatalf("the hospital pack has no addressed %s", onvifPackCamera)
	}

	conn, err := net.Dial("udp4", net.JoinHostPort(host, "3702"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	const messageID = "urn:uuid:5d1c6a0e-2b7f-4f43-8f6a-0a1b2c3d4e5f"
	probe := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>`+
		`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" `+
		`xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" `+
		`xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery" `+
		`xmlns:dn="http://www.onvif.org/ver10/network/wsdl">`+
		`<s:Header><a:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>`+
		`<a:MessageID>%s</a:MessageID>`+
		`<a:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To></s:Header>`+
		`<s:Body><d:Probe><d:Types>dn:NetworkVideoTransmitter</d:Types></d:Probe></s:Body></s:Envelope>`, messageID)

	// UDP: the pack may still be settling, so resend until it answers.
	buffer := make([]byte, 4096)
	var response string
	for attempt := 0; attempt < 10 && response == ""; attempt++ {
		if _, err = conn.Write([]byte(probe)); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if n, readErr := conn.Read(buffer); readErr == nil {
			response = string(buffer[:n])
		}
	}
	t.Logf("Probe %s:3702 answered:\n%s", host, response)
	if response == "" {
		t.Fatalf("%s did not answer the Probe", onvifPackCamera)
	}
	for _, want := range []string{
		"<wsa:RelatesTo>" + messageID + "</wsa:RelatesTo>",
		"/discovery/ProbeMatches</wsa:Action>",
		"<d:Types>dn:NetworkVideoTransmitter tds:Device</d:Types>",
		"onvif://www.onvif.org/name/AXIS%20P3265-LVE",
		"onvif://www.onvif.org/hardware/P3265-LVE",
		"<d:XAddrs>http://" + host + "/onvif/device_service</d:XAddrs>",
	} {
		if !strings.Contains(response, want) {
			t.Errorf("ProbeMatch lacks %q", want)
		}
	}
}
