package snmp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// SNMPv3 (USM) authoritative-engine constants.
const (
	// timeWindowSecs is the RFC 3414 §3.2 authenticated-message time window.
	timeWindowSecs = 150

	// engineTimeWrap is 2^31; engineTime wraps here per RFC 3411.
	engineTimeWrap = int64(2147483648)

	// niacEnterprisePrefix is the 4-octet SNMP engine-ID enterprise header
	// (RFC 3411 §5): high bit of octet 0 set, low 31 bits = synthetic PEN.
	niacEngineFormatMAC = 0x03 // engine-ID format 3 = MAC address
	engineIDMinLen      = 5    // 4 enterprise octets + 1 format octet
)

// usmStats report OIDs (RFC 3414 §5).
const (
	oidUsmStatsUnknownEngineIDs = ".1.3.6.1.6.3.15.1.1.4.0"
	oidUsmStatsNotInTimeWindows = ".1.3.6.1.6.3.15.1.1.2.0"
	oidUsmStatsUnknownUserNames = ".1.3.6.1.6.3.15.1.1.3.0"
)

// ErrV3NotDiscovery indicates a v3 datagram that is neither a decodable
// authenticated request nor an engine-discovery probe (e.g. bad credentials).
var ErrV3NotDiscovery = errors.New("snmpv3: undecodable non-discovery datagram")

// ProcessFunc processes a decoded PDU and returns the response variables. It is
// the same contract as (*Agent).ProcessPDU, letting the v3 engine reuse the
// existing v1/v2c MIB machinery without depending on *Agent directly.
type ProcessFunc func(pduType gosnmp.PDUType, vars []gosnmp.SnmpPDU, nonRepeaters int, maxRepetitions uint32) []gosnmp.SnmpPDU

// v3User is a resolved USM account with gosnmp protocol enums.
type v3User struct {
	name      string
	authProto gosnmp.SnmpV3AuthProtocol
	authPass  string
	privProto gosnmp.SnmpV3PrivProtocol
	privPass  string
	msgFlags  gosnmp.SnmpV3MsgFlags // security level this user answers at
}

// V3Engine is a per-device SNMPv3 authoritative engine implementing the
// User-based Security Model (RFC 3414). It performs engine discovery, verifies
// inbound authentication, decrypts/encrypts scoped PDUs, and enforces the
// authenticated-message time window.
type V3Engine struct {
	engineID string // raw engine-ID octets (gosnmp represents these as a string)
	boots    uint32
	bootTime time.Time
	users    map[string]v3User

	unknownEngineIDs atomic.Uint32
	notInTimeWindows atomic.Uint32
	unknownUsers     atomic.Uint32
}

// NewV3Engine builds an authoritative engine from a device's SNMPv3 config.
// It returns (nil, nil) when v3 is not configured or has no users — callers
// treat a nil engine as "v3 disabled" for this device.
func NewV3Engine(cfg *config.SNMPv3Config, mac net.HardwareAddr) (*V3Engine, error) {
	if cfg == nil || !cfg.Enabled || len(cfg.Users) == 0 {
		return nil, nil //nolint:nilnil // nil engine is the documented "v3 disabled" signal
	}

	engineID, err := resolveEngineID(cfg.EngineID, mac)
	if err != nil {
		return nil, err
	}

	users := make(map[string]v3User, len(cfg.Users))
	for i := range cfg.Users {
		u, uerr := resolveUser(&cfg.Users[i])
		if uerr != nil {
			return nil, uerr
		}
		users[u.name] = u
	}

	return &V3Engine{
		engineID: string(engineID),
		boots:    1,
		bootTime: time.Now(),
		users:    users,
	}, nil
}

// engineTime returns seconds since engine boot, wrapped per RFC 3411.
func (e *V3Engine) engineTime() uint32 {
	secs := int64(time.Since(e.bootTime).Seconds())
	return uint32(secs % engineTimeWrap) //nolint:gosec // wrapped into [0,2^31)
}

