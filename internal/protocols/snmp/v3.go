package snmp

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sort"
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
	oidUsmStatsUnsupportedSecLevels = ".1.3.6.1.6.3.15.1.1.1.0"
	oidUsmStatsNotInTimeWindows     = ".1.3.6.1.6.3.15.1.1.2.0"
	oidUsmStatsUnknownUserNames     = ".1.3.6.1.6.3.15.1.1.3.0"
	oidUsmStatsUnknownEngineIDs     = ".1.3.6.1.6.3.15.1.1.4.0"
	oidUsmStatsWrongDigests         = ".1.3.6.1.6.3.15.1.1.5.0"
	oidUsmStatsDecryptionErrors     = ".1.3.6.1.6.3.15.1.1.6.0"
)

// ErrV3Dropped indicates a v3 datagram the engine refuses without a Report:
// the sender did not set the reportable flag (RFC 3412 §7.1), or no usmStats
// counter describes the failure.
var ErrV3Dropped = errors.New("snmpv3: datagram dropped")

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
	msgFlags  gosnmp.SnmpV3MsgFlags // the one security level this user answers at
	authKey   []byte                // authentication key localized to the engine
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

	unknownEngineIDs     atomic.Uint32
	unknownUsers         atomic.Uint32
	unsupportedSecLevels atomic.Uint32
	wrongDigests         atomic.Uint32
	notInTimeWindows     atomic.Uint32
	decryptionErrors     atomic.Uint32
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

	e := &V3Engine{
		engineID: string(engineID),
		boots:    1,
		bootTime: time.Now(),
		users:    make(map[string]v3User, len(cfg.Users)),
	}
	for i := range cfg.Users {
		u, uerr := resolveUser(&cfg.Users[i])
		if uerr != nil {
			return nil, uerr
		}
		if u.authProto != gosnmp.NoAuth {
			usm := e.decodeUSM(u)
			if kerr := usm.InitSecurityKeys(); kerr != nil {
				return nil, fmt.Errorf("snmpv3 user %q: localize keys: %w", u.name, kerr)
			}
			u.authKey = usm.SecretKey
		}
		e.users[u.name] = u
	}

	return e, nil
}

// engineTime returns seconds since engine boot, wrapped per RFC 3411.
func (e *V3Engine) engineTime() uint32 {
	secs := int64(time.Since(e.bootTime).Seconds())
	return uint32(secs % engineTimeWrap) //nolint:gosec // wrapped into [0,2^31)
}

// Respond processes an inbound SNMPv3 datagram in RFC 3414 §3.2 order and
// returns the marshalled response bytes: a usmStats Report when the request
// fails a USM check (engine discovery is the first of these), or an
// authenticated GetResponse. A nil response with an error means "drop".
func (e *V3Engine) Respond(req []byte, process ProcessFunc) ([]byte, error) {
	msg, err := parseV3Message(req)
	if err != nil {
		return nil, err
	}

	// Engine discovery (RFC 3414 §4) and a stale engine ID get the same
	// answer: a Report carrying this engine's ID.
	if msg.engineID == "" || msg.userName == "" || msg.engineID != e.engineID {
		return e.report(msg, oidUsmStatsUnknownEngineIDs, &e.unknownEngineIDs)
	}

	user, ok := e.users[msg.userName]
	if !ok {
		return e.report(msg, oidUsmStatsUnknownUserNames, &e.unknownUsers)
	}
	// A user answers at exactly its configured level. Below it, a request
	// would be served without the authentication the user demands; above
	// it, the engine holds no key to verify or decrypt with.
	if msg.flags&gosnmp.AuthPriv != user.msgFlags {
		return e.report(msg, oidUsmStatsUnsupportedSecLevels, &e.unsupportedSecLevels)
	}
	if user.authKey != nil && !authentic(msg, &user) {
		return e.report(msg, oidUsmStatsWrongDigests, &e.wrongDigests)
	}

	decoded, err := e.decode(req)
	if err != nil {
		if user.privProto != gosnmp.NoPriv {
			// The digest verified, so the privacy key or the ciphertext is wrong.
			return e.report(msg, oidUsmStatsDecryptionErrors, &e.decryptionErrors)
		}
		return nil, err
	}

	return e.respondToRequest(decoded, &user, process)
}

// respondToRequest handles a fully-decoded request from a known user at its
// configured level: it enforces the time window, then builds an authenticated
// (and, for authPriv, encrypted) GetResponse.
func (e *V3Engine) respondToRequest(req *gosnmp.SnmpPacket, user *v3User, process ProcessFunc) ([]byte, error) {
	if user.msgFlags&gosnmp.AuthNoPriv > 0 && !e.inTimeWindow(usmOf(req)) {
		count := e.notInTimeWindows.Add(1)
		return e.buildReport(req, oidUsmStatsNotInTimeWindows, count, user.msgFlags, user)
	}

	respVars := process(req.PDUType, req.Variables, int(req.NonRepeaters), req.MaxRepetitions)

	return e.marshalResponse(req, user, gosnmp.GetResponse, respVars)
}

