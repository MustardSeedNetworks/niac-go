package snmp

import (
	"net"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// sysDescrOID is the value the fake agent returns for any Get in these tests.
const sysDescrOID = ".1.3.6.1.2.1.1.1.0"

// echoProcess is a minimal ProcessFunc: it answers a Get for sysDescr with a
// fixed OctetString and NoSuchObject for anything else.
func echoProcess(_ gosnmp.PDUType, vars []gosnmp.SnmpPDU, _ int, _ uint32) []gosnmp.SnmpPDU {
	out := make([]gosnmp.SnmpPDU, 0, len(vars))
	for _, v := range vars {
		if v.Name == sysDescrOID {
			out = append(out, gosnmp.SnmpPDU{Name: sysDescrOID, Type: gosnmp.OctetString, Value: []byte("niac-sim")})
			continue
		}
		out = append(out, gosnmp.SnmpPDU{Name: v.Name, Type: gosnmp.NoSuchObject, Value: nil})
	}
	return out
}

// serveEngine runs an SNMPv3 responder on a loopback UDP socket driven by the
// engine, returning the bound port and a stop func. It is the local stand-in
// for the CT304 agent socket.
func serveEngine(t *testing.T, e *V3Engine) (int, func()) {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 64*1024)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, rerr := conn.ReadFromUDP(buf)
			if rerr != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			req := make([]byte, n)
			copy(req, buf[:n])
			resp, aerr := e.Respond(req, echoProcess)
			if aerr != nil || resp == nil {
				continue
			}
			_, _ = conn.WriteToUDP(resp, addr)
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).Port, func() {
		close(done)
		_ = conn.Close()
	}
}

// newClient builds a gosnmp v3 manager pointed at the loopback responder.
func newClient(port int, flags gosnmp.SnmpV3MsgFlags, sp *gosnmp.UsmSecurityParameters) *gosnmp.GoSNMP {
	return &gosnmp.GoSNMP{
		Target:             "127.0.0.1",
		Port:               uint16(port),
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           flags,
		SecurityParameters: sp,
		Timeout:            2 * time.Second,
		Retries:            2,
	}
}

func engineFor(t *testing.T, users []config.SNMPv3User) *V3Engine {
	t.Helper()
	mac, _ := net.ParseMAC("02:00:00:11:22:33")
	e, err := NewV3Engine(&config.SNMPv3Config{Enabled: true, Users: users}, mac)
	if err != nil {
		t.Fatalf("NewV3Engine: %v", err)
	}
	if e == nil {
		t.Fatal("expected engine, got nil")
	}
	return e
}

// TestV3RoundTripAuthPriv is the headline proof: a real gosnmp manager performs
// engine discovery and an authPriv (SHA + AES) Get end-to-end against the
// engine, and reads back the value the agent produced.
func TestV3RoundTripAuthPriv(t *testing.T) {
	e := engineFor(t, []config.SNMPv3User{{
		Username:     "admin",
		AuthProtocol: "sha",
		AuthPassword: "authpass123",
		PrivProtocol: "aes",
		PrivPassword: "privpass123",
	}})
	port, stop := serveEngine(t, e)
	defer stop()

	client := newClient(port, gosnmp.AuthPriv, &gosnmp.UsmSecurityParameters{
		UserName:                 "admin",
		AuthenticationProtocol:   gosnmp.SHA,
		AuthenticationPassphrase: "authpass123",
		PrivacyProtocol:          gosnmp.AES,
		PrivacyPassphrase:        "privpass123",
	})
	if err := client.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = client.Conn.Close() }()

	result, err := client.Get([]string{sysDescrOID})
	if err != nil {
		t.Fatalf("authPriv Get: %v", err)
	}
	if len(result.Variables) != 1 {
		t.Fatalf("want 1 var, got %d", len(result.Variables))
	}
	if got := string(result.Variables[0].Value.([]byte)); got != "niac-sim" {
		t.Errorf("value = %q, want niac-sim", got)
	}
}