// Respond decodes an inbound SNMPv3 datagram and returns the marshalled
// response bytes (a Report PDU for discovery/time-sync, or an authenticated
// GetResponse otherwise). A nil response with nil error means "silently drop".
func (e *V3Engine) Respond(req []byte, process ProcessFunc) ([]byte, error) {
	decoded, err := e.decode(req)
	if err == nil {
		return e.respondToRequest(decoded, process)
	}

	// Not an authenticated request we could decode — is it an engine-discovery
	// probe? Parse the (unauthenticated) header to find out.
	hdr, herr := parseV3Header(req)
	if herr != nil {
		return nil, fmt.Errorf("snmpv3: decode failed (%w) and header parse failed: %w", err, herr)
	}

	usm := usmOf(hdr)
	if usm == nil {
		return nil, ErrV3NotDiscovery
	}

	switch {
	case usm.AuthoritativeEngineID == "" || usm.UserName == "":
		// RFC 3414 §4: engine discovery — reply with our engine ID.
		e.unknownEngineIDs.Add(1)
		return e.buildReport(hdr, oidUsmStatsUnknownEngineIDs, e.unknownEngineIDs.Load(), gosnmp.NoAuthNoPriv, nil)
	default:
		// Known-shaped but undecodable (bad password / wrong engine ID).
		return nil, ErrV3NotDiscovery
	}
}

// respondToRequest handles a fully-decoded, authenticated request: it enforces
// the time window, then builds an authenticated (and, for authPriv, encrypted)
// GetResponse.
func (e *V3Engine) respondToRequest(req *gosnmp.SnmpPacket, process ProcessFunc) ([]byte, error) {
	usm := usmOf(req)
	if usm == nil {
		return nil, ErrV3NotDiscovery
	}

	user, ok := e.users[usm.UserName]
	if !ok {
		e.unknownUsers.Add(1)
		return e.buildReport(req, oidUsmStatsUnknownUserNames, e.unknownUsers.Load(), gosnmp.NoAuthNoPriv, &user)
	}

	// Time-window check for authenticated messages (RFC 3414 §3.2).
	if user.msgFlags&gosnmp.AuthNoPriv > 0 && !e.inTimeWindow(usm) {
		e.notInTimeWindows.Add(1)
		return e.buildReport(req, oidUsmStatsNotInTimeWindows, e.notInTimeWindows.Load(), user.msgFlags, &user)
	}

	respVars := process(req.PDUType, req.Variables, int(req.NonRepeaters), req.MaxRepetitions)

	return e.marshalResponse(req, &user, gosnmp.GetResponse, respVars)
}

// inTimeWindow reports whether an authenticated message's engine boots/time are
// within the acceptance window relative to this engine.
func (e *V3Engine) inTimeWindow(usm *gosnmp.UsmSecurityParameters) bool {
	if usm.AuthoritativeEngineBoots != e.boots {
		return false
	}
	delta := int64(e.engineTime()) - int64(usm.AuthoritativeEngineTime)
	if delta < 0 {
		delta = -delta
	}
	return delta <= timeWindowSecs
}

// buildReport marshals a Report PDU carrying a usmStats counter. When user is
// non-nil and level requires auth, the report is authenticated/encrypted.
func (e *V3Engine) buildReport(
	req *gosnmp.SnmpPacket,
	statOID string,
	count uint32,
	level gosnmp.SnmpV3MsgFlags,
	user *v3User,
) ([]byte, error) {
	vars := []gosnmp.SnmpPDU{{
		Name:  statOID,
		Type:  gosnmp.Counter32,
		Value: count,
	}}

	u := user
	if level == gosnmp.NoAuthNoPriv {
		u = nil // discovery report is unauthenticated regardless of the user
	}

	return e.marshalPacket(req, u, gosnmp.Report, level, vars)
}

// marshalResponse marshals an authenticated GetResponse for a decoded request.
func (e *V3Engine) marshalResponse(
	req *gosnmp.SnmpPacket,
	user *v3User,
	pduType gosnmp.PDUType,
	vars []gosnmp.SnmpPDU,
) ([]byte, error) {
	return e.marshalPacket(req, user, pduType, user.msgFlags, vars)
}

