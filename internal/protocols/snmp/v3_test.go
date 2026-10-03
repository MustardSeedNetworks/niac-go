package snmp

import (
	"bytes"
	"errors"
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
	return serveEngineWith(t, e, echoProcess)
}

func serveEngineWith(t *testing.T, e *V3Engine, process ProcessFunc) (int, func()) {
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
			resp, aerr := e.Respond(req, process)
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

// v3Request marshals a reportable GetRequest from the manager side, addressed
// to e's engine ID and inside its time window, as sp's user at flags.
func v3Request(
	t *testing.T,
	e *V3Engine,
	flags gosnmp.SnmpV3MsgFlags,
	sp *gosnmp.UsmSecurityParameters,
	vars ...gosnmp.SnmpPDU,
) []byte {
	t.Helper()
	if sp.AuthoritativeEngineID == "" {
		sp.AuthoritativeEngineID = e.engineID
	}
	sp.AuthoritativeEngineBoots = e.boots
	sp.AuthoritativeEngineTime = e.engineTime()
	if sp.AuthenticationProtocol == 0 {
		sp.AuthenticationProtocol = gosnmp.NoAuth
	}
	if sp.PrivacyProtocol == 0 {
		sp.PrivacyProtocol = gosnmp.NoPriv
	}
	if err := sp.InitSecurityKeys(); err != nil {
		t.Fatalf("manager keys: %v", err)
	}
	pkt := &gosnmp.SnmpPacket{
		Version:            gosnmp.Version3,
		MsgFlags:           flags | gosnmp.Reportable,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: sp,
		ContextEngineID:    e.engineID,
		PDUType:            gosnmp.GetRequest,
		MsgID:              4242,
		RequestID:          777,
		MsgMaxSize:         65507,
		Variables:          vars,
	}
	if flags&gosnmp.AuthPriv == gosnmp.AuthPriv {
		if err := sp.InitPacket(pkt); err != nil {
			t.Fatalf("manager salt: %v", err)
		}
	}
	wire, err := pkt.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return wire
}

// decodeV3NoAuth decodes an unauthenticated v3 message such as a Report.
func decodeV3NoAuth(t *testing.T, wire []byte) *gosnmp.SnmpPacket {
	t.Helper()
	// gosnmp requires a non-empty UserName before it parses; the wire value
	// replaces it.
	decoder := &gosnmp.GoSNMP{
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: &gosnmp.UsmSecurityParameters{UserName: "decoder"},
	}
	pkt, err := decoder.SnmpDecodePacket(wire)
	if err != nil {
		t.Fatalf("decode unauthenticated v3 message: %v", err)
	}
	return pkt
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
// the right username but the wrong auth passphrase cannot read a value, and the
// engine counts it as a wrong digest (RFC 3414 §3.2 step 6, #2370). gosnmp's
// manager discards the unauthenticated Report it gets back, so the Report
// itself is pinned by TestV3WrongDigestReport.
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
	if e.wrongDigests.Load() == 0 {
		t.Error("usmStatsWrongDigests did not count the wrong passphrase")
	}
}

// TestV3WrongDigestReport pins the Report itself for every digest length: it
// carries usmStatsWrongDigests.0, the counter counts each refusal, the agent
// never sees the request, and the Report answers the request's identifiers.
func TestV3WrongDigestReport(t *testing.T) {
	cases := []struct {
		name      string
		user      config.SNMPv3User
		authProto gosnmp.SnmpV3AuthProtocol
		privProto gosnmp.SnmpV3PrivProtocol
	}{
		{
			name:      "md5",
			user:      config.SNMPv3User{Username: "u", AuthProtocol: "md5", AuthPassword: "correct-pass"},
			authProto: gosnmp.MD5,
		},
		{
			name:      "sha",
			user:      config.SNMPv3User{Username: "u", AuthProtocol: "sha", AuthPassword: "correct-pass"},
			authProto: gosnmp.SHA,
		},
		{
			name:      "sha256",
			user:      config.SNMPv3User{Username: "u", AuthProtocol: "sha256", AuthPassword: "correct-pass"},
			authProto: gosnmp.SHA256,
		},
		{
			name:      "sha512",
			user:      config.SNMPv3User{Username: "u", AuthProtocol: "sha512", AuthPassword: "correct-pass"},
			authProto: gosnmp.SHA512,
		},
		{
			name: "sha256-aes",
			user: config.SNMPv3User{
				Username: "u", AuthProtocol: "sha256", AuthPassword: "correct-pass",
				PrivProtocol: "aes", PrivPassword: "privpass123",
			},
			authProto: gosnmp.SHA256,
			privProto: gosnmp.AES,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := engineFor(t, []config.SNMPv3User{tc.user})
			processed := 0
			process := func(p gosnmp.PDUType, v []gosnmp.SnmpPDU, n int, m uint32) []gosnmp.SnmpPDU {
				processed++
				return echoProcess(p, v, n, m)
			}
			flags, wantRequestID := gosnmp.AuthNoPriv, uint32(777)
			if tc.privProto != 0 {
				flags, wantRequestID = gosnmp.AuthPriv, 0 // encrypted: not extractable (RFC 3412 §7.1)
			}

			for want := uint32(1); want <= 2; want++ {
				req := v3Request(t, e, flags, &gosnmp.UsmSecurityParameters{
					UserName:                 "u",
					AuthenticationProtocol:   tc.authProto,
					AuthenticationPassphrase: "wrong-pass",
					PrivacyProtocol:          tc.privProto,
					PrivacyPassphrase:        "privpass123",
				}, gosnmp.SnmpPDU{Name: sysDescrOID, Type: gosnmp.Null})

				wire, err := e.Respond(req, process)
				if err != nil {
					t.Fatalf("Respond: %v", err)
				}
				report := decodeV3NoAuth(t, wire)
				assertReport(t, report, oidUsmStatsWrongDigests, want)
				if report.MsgID != 4242 || report.RequestID != wantRequestID {
					t.Errorf("Report msgID/request-id = %d/%d, want 4242/%d",
						report.MsgID, report.RequestID, wantRequestID)
				}
			}
			if processed != 0 {
				t.Errorf("agent processed %d unauthenticated requests", processed)
			}
		})
	}
}

