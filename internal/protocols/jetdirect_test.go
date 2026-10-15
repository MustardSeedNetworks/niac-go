package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const jetDirectTestID = "HP LaserJet MFP M634"

// nmapPJLProbe is nmap's hp-pjl service probe, byte for byte.
const nmapPJLProbe = "\x1b%-12345X@PJL INFO ID\r\n\x1b%-12345X\r\n"

const pjlTestStatus = "@PJL INFO STATUS\r\nCODE=10001\r\nDISPLAY=\"Ready\"\r\nONLINE=TRUE\r\n\f"

func jetDirectTestPrinter() *config.Device {
	return &config.Device{
		Name:            "MED-PRN-01",
		MACAddress:      net.HardwareAddr{0x00, 0x1b, 0x78, 0x00, 0x00, 0x03},
		IPAddresses:     []net.IP{net.ParseIP("10.20.30.41")},
		JetDirectConfig: &config.JetDirectConfig{Enabled: true, PJLID: jetDirectTestID},
	}
}

func TestPJLAnswer(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		pjlID   string
		want    string
	}{
		{
			name: "nmap probe", payload: nmapPJLProbe, pjlID: jetDirectTestID,
			want: "@PJL INFO ID\r\n\"" + jetDirectTestID + "\"\r\n\f",
		},
		{name: "status, bare newline", payload: "@PJL INFO STATUS\n", pjlID: jetDirectTestID, want: pjlTestStatus},
		{
			name:    "category is case-insensitive",
			payload: "@PJL info status\r\n",
			pjlID:   jetDirectTestID,
			want:    pjlTestStatus,
		},
		{
			name: "answers in order", payload: "@PJL INFO STATUS\r\n@PJL INFO ID\r\n", pjlID: jetDirectTestID,
			want: pjlTestStatus + "@PJL INFO ID\r\n\"" + jetDirectTestID + "\"\r\n\f",
		},
		{name: "no PJL without an ID", payload: nmapPJLProbe},
		{name: "unanswered category", payload: "@PJL INFO CONFIG\r\n", pjlID: jetDirectTestID},
		{name: "prefix must be upper case", payload: "@pjl INFO ID\r\n", pjlID: jetDirectTestID},
		{name: "print data", payload: "%!PS-Adobe-3.0\nshowpage\n", pjlID: jetDirectTestID},
		{name: "trailing words", payload: "@PJL INFO ID now\r\n", pjlID: jetDirectTestID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pjlAnswer([]byte(tt.payload), tt.pjlID); got != tt.want {
				t.Errorf("pjlAnswer = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTCPHandlerServesJetDirect(t *testing.T) {
	printer := jetDirectTestPrinter()

	t.Run("SYN is accepted", func(t *testing.T) {
		reply := serviceReply(t, printer, TCPPortJetDirect, &layers.TCP{SYN: true, Seq: 500})
		if reply == nil || !reply.SYN || !reply.ACK || reply.RST || reply.Ack != 501 {
			t.Fatalf("SYN answered with %+v, want a SYN-ACK acknowledging 501", reply)
		}
	})

	t.Run("PJL query is answered", func(t *testing.T) {
		probe := []byte(nmapPJLProbe)
		reply := serviceReply(
			t,
			printer,
			TCPPortJetDirect,
			&layers.TCP{ACK: true, PSH: true, Seq: 501, Ack: 1001, Payload: probe},
		)
		want := "@PJL INFO ID\r\n\"" + jetDirectTestID + "\"\r\n\f"
		if reply == nil || string(reply.Payload) != want || reply.FIN {
			t.Fatalf("probe answered with %+v, want %q", reply, want)
		}
		if reply.Seq != 1001 || reply.Ack != 501+uint32(len(probe)) {
			t.Errorf("seq/ack = %d/%d, want 1001/%d", reply.Seq, reply.Ack, 501+len(probe))
		}
	})

	t.Run("print data is acknowledged without an answer", func(t *testing.T) {
		job := []byte("%!PS-Adobe-3.0\nshowpage\n")
		reply := serviceReply(
			t,
			printer,
			TCPPortJetDirect,
			&layers.TCP{ACK: true, PSH: true, Seq: 501, Ack: 1001, Payload: job},
		)
		if reply == nil || len(reply.Payload) != 0 || reply.Ack != 501+uint32(len(job)) {
			t.Fatalf("print data answered with %+v, want a bare ACK of %d", reply, 501+len(job))
		}
	})

	t.Run("FIN is answered with FIN", func(t *testing.T) {
		reply := serviceReply(t, printer, TCPPortJetDirect, &layers.TCP{ACK: true, FIN: true, Seq: 600, Ack: 1050})
		if reply == nil || !reply.FIN || !reply.ACK || reply.Seq != 1050 || reply.Ack != 601 {
			t.Fatalf("FIN answered with %+v, want FIN-ACK 1050/601", reply)
		}
	})

	t.Run("RST is not answered, even with data", func(t *testing.T) {
		reset := &layers.TCP{RST: true, Seq: 600, Payload: []byte(nmapPJLProbe)}
		if reply := serviceReply(t, printer, TCPPortJetDirect, reset); reply != nil {
			t.Fatalf("RST answered with %+v", reply)
		}
	})
}

// A device without JetDirect keeps the closed-port RST on 9100, and so does a
// printer whose port is switched off.
func TestTCPHandlerJetDirectClosedWithoutJetDirect(t *testing.T) {
	for _, jetDirect := range []*config.JetDirectConfig{nil, {PJLID: jetDirectTestID}} {
		device := jetDirectTestPrinter()
		device.JetDirectConfig = jetDirect

		if reply := serviceReply(
			t,
			device,
			TCPPortJetDirect,
			&layers.TCP{SYN: true, Seq: 500},
		); reply == nil ||
			!reply.RST {
			t.Errorf("JetDirect %+v: SYN answered with %+v, want RST", jetDirect, reply)
		}
		data := &layers.TCP{ACK: true, Seq: 501, Ack: 1001, Payload: []byte(nmapPJLProbe)}
		if reply := serviceReply(t, device, TCPPortJetDirect, data); reply != nil {
			t.Errorf("JetDirect %+v: data answered with %+v", jetDirect, reply)
		}
	}
}