// TestV3RoundTripLevels covers noAuthNoPriv, authNoPriv (SHA), and authPriv with
// DES to exercise both cipher paths and all three security levels.
func TestV3RoundTripLevels(t *testing.T) {
	cases := []struct {
		name  string
		user  config.SNMPv3User
		flags gosnmp.SnmpV3MsgFlags
		sp    *gosnmp.UsmSecurityParameters
	}{
		{
			name:  "noAuthNoPriv",
			user:  config.SNMPv3User{Username: "noauth"},
			flags: gosnmp.NoAuthNoPriv,
			sp:    &gosnmp.UsmSecurityParameters{UserName: "noauth"},
		},
		{
			name:  "authNoPriv-md5",
			user:  config.SNMPv3User{Username: "authonly", AuthProtocol: "md5", AuthPassword: "authpass123"},
			flags: gosnmp.AuthNoPriv,
			sp: &gosnmp.UsmSecurityParameters{
				UserName:                 "authonly",
				AuthenticationProtocol:   gosnmp.MD5,
				AuthenticationPassphrase: "authpass123",
			},
		},
		{
			name: "authPriv-des",
			user: config.SNMPv3User{
				Username: "full", AuthProtocol: "sha", AuthPassword: "authpass123",
				PrivProtocol: "des", PrivPassword: "privpass123",
			},
			flags: gosnmp.AuthPriv,
			sp: &gosnmp.UsmSecurityParameters{
				UserName:                 "full",
				AuthenticationProtocol:   gosnmp.SHA,
				AuthenticationPassphrase: "authpass123",
				PrivacyProtocol:          gosnmp.DES,
				PrivacyPassphrase:        "privpass123",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := engineFor(t, []config.SNMPv3User{tc.user})
			port, stop := serveEngine(t, e)
			defer stop()

			client := newClient(port, tc.flags, tc.sp)
			if err := client.Connect(); err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer func() { _ = client.Conn.Close() }()

			result, err := client.Get([]string{sysDescrOID})
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got := string(result.Variables[0].Value.([]byte)); got != "niac-sim" {
				t.Errorf("value = %q, want niac-sim", got)
			}
		})
	}
}

// TestV3WrongPasswordRejected proves the engine authenticates: a manager with
// the right username but wrong auth passphrase cannot read a value.
func TestV3WrongPasswordRejected(t *testing.T) {
	e := engineFor(t, []config.SNMPv3User{{
		Username: "admin", AuthProtocol: "sha", AuthPassword: "correct-pass",
	}})
	port, stop := serveEngine(t, e)
	defer stop()

	client := newClient(port, gosnmp.AuthNoPriv, &gosnmp.UsmSecurityParameters{
		UserName:                 "admin",
		AuthenticationProtocol:   gosnmp.SHA,
		AuthenticationPassphrase: "wrong-pass",
	})
	if err := client.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = client.Conn.Close() }()

	if _, err := client.Get([]string{sysDescrOID}); err == nil {
		t.Fatal("expected auth failure for wrong passphrase, got success")
	}
}

// TestNewV3EngineDisabled confirms the nil-engine "v3 disabled" contract.
func TestNewV3EngineDisabled(t *testing.T) {
	mac, _ := net.ParseMAC("02:00:00:11:22:33")
	for _, cfg := range []*config.SNMPv3Config{
		nil,
		{Enabled: false, Users: []config.SNMPv3User{{Username: "x"}}},
		{Enabled: true},
	} {
		e, err := NewV3Engine(cfg, mac)
		if err != nil {
			t.Fatalf("NewV3Engine(%v): %v", cfg, err)
		}
		if e != nil {
			t.Errorf("NewV3Engine(%v) = engine, want nil", cfg)
		}
	}
}

// TestEngineIDFromConfigHex validates explicit engine-ID parsing.
func TestEngineIDFromConfigHex(t *testing.T) {
	raw, err := resolveEngineID("8000000001020304", nil)
	if err != nil {
		t.Fatalf("resolveEngineID: %v", err)
	}
	if len(raw) != 8 {
		t.Errorf("engine ID len = %d, want 8", len(raw))
	}
	if _, badErr := resolveEngineID("zz", nil); badErr == nil {
		t.Error("expected error for invalid hex engine ID")
	}
}