// marshalPacket constructs and marshals a v3 response/report. A nil user (or
// NoAuthNoPriv level) produces an unauthenticated message.
func (e *V3Engine) marshalPacket(
	req *gosnmp.SnmpPacket,
	user *v3User,
	pduType gosnmp.PDUType,
	level gosnmp.SnmpV3MsgFlags,
	vars []gosnmp.SnmpPDU,
) ([]byte, error) {
	pkt := &gosnmp.SnmpPacket{
		Version:            gosnmp.Version3,
		MsgFlags:           level &^ gosnmp.Reportable, // responses are never reportable
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: e.responseUSM(user, level),
		ContextEngineID:    e.engineID,
		ContextName:        req.ContextName,
		PDUType:            pduType,
		MsgID:              req.MsgID,
		RequestID:          req.RequestID,
		MsgMaxSize:         req.MsgMaxSize,
		Error:              gosnmp.NoError,
		Variables:          vars,
	}

	if err := pkt.SecurityParameters.InitSecurityKeys(); err != nil {
		return nil, fmt.Errorf("snmpv3: init response keys: %w", err)
	}
	if level&gosnmp.AuthPriv > gosnmp.AuthNoPriv {
		if err := pkt.SecurityParameters.InitPacket(pkt); err != nil {
			return nil, fmt.Errorf("snmpv3: init response salt: %w", err)
		}
	}

	out, err := pkt.MarshalMsg()
	if err != nil {
		return nil, fmt.Errorf("snmpv3: marshal response: %w", err)
	}
	return out, nil
}

// responseUSM builds outbound security parameters bound to this engine.
func (e *V3Engine) responseUSM(user *v3User, level gosnmp.SnmpV3MsgFlags) *gosnmp.UsmSecurityParameters {
	usm := &gosnmp.UsmSecurityParameters{
		AuthoritativeEngineID:    e.engineID,
		AuthoritativeEngineBoots: e.boots,
		AuthoritativeEngineTime:  e.engineTime(),
		AuthenticationProtocol:   gosnmp.NoAuth,
		PrivacyProtocol:          gosnmp.NoPriv,
	}
	if user == nil || level == gosnmp.NoAuthNoPriv {
		return usm
	}

	usm.UserName = user.name
	usm.AuthenticationProtocol = user.authProto
	usm.AuthenticationPassphrase = user.authPass
	if level&gosnmp.AuthPriv > gosnmp.AuthNoPriv {
		usm.PrivacyProtocol = user.privProto
		usm.PrivacyPassphrase = user.privPass
	}
	return usm
}

// decode performs the two-pass USM decode (mirrors gosnmp's own agent-side
// UnmarshalTrap path): it selects the user by msgUserName, verifies
// authentication, and decrypts the scoped PDU.
func (e *V3Engine) decode(req []byte) (*gosnmp.SnmpPacket, error) {
	table := gosnmp.NewSnmpV3SecurityParametersTable(gosnmp.Logger{})
	for _, u := range e.users {
		if err := table.Add(u.name, e.decodeUSM(u)); err != nil {
			return nil, fmt.Errorf("snmpv3: build decode table: %w", err)
		}
	}

	decoder := &gosnmp.GoSNMP{
		Version:                     gosnmp.Version3,
		SecurityModel:               gosnmp.UserSecurityModel,
		TrapSecurityParametersTable: table,
	}

	pkt, err := decoder.UnmarshalTrap(req, true)
	if err != nil {
		return nil, fmt.Errorf("snmpv3: unmarshal request: %w", err)
	}
	return pkt, nil
}

// decodeUSM builds inbound security parameters for a user. Key localization
// (InitSecurityKeys, run by the table on Add) uses our engine ID — the same
// value the manager discovered — so the derived keys match the manager's.
func (e *V3Engine) decodeUSM(u v3User) *gosnmp.UsmSecurityParameters {
	return &gosnmp.UsmSecurityParameters{
		AuthoritativeEngineID:    e.engineID,
		UserName:                 u.name,
		AuthenticationProtocol:   u.authProto,
		AuthenticationPassphrase: u.authPass,
		PrivacyProtocol:          u.privProto,
		PrivacyPassphrase:        u.privPass,
	}
}