// TestV3RefusalReports covers the other RFC 3414 §3.2 refusals. Before #2370
// each was a silent drop, and a request below the user's level was served.
func TestV3RefusalReports(t *testing.T) {
	authUser := config.SNMPv3User{Username: "auth", AuthProtocol: "sha", AuthPassword: "correct-pass"}
	privUser := config.SNMPv3User{
		Username: "priv", AuthProtocol: "sha", AuthPassword: "correct-pass",
		PrivProtocol: "aes", PrivPassword: "privpass123",
	}
	authSP := func(name string) *gosnmp.UsmSecurityParameters {
		return &gosnmp.UsmSecurityParameters{
			UserName: name, AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "correct-pass",
		}
	}

	cases := []struct {
		name  string
		flags gosnmp.SnmpV3MsgFlags
		sp    *gosnmp.UsmSecurityParameters
		oid   string
	}{
		{
			name:  "unknown engine ID",
			flags: gosnmp.AuthNoPriv,
			sp: &gosnmp.UsmSecurityParameters{
				AuthoritativeEngineID: "\x80\x00\x00\x00\x09other", UserName: "auth",
				AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "correct-pass",
			},
			oid: oidUsmStatsUnknownEngineIDs,
		},
		{
			name:  "unknown user",
			flags: gosnmp.AuthNoPriv,
			sp:    authSP("nobody"),
			oid:   oidUsmStatsUnknownUserNames,
		},
		{
			name:  "noAuthNoPriv for an auth user",
			flags: gosnmp.NoAuthNoPriv,
			sp:    &gosnmp.UsmSecurityParameters{UserName: "auth"},
			oid:   oidUsmStatsUnsupportedSecLevels,
		},
		{
			name:  "authNoPriv for a priv user",
			flags: gosnmp.AuthNoPriv,
			sp:    authSP("priv"),
			oid:   oidUsmStatsUnsupportedSecLevels,
		},
		{
			name:  "authPriv for an auth-only user",
			flags: gosnmp.AuthPriv,
			sp: &gosnmp.UsmSecurityParameters{
				UserName: "auth", AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "correct-pass",
				PrivacyProtocol: gosnmp.AES, PrivacyPassphrase: "privpass123",
			},
			oid: oidUsmStatsUnsupportedSecLevels,
		},
		{
			name:  "right digest, wrong privacy passphrase",
			flags: gosnmp.AuthPriv,
			sp: &gosnmp.UsmSecurityParameters{
				UserName: "priv", AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "correct-pass",
				PrivacyProtocol: gosnmp.AES, PrivacyPassphrase: "wrong-priv-pass",
			},
			oid: oidUsmStatsDecryptionErrors,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := engineFor(t, []config.SNMPv3User{authUser, privUser})
			processed := 0
			process := func(p gosnmp.PDUType, v []gosnmp.SnmpPDU, n int, m uint32) []gosnmp.SnmpPDU {
				processed++
				return echoProcess(p, v, n, m)
			}
			req := v3Request(t, e, tc.flags, tc.sp, gosnmp.SnmpPDU{Name: sysDescrOID, Type: gosnmp.Null})

			wire, err := e.Respond(req, process)
			if err != nil {
				t.Fatalf("Respond: %v", err)
			}
			assertReport(t, decodeV3NoAuth(t, wire), tc.oid, 1)
			if processed != 0 {
				t.Errorf("agent processed a refused request")
			}
		})
	}
}

