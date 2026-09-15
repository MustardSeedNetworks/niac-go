package scenario_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The OIDs below are literals read off the Cisco AIR-AP1200 capture in the
// walk corpus (walks/raw/cisco/cisco-c1200-01.walk), the only real access
// point walk we have. Writing them as the constants under test is how the
// generator came to serve the PHY triple on the MAC table for a year without a
// test noticing -- the shape POWER-ETHERNET-MIB taught in G2.
const (
	captureStationID      = "1.2.840.10036.1.1.1.1"
	captureDesiredSSID    = "1.2.840.10036.1.1.1.9"
	captureMACAddress     = "1.2.840.10036.2.1.1.1"
	captureRTSThreshold   = "1.2.840.10036.2.1.1.2"
	captureShortRetry     = "1.2.840.10036.2.1.1.3"
	capturePHYType        = "1.2.840.10036.4.1.1.1"
	captureRegDomain      = "1.2.840.10036.4.1.1.2"
	capturePowerLevels    = "1.2.840.10036.4.3.1.1"
	captureTxPowerLevel1  = "1.2.840.10036.4.3.1.2"
	captureCurrentTxPower = "1.2.840.10036.4.3.1.10"
	captureCurrentChannel = "1.2.840.10036.4.5.1.1"
	captureCurrentFreq    = "1.2.840.10036.4.11.1.1"
)

// authoredDot11Columns are the objects the `wifi` block owns. A generated
// access point that also carried them as add_mibs rows would answer two
// sources for one object (#2163), and the last writer would decide which.
func authoredDot11Columns() []string {
	return []string{
		captureStationID,
		captureDesiredSSID,
		captureMACAddress,
		capturePHYType,
		capturePowerLevels,
		captureTxPowerLevel1,
		captureCurrentTxPower,
		captureCurrentChannel,
		captureCurrentFreq,
	}
}

func generatedAccessPoints(t *testing.T, packID string) []*config.Device {
	t.Helper()
	cfg := generatePack(t, packID)
	var aps []*config.Device
	for index := range cfg.Devices {
		if strings.Contains(cfg.Devices[index].Name, "-WAP-") {
			aps = append(aps, &cfg.Devices[index])
		}
	}
	if len(aps) == 0 {
		t.Fatalf("%s generated no access points", packID)
	}

	return aps
}

// TestGeneratedAccessPointsAuthorTheirRadios is the pack half of W1: every
// access point in every built-in pack states its own radios, so the Wi-Fi
// tier stops being scenery.
func TestGeneratedAccessPointsAuthorTheirRadios(t *testing.T) {
	bssids := make(map[string]string)
	for _, pack := range scenario.Packs() {
		for _, device := range generatedAccessPoints(t, pack.ID) {
			if device.WiFiConfig == nil {
				t.Fatalf("%s authors no radios", device.Name)
			}
			radios := device.WiFiConfig.Radios
			if len(radios) != 4 {
				t.Fatalf("%s authors %d radios, want 4", device.Name, len(radios))
			}
			for index, radio := range radios {
				assertAuthoredRadio(t, device, index, radio, bssids)
			}
		}
	}
}

func assertAuthoredRadio(
	t *testing.T,
	device *config.Device,
	index int,
	radio config.WiFiRadio,
	bssids map[string]string,
) {
	t.Helper()
	if name := fmt.Sprintf("Dot11Radio%d", index); radio.Interface != name {
		t.Errorf("%s radio %d names interface %q, want %q",
			device.Name, index, radio.Interface, name)
	}
	if want := scenario.APRadioBandForTest(index); radio.Band != want {
		t.Errorf("%s radio %d band = %q, want %q", device.Name, index, radio.Band, want)
	}
	if radio.SSID == "" || radio.Channel == 0 || radio.TxPowerDBM == 0 {
		t.Errorf("%s radio %d is incomplete: %+v", device.Name, index, radio)
	}
	if owner, taken := bssids[radio.BSSID]; taken {
		t.Errorf("BSSID %s belongs to both %s and %s", radio.BSSID, owner, device.Name)
	}
	bssids[radio.BSSID] = device.Name
}

// TestGeneratedAccessPointsServeOneSourcePerDot11Object is #2163's acceptance
// stated as a test: no add_mibs row may restate an object the authored radio
// already answers.
func TestGeneratedAccessPointsServeOneSourcePerDot11Object(t *testing.T) {
	for _, pack := range scenario.Packs() {
		for _, device := range generatedAccessPoints(t, pack.ID) {
			for _, mib := range device.SNMPConfig.AddMibs {
				oid := strings.TrimPrefix(mib.OID, ".")
				for _, column := range authoredDot11Columns() {
					if strings.HasPrefix(oid, column+".") {
						t.Errorf("%s restates %s as an add_mibs row (%s)",
							device.Name, column, oid)
					}
				}
			}
		}
	}
}