// report counts a usmStats event and, when the request asked for a report
// (RFC 3412 §7.1), returns the unauthenticated Report carrying the counter.
func (e *V3Engine) report(msg *v3Message, statOID string, counter *atomic.Uint32) ([]byte, error) {
	count := counter.Add(1)
	if msg.flags&gosnmp.Reportable == 0 {
		return nil, fmt.Errorf("%w: unreportable request (%s)", ErrV3Dropped, statOID)
	}
	return e.buildReport(msg.reportTo(), statOID, count, gosnmp.NoAuthNoPriv, nil)
}

// authentic reports whether msg's digest verifies under user's localized key
// (HMAC-MD5/SHA-96 per RFC 3414 §6.3.2 and §7.3.2, HMAC-SHA-2 per RFC 7860
// §4.2.2): the MAC over the whole message with the digest field zeroed.
func authentic(msg *v3Message, user *v3User) bool {
	if len(msg.authParams) != macLen(user.authProto) {
		return false
	}
	received := bytes.Clone(msg.authParams)
	clear(msg.authParams)
	mac := hmac.New(user.authProto.HashType().New, user.authKey)
	mac.Write(msg.raw)
	return hmac.Equal(mac.Sum(nil)[:len(received)], received)
}

// Truncated digest lengths on the wire (RFC 3414 §6.3.1 and §7.3.1, RFC 7860
// §4.1).
const (
	macLenHMAC96 = 12
	macLenSHA224 = 16
	macLenSHA256 = 24
	macLenSHA384 = 32
	macLenSHA512 = 48
)

// macLen is the truncated digest length each protocol puts on the wire.
func macLen(proto gosnmp.SnmpV3AuthProtocol) int {
	switch proto {
	case gosnmp.MD5, gosnmp.SHA:
		return macLenHMAC96
	case gosnmp.SHA224:
		return macLenSHA224
	case gosnmp.SHA256:
		return macLenSHA256
	case gosnmp.SHA384:
		return macLenSHA384
	case gosnmp.SHA512:
		return macLenSHA512
	case gosnmp.NoAuth:
		return 0
	}
	return 0
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

// MarshalNotification builds an authenticated (and, where the user has a
// privacy protocol, encrypted) v3 trap or inform.
//
// A v2c trap carries a community string in the clear, so a manager configured
// for v3-only rejects it and a NIAC device that only sends v2c is silent to
// that manager. The engine that answers this device's queries is also its
// notification originator, so the same engine ID and boots/time are
// authoritative here — which is what lets a receiver validate the message
// without a prior discovery exchange.
func (e *V3Engine) MarshalNotification(
	username string,
	pduType gosnmp.PDUType,
	requestID uint32,
	variables []gosnmp.SnmpPDU,
) ([]byte, error) {
	if e == nil {
		return nil, ErrV3Disabled
	}

	user := e.notificationUser(username)
	if user == nil {
		return nil, fmt.Errorf("%w: %q", ErrV3UnknownUser, username)
	}

	level := user.msgFlags
	pkt := &gosnmp.SnmpPacket{
		Version:       gosnmp.Version3,
		MsgFlags:      level,
		SecurityModel: gosnmp.UserSecurityModel,
		// An inform is acknowledged, so it must be reportable for the receiver
		// to be able to answer at all; a trap is fire-and-forget.
		SecurityParameters: e.responseUSM(user, level),
		ContextEngineID:    e.engineID,
		PDUType:            pduType,
		MsgID:              requestID,
		RequestID:          requestID,
		MsgMaxSize:         snmpMaxMessageSize,
		Error:              gosnmp.NoError,
		Variables:          variables,
	}
	if pduType == gosnmp.InformRequest {
		pkt.MsgFlags |= gosnmp.Reportable
	}

	if err := pkt.SecurityParameters.InitSecurityKeys(); err != nil {
		return nil, fmt.Errorf("snmpv3: init notification keys: %w", err)
	}
	if level&gosnmp.AuthPriv > gosnmp.AuthNoPriv {
		if err := pkt.SecurityParameters.InitPacket(pkt); err != nil {
			return nil, fmt.Errorf("snmpv3: init notification salt: %w", err)
		}
	}

	out, err := pkt.MarshalMsg()
	if err != nil {
		return nil, fmt.Errorf("snmpv3: marshal notification: %w", err)
	}

	return out, nil
}

// notificationUser picks the USM user a notification is sent as.
//
// With no name configured it takes the lexicographically first user rather
// than "the first one", which a map has no notion of: an arbitrary pick would
// send the same config as a different user between runs, and a receiver
// validating by user name would see a device that changes identity.
func (e *V3Engine) notificationUser(username string) *v3User {
	if username != "" {
		user, known := e.users[username]
		if !known {
			return nil
		}

		return &user
	}

	names := make([]string, 0, len(e.users))
	for name := range e.users {
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	user := e.users[names[0]]

	return &user
}

// snmpMaxMessageSize is the message size a v3 notification advertises. RFC 3412
// sets 484 as the floor every implementation must accept; this is the
// conventional larger value managers use.
const snmpMaxMessageSize = 65507