// parseV3Header parses just the version/header/security-parameters of a v3
// datagram without needing keys — used to detect engine-discovery probes.
func parseV3Header(req []byte) (*gosnmp.SnmpPacket, error) {
	// gosnmp validates the decoder's own SecurityParameters before parsing and
	// requires a non-empty UserName; the wire values overwrite this placeholder
	// during header parsing, so it only needs to satisfy that gate.
	decoder := &gosnmp.GoSNMP{
		Version:            gosnmp.Version3,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: &gosnmp.UsmSecurityParameters{UserName: "niac-discovery"},
	}
	pkt, err := decoder.SnmpDecodePacket(req)
	if err != nil {
		return nil, fmt.Errorf("snmpv3: parse header: %w", err)
	}
	return pkt, nil
}

// usmOf extracts the USM security parameters from a decoded packet, or nil.
func usmOf(pkt *gosnmp.SnmpPacket) *gosnmp.UsmSecurityParameters {
	if pkt == nil {
		return nil
	}
	usm, _ := pkt.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	return usm
}

// resolveEngineID returns the configured engine ID (hex) or derives a stable
// MAC-based engine ID (RFC 3411 §5, format 3).
func resolveEngineID(configured string, mac net.HardwareAddr) ([]byte, error) {
	if configured != "" {
		raw, err := hex.DecodeString(strings.TrimPrefix(configured, "0x"))
		if err != nil {
			return nil, fmt.Errorf("snmpv3: invalid engine_id hex %q: %w", configured, err)
		}
		if len(raw) < engineIDMinLen {
			return nil, fmt.Errorf("snmpv3: engine_id too short (%d octets, need >= %d)", len(raw), engineIDMinLen)
		}
		return raw, nil
	}

	// Synthetic private-enterprise header (high bit set) + MAC-address format.
	id := []byte{0x80, 0x00, 0x4e, 0x1a, niacEngineFormatMAC}
	if len(mac) == snmpMACLen {
		id = append(id, mac...)
	} else {
		id = append(id, 0, 0, 0, 0, 0, 0)
	}
	return id, nil
}

// snmpMACLen is the octet length of an Ethernet MAC address.
const snmpMACLen = 6

// resolveUser maps a config user to gosnmp protocol enums and derives the
// security level the user answers at.
func resolveUser(u *config.SNMPv3User) (v3User, error) {
	authProto, err := mapAuthProtocol(u.AuthProtocol)
	if err != nil {
		return v3User{}, fmt.Errorf("snmpv3 user %q: %w", u.Username, err)
	}
	privProto, err := mapPrivProtocol(u.PrivProtocol)
	if err != nil {
		return v3User{}, fmt.Errorf("snmpv3 user %q: %w", u.Username, err)
	}

	flags := gosnmp.NoAuthNoPriv
	switch {
	case authProto != gosnmp.NoAuth && privProto != gosnmp.NoPriv:
		flags = gosnmp.AuthPriv
	case authProto != gosnmp.NoAuth:
		flags = gosnmp.AuthNoPriv
	}
	if privProto != gosnmp.NoPriv && authProto == gosnmp.NoAuth {
		return v3User{}, fmt.Errorf("snmpv3 user %q: privacy requires authentication", u.Username)
	}

	return v3User{
		name:      u.Username,
		authProto: authProto,
		authPass:  u.AuthPassword,
		privProto: privProto,
		privPass:  u.PrivPassword,
		msgFlags:  flags,
	}, nil
}

func mapAuthProtocol(name string) (gosnmp.SnmpV3AuthProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "none":
		return gosnmp.NoAuth, nil
	case "md5":
		return gosnmp.MD5, nil
	case "sha", "sha1":
		return gosnmp.SHA, nil
	case "sha224":
		return gosnmp.SHA224, nil
	case "sha256":
		return gosnmp.SHA256, nil
	case "sha384":
		return gosnmp.SHA384, nil
	case "sha512":
		return gosnmp.SHA512, nil
	default:
		return gosnmp.NoAuth, fmt.Errorf("unknown auth protocol %q", name)
	}
}

func mapPrivProtocol(name string) (gosnmp.SnmpV3PrivProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "none":
		return gosnmp.NoPriv, nil
	case "des":
		return gosnmp.DES, nil
	case "aes", "aes128":
		return gosnmp.AES, nil
	case "aes192":
		return gosnmp.AES192, nil
	case "aes256":
		return gosnmp.AES256, nil
	default:
		return gosnmp.NoPriv, fmt.Errorf("unknown privacy protocol %q", name)
	}
}
