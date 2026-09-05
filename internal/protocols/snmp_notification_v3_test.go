package protocols_test

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// authPrivUser is a USM user with both authentication and privacy, the level a
// manager configured for v3-only requires.
func authPrivUser() config.SNMPv3User {
	return config.SNMPv3User{
		Username:     "niac-notify",
		AuthProtocol: "sha256",
		AuthPassword: "authpassphrase",
		PrivProtocol: "aes",
		PrivPassword: "privpassphrase",
	}
}

// notifyEngine builds a device engine with one authPriv user.
func notifyEngine(t *testing.T) *snmp.V3Engine {
	t.Helper()

	engine, err := snmp.NewV3Engine(
		&config.SNMPv3Config{Enabled: true, Users: []config.SNMPv3User{authPrivUser()}},
		[]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x01},
	)
	if err != nil {
		t.Fatalf("NewV3Engine: %v", err)
	}
	if engine == nil {
		t.Fatal("engine is nil with v3 enabled")
	}

	return engine
}

// receiverFor builds the decoder a trap receiver would use: the same user, and
// nothing else about the sender known in advance.
func receiverFor(t *testing.T) *gosnmp.GoSNMP {
	t.Helper()

	table := gosnmp.NewSnmpV3SecurityParametersTable(gosnmp.Logger{})
	user := authPrivUser()
	err := table.Add(user.Username, &gosnmp.UsmSecurityParameters{
		UserName:                 user.Username,
		AuthenticationProtocol:   gosnmp.SHA256,
		AuthenticationPassphrase: user.AuthPassword,
		PrivacyProtocol:          gosnmp.AES,
		PrivacyPassphrase:        user.PrivPassword,
	})
	if err != nil {
		t.Fatalf("build receiver table: %v", err)
	}

	return &gosnmp.GoSNMP{
		Version:                     gosnmp.Version3,
		SecurityModel:               gosnmp.UserSecurityModel,
		TrapSecurityParametersTable: table,
	}
}

// A v2c trap carries its community in the clear, so a manager configured for
// v3-only rejects it and a device that can only send v2c is silent to that
// manager. The test is a decode by the receiver's own code path, not an
// inspection of our bytes: authentication and decryption either work against
// the configured credentials or they do not.
func TestV3TrapIsAuthenticatedAndEncrypted(t *testing.T) {
	engine := notifyEngine(t)

	payload, err := engine.MarshalNotification("niac-notify", gosnmp.SNMPv2Trap, 42, []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: ".1.3.6.1.6.3.1.1.5.3"},
	})
	if err != nil {
		t.Fatalf("MarshalNotification: %v", err)
	}

	decoded, err := receiverFor(t).UnmarshalTrap(payload, true)
	if err != nil {
		t.Fatalf("the receiver could not validate the trap: %v", err)
	}
	if decoded == nil {
		t.Fatal("receiver decoded no packet")
	}
	if decoded.Version != gosnmp.Version3 {
		t.Errorf("version = %v, want v3", decoded.Version)
	}
	if decoded.MsgFlags&gosnmp.AuthPriv != gosnmp.AuthPriv {
		t.Errorf("msgFlags = %v, want authPriv", decoded.MsgFlags)
	}

	// The variables only survive if decryption worked; a failed decrypt yields
	// an empty or garbled scoped PDU rather than an error in every case.
	if len(decoded.Variables) == 0 {
		t.Fatal("no variables survived the decode, so the scoped PDU did not decrypt")
	}
	if got := decoded.Variables[0].Name; got != ".1.3.6.1.6.3.1.1.4.1.0" {
		t.Errorf("first variable = %q, want the trap OID", got)
	}
}

// An inform must be reportable, or the receiver has no basis to answer it.
func TestV3InformIsReportable(t *testing.T) {
	engine := notifyEngine(t)

	payload, err := engine.MarshalNotification("niac-notify", gosnmp.InformRequest, 7, nil)
	if err != nil {
		t.Fatalf("MarshalNotification: %v", err)
	}
	decoded, err := receiverFor(t).UnmarshalTrap(payload, true)
	if err != nil {
		t.Fatalf("the receiver could not validate the inform: %v", err)
	}
	if decoded.PDUType != gosnmp.InformRequest {
		t.Errorf("PDU type = %v, want InformRequest", decoded.PDUType)
	}
	if decoded.MsgFlags&gosnmp.Reportable == 0 {
		t.Error("the inform is not reportable, so the receiver cannot answer it")
	}
}

// A notification addressed to a user the device does not have is refused rather
// than sent unauthenticated, which a receiver would drop anyway.
func TestV3NotificationRefusesAnUnknownUser(t *testing.T) {
	if _, err := notifyEngine(t).MarshalNotification("nobody", gosnmp.SNMPv2Trap, 1, nil); err == nil {
		t.Fatal("MarshalNotification accepted an unconfigured user")
	}
}

// With no user named, the same config must always send as the same user. The
// engine holds users in a map, so an arbitrary pick would change identity
// between runs and a receiver validating by user name would see a device that
// keeps changing.
func TestV3NotificationUserChoiceIsStable(t *testing.T) {
	engine, err := snmp.NewV3Engine(&config.SNMPv3Config{
		Enabled: true,
		Users: []config.SNMPv3User{
			{Username: "zulu", AuthProtocol: "sha", AuthPassword: "zulupassphrase"},
			{Username: "alpha", AuthProtocol: "sha", AuthPassword: "alphapassphrase"},
			{Username: "mike", AuthProtocol: "sha", AuthPassword: "mikepassphrase"},
		},
	}, []byte{0x02, 0, 0, 0, 0, 2})
	if err != nil {
		t.Fatalf("NewV3Engine: %v", err)
	}

	first, err := engine.MarshalNotification("", gosnmp.SNMPv2Trap, 1, nil)
	if err != nil {
		t.Fatalf("MarshalNotification: %v", err)
	}
	for range 20 {
		again, marshalErr := engine.MarshalNotification("", gosnmp.SNMPv2Trap, 1, nil)
		if marshalErr != nil {
			t.Fatalf("MarshalNotification: %v", marshalErr)
		}
		if string(again) != string(first) {
			t.Fatal("the same config sent the notification as a different user")
		}
	}
}