// TestGeneratedAccessPointRadiosAnswerTheCaptureShape walks a real agent built
// from a generated access point. The add_mibs rows and the authored radios are
// merged by the agent, so this is the only place that proves which one a
// consumer actually reads.
func TestGeneratedAccessPointRadiosAnswerTheCaptureShape(t *testing.T) {
	device := generatedAccessPoints(t, "hospital")[0]
	agent := agentAsTheStackBuildsIt(t, device)
	radios := device.WiFiConfig.Radios

	for index, radio := range radios {
		suffix := "." + radioIfIndex(t, agent, device, radio.Interface)
		wantMACValue(t, agent, captureStationID+suffix, radio.BSSID)
		wantMACValue(t, agent, captureMACAddress+suffix, radio.BSSID)
		wantValue(t, agent, captureDesiredSSID+suffix, radio.SSID)

		// The band decides which table carries the channel and which PHY the
		// radio reports: the capture's own 2.4 GHz radio answered dsss(2) and
		// dot11CurrentChannel, its 5 GHz one ofdm(4) and dot11CurrentFrequency,
		// and neither answered the other's.
		channelOID, otherOID, phy := captureCurrentChannel, captureCurrentFreq, "2"
		if radio.Band != "2.4GHz" {
			channelOID, otherOID, phy = captureCurrentFreq, captureCurrentChannel, "4"
		}
		wantValue(t, agent, capturePHYType+suffix, phy)
		wantValue(t, agent, channelOID+suffix, strconv.Itoa(radio.Channel))
		if got, ok := agentValue(agent, otherOID+suffix); ok {
			t.Errorf("%s radio %d (%s) answers %s = %s",
				device.Name, index, radio.Band, otherOID, got)
		}

		// The rows the block does not carry still come from the capture, at
		// the table the capture put them on.
		wantValue(t, agent, captureRTSThreshold+suffix, "2312")
		wantValue(t, agent, captureShortRetry+suffix, "64")
		wantValue(t, agent, captureRegDomain+suffix, "16")
	}
}

// agentAsTheStackBuildsIt mirrors internal/protocols/stack_snmp.go: the agent
// is constructed, and then the authored add_mibs rows are applied over it. The
// order is the point -- an add_mibs row that restated an authored radio's
// object would win, silently.
func agentAsTheStackBuildsIt(t *testing.T, device *config.Device) *snmp.Agent {
	t.Helper()
	agent := snmp.NewAgent(device, 0)
	for _, mib := range device.SNMPConfig.AddMibs {
		if err := agent.AddMib(mib.OID, mib.Type, mib.Value); err != nil {
			t.Fatalf("%s add_mibs %s: %v", device.Name, mib.OID, err)
		}
	}

	return agent
}

func radioIfIndex(t *testing.T, agent *snmp.Agent, device *config.Device, name string) string {
	t.Helper()
	const ifDescr = "1.3.6.1.2.1.2.2.1.2"
	for index := range device.Interfaces {
		suffix := strconv.Itoa(index + 1)
		if value, ok := agentValue(agent, ifDescr+"."+suffix); ok && value == name {
			return suffix
		}
	}
	t.Fatalf("%s reports no ifIndex for %s", device.Name, name)

	return ""
}

// agentValue reads one object the way a manager does -- through the agent's own
// GET path, after the add_mibs rows and the authored radios have been merged.
func agentValue(agent *snmp.Agent, oid string) (string, bool) {
	value, err := agent.HandleGet(oid)
	if err != nil || value == nil {
		return "", false
	}
	if bytes, isBytes := value.Value.([]byte); isBytes {
		return string(bytes), true
	}

	return fmt.Sprint(value.Value), true
}

func wantValue(t *testing.T, agent *snmp.Agent, oid, want string) {
	t.Helper()
	got, ok := agentValue(agent, oid)
	if !ok || got != want {
		t.Errorf("%s = %q (present %t), want %q", oid, got, ok, want)
	}
}

func wantMACValue(t *testing.T, agent *snmp.Agent, oid, want string) {
	t.Helper()
	got, ok := agentValue(agent, oid)
	if !ok {
		t.Errorf("%s is absent, want MAC %s", oid, want)

		return
	}
	address := make([]string, 0, len(got))
	for _, octet := range []byte(got) {
		address = append(address, fmt.Sprintf("%02x", octet))
	}
	if joined := strings.Join(address, ":"); !strings.EqualFold(joined, want) {
		t.Errorf("%s = %s, want MAC %s", oid, joined, want)
	}
}