// TestV3UnreportableRefusalDropped: without the reportable flag the engine
// counts the failure and sends nothing (RFC 3412 §7.1).
func TestV3UnreportableRefusalDropped(t *testing.T) {
	e := engineFor(t, []config.SNMPv3User{{Username: "admin", AuthProtocol: "sha", AuthPassword: "correct-pass"}})
	req := v3Request(t, e, gosnmp.AuthNoPriv, &gosnmp.UsmSecurityParameters{
		UserName: "admin", AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "wrong-pass",
	})
	msg, err := parseV3Message(req)
	if err != nil {
		t.Fatal(err)
	}
	flagsAt := bytes.Index(req, []byte{byte(gosnmp.OctetString), 1, byte(msg.flags)})
	if flagsAt < 0 {
		t.Fatal("msgFlags not found in the request")
	}
	req[flagsAt+2] &^= byte(gosnmp.Reportable)

	wire, err := e.Respond(req, echoProcess)
	if wire != nil || !errors.Is(err, ErrV3Dropped) {
		t.Fatalf("unreportable wrong digest: wire=%d bytes err=%v, want a drop", len(wire), err)
	}
	if got := e.wrongDigests.Load(); got != 1 {
		t.Errorf("usmStatsWrongDigests = %d, want 1", got)
	}
}

func assertReport(t *testing.T, report *gosnmp.SnmpPacket, oid string, count uint32) {
	t.Helper()
	if report.PDUType != gosnmp.Report || len(report.Variables) != 1 {
		t.Fatalf("got %v with %d varbinds, want a one-varbind Report", report.PDUType, len(report.Variables))
	}
	if v := report.Variables[0]; v.Name != oid || gosnmp.ToBigInt(v.Value).Uint64() != uint64(count) {
		t.Errorf("Report varbind = %s %v, want %s %d", v.Name, v.Value, oid, count)
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

// TestParseV3MessageTruncated: the header reader runs on unauthenticated input,
// so every truncation of a valid request must fail cleanly, never panic or
// read past the datagram.
func TestParseV3MessageTruncated(t *testing.T) {
	e := engineFor(t, []config.SNMPv3User{{Username: "admin", AuthProtocol: "sha", AuthPassword: "correct-pass"}})
	req := v3Request(t, e, gosnmp.AuthNoPriv, &gosnmp.UsmSecurityParameters{
		UserName: "admin", AuthenticationProtocol: gosnmp.SHA, AuthenticationPassphrase: "correct-pass",
	}, gosnmp.SnmpPDU{Name: sysDescrOID, Type: gosnmp.Null})

	msg, err := parseV3Message(req)
	if err != nil {
		t.Fatalf("full request: %v", err)
	}
	if msg.userName != "admin" || msg.engineID != e.engineID || msg.requestID != 777 || len(msg.authParams) != 12 {
		t.Fatalf("parsed %+v", msg)
	}
	for n := range len(req) - 1 {
		if _, perr := parseV3Message(req[:n]); perr == nil {
			t.Errorf("a %d-octet prefix of a %d-octet request parsed", n, len(req))
		}
	}
}
