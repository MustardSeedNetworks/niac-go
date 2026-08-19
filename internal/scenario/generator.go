package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

const (
	// congestionWarningPercent is the utilization at which Link-Live raises an
	// interface Warning; a pack authors its trouble spots at or above it.
	congestionWarningPercent = 80

	// neighborSettleCycles is how many advertisement intervals a consumer waits
	// before a neighbour table is trustworthy: one to transmit, one to prove
	// nothing further arrives.
	neighborSettleCycles = 2

	transitSubnet    = "10.254.200.0/24"
	transitGateway   = "10.254.200.1"
	internetLoopback = "8.8.8.8"
)

// ErrInvalidRequest identifies authoring input that cannot produce a valid scenario.
var ErrInvalidRequest = errors.New("invalid scenario request")

type vlanDefinition struct {
	id         int
	slug       string
	thirdOctet int
}

func vlanDefinitions() []vlanDefinition {
	return []vlanDefinition{
		{vlanManagement, "mgmt", vlanManagement},
		{vlanData, "data", vlanData},
		{vlanWiFiCorp, "wifi-corp", vlanWiFiCorp},
		{vlanWiFiGuest, "wifi-guest", vlanWiFiGuest},
		{vlanServers, "servers", vlanServers},
		{vlanVoiceIoT, "voice-iot", vlanVoiceIoT},
	}
}

// Generate builds and validates one portable scenario. It has no filesystem,
// runtime, or network side effects.
func Generate(request Request) (Result, error) {
	if err := validateRequest(request); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	links := buildLinks(request)
	authored := converter.Config{
		Networks: buildNetworks(request),
		Attachments: []converter.LogicalAttachment{{
			Name: request.AttachmentName, Connect: "lab-transit",
		}},
		Devices: buildDevices(request, links),
	}
	if err := applyCongestion(&authored, request.Congestion); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}
	if err := converter.ValidateConfig(&authored); err != nil {
		return Result{}, fmt.Errorf("generated authoring config: %w", err)
	}

	data, err := yaml.Marshal(authored)
	if err != nil {
		return Result{}, fmt.Errorf("marshal generated scenario: %w", err)
	}
	runtimeConfig, err := config.LoadYAMLBytes(data)
	if err != nil {
		return Result{}, fmt.Errorf("load generated scenario: %w", err)
	}
	validation := config.NewValidator("generated-scenario.yaml").Validate(runtimeConfig)
	if !validation.Valid {
		return Result{}, errors.New(validation.Format())
	}

	identity, err := buildIdentity(request)
	if err != nil {
		return Result{}, err
	}
	manifest := buildManifest(&authored)
	manifest.Identity = identity

	return Result{YAML: data, Manifest: manifest}, nil
}

func buildDevices(request Request, links linkMap) []converter.Device {
	devices := []converter.Device{
		labEdge(request, links),
		providerWANRouter(request, 1, links),
		providerWANRouter(request, secondProviderWANRouterIndex, links),
	}
	for siteIndex := range request.Sites {
		site := request.Sites[siteIndex]
		for index := 1; index <= request.Counts.SiteWANRouters; index++ {
			devices = append(devices, managedDevice(request, siteWANRouter(site, siteIndex, index), links))
		}
		for index := 1; index <= request.Counts.Firewalls; index++ {
			devices = append(devices, managedDevice(request, siteFirewall(site, siteIndex, index), links))
		}
		devices = append(devices, buildSiteLAN(request, site, siteIndex, links)...)
		devices = append(devices, buildSiteEndpoints(request, site, links)...)
	}
	setLabEdgeDNS(devices, request.Domain)
	return devices
}

func buildNetworks(request Request) []converter.Network {
	networks := []converter.Network{
		{Name: "lab-transit", Subnet: transitSubnet, VirtualVLAN: vlanManagement},
		{Name: "lab-wan", Subnet: "203.0.113.0/29"},
		{Name: "internet-loopback", Subnet: internetLoopback + "/32"},
	}
	for siteIndex, site := range request.Sites {
		for _, kind := range []string{"wan", "security", "core"} {
			name, subnet, _ := transit(site, kind, siteIndex)
			networks = append(networks, converter.Network{Name: name, Subnet: subnet})
		}
		for _, vlan := range vlanDefinitions() {
			networks = append(networks, converter.Network{
				Name:        siteNetworkName(site, vlan.slug),
				Subnet:      fmt.Sprintf("10.%d.%d.0/24", site.Octet, vlan.thirdOctet),
				VirtualVLAN: vlan.id,
			})
		}
	}
	return networks
}

func buildManifest(authored *converter.Config) Manifest {
	names := make([]string, 0, len(authored.Devices))
	for _, device := range authored.Devices {
		names = append(names, device.Name)
	}
	sort.Strings(names)

	networks := make([]string, 0, len(authored.Networks))
	for _, network := range authored.Networks {
		networks = append(networks, fmt.Sprintf("%s|%s|%d", network.Name, network.Subnet, network.VirtualVLAN))
	}
	sort.Strings(networks)

	edgeSet := make(map[string]bool)
	for _, device := range authored.Devices {
		for _, port := range device.TrunkPorts {
			local := device.Name + "|" + port.Interface
			remote := port.RemoteDevice + "|" + port.RemoteInterface
			vlans := intList(port.VLANs)
			if port.FDBOnly {
				edgeSet["FDB|"+local+"|"+remote+"|"+vlans] = true
				continue
			}
			if remote < local {
				local, remote = remote, local
			}
			edgeSet["LINK|"+local+"|"+remote+"|"+vlans] = true
		}
	}
	edges := make([]string, 0, len(edgeSet))
	for edge := range edgeSet {
		edges = append(edges, edge)
	}
	sort.Strings(edges)

	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		DeviceCount:   len(authored.Devices), NetworkCount: len(authored.Networks), LinkCount: len(edges),
		DeviceNamesSHA256: hashLines(names), NetworksSHA256: hashLines(networks), LinksSHA256: hashLines(edges),
		Interfaces:   buildInterfaceTruth(authored),
		Observations: buildObservations(authored),
		Timing:       buildTiming(authored),
	}
}

// buildIdentity digests the request that produced the scenario. Generation is
// deterministic, so this digest is the seed a consumer replays from.
func buildIdentity(request Request) (Identity, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return Identity{}, fmt.Errorf("digest scenario request: %w", err)
	}
	sum := sha256.Sum256(encoded)

	return Identity{
		RequestSHA256:   hex.EncodeToString(sum[:]),
		Domain:          request.Domain,
		AccessLayer:     string(request.AccessLayer),
		EndpointProfile: request.EndpointProfile,
	}, nil
}

// buildInterfaceTruth digests the operational facts an ifTable collector reads,
// and lifts out the authored congestion because that is the whole of the
// behaviour a pack currently authors.
func buildInterfaceTruth(authored *converter.Config) InterfaceTruth {
	lines := make([]string, 0)
	congested := make([]CongestedLink, 0)
	for _, device := range authored.Devices {
		for _, iface := range device.Interfaces {
			lines = append(lines, fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s|%d",
				device.Name, iface.Name, iface.Type, iface.Speed, iface.Duplex,
				iface.AdminStatus, iface.OperStatus, iface.MTU))
			if iface.InUtilization >= congestionWarningPercent ||
				iface.OutUtilization >= congestionWarningPercent {
				congested = append(congested, CongestedLink{
					Device: device.Name, Interface: iface.Name,
					InUtilization: iface.InUtilization, OutUtilization: iface.OutUtilization,
				})
			}
		}
	}
	sort.Strings(lines)
	sort.Slice(congested, func(i, j int) bool {
		if congested[i].Device != congested[j].Device {
			return congested[i].Device < congested[j].Device
		}

		return congested[i].Interface < congested[j].Interface
	})

	return InterfaceTruth{Count: len(lines), SHA256: hashLines(lines), Congested: congested}
}

// buildObservations records what each SEED collector should find. A collector
// the scenario authors nothing for is omitted rather than recorded as zero —
// see the Observation doc comment for why those are different claims.
func buildObservations(authored *converter.Config) map[string]Observation {
	observations := make(map[string]Observation)

	// Collectors whose answer is one row per device report only a device count;
	// table collectors also report how many rows those devices contribute.
	perDevice := func(collector string, rows deviceCounter) {
		if devices, _ := countDevices(authored, rows); devices > 0 {
			observations[collector] = Observation{Devices: devices}
		}
	}
	tabular := func(collector string, rows deviceCounter) {
		if devices, total := countDevices(authored, rows); devices > 0 {
			observations[collector] = Observation{Devices: devices, Rows: total}
		}
	}

	perDevice(CollectorSysInfo, func(d converter.Device) int { return present(d.SnmpAgent != nil) })
	perDevice(CollectorLLDP, func(d converter.Device) int {
		return present(d.Lldp != nil && d.Lldp.Enabled)
	})
	perDevice(CollectorCDP, func(d converter.Device) int {
		return present(d.Cdp != nil && d.Cdp.Enabled)
	})
	perDevice(CollectorFDP, func(d converter.Device) int {
		return present(d.Fdp != nil && d.Fdp.Enabled)
	})
	tabular(CollectorIfTable, func(d converter.Device) int { return len(d.Interfaces) })
	tabular(CollectorRouting, func(d converter.Device) int { return len(d.Routes) })
	tabular(CollectorFDB, countFDBPorts)

	return observations
}

// deviceCounter reports how many rows one device contributes to a collector.
type deviceCounter func(converter.Device) int

// countDevices returns how many devices contribute at least one row, and the
// total row count across them.
func countDevices(authored *converter.Config, rows deviceCounter) (int, int) {
	devices, total := 0, 0
	for _, device := range authored.Devices {
		count := rows(device)
		if count == 0 {
			continue
		}
		devices++
		total += count
	}

	return devices, total
}

func present(enabled bool) int {
	if enabled {
		return 1
	}

	return 0
}

func countFDBPorts(device converter.Device) int {
	ports := 0
	for _, port := range device.TrunkPorts {
		if port.FDBOnly {
			ports++
		}
	}

	return ports
}

// buildTiming derives the wait tolerance from the advertisement intervals of
// the protocols the scenario actually authors. A pack that advertises only LLDP
// and CDP settles in two 15s cycles; one that authors FDP needs two 60s cycles.
func buildTiming(authored *converter.Config) Timing {
	var timing Timing
	slowest := time.Duration(0)
	for _, device := range authored.Devices {
		if device.Lldp != nil && device.Lldp.Enabled {
			timing.LLDPIntervalSeconds = int(protocols.LLDPAdvertiseInterval.Seconds())
			slowest = max(slowest, protocols.LLDPAdvertiseInterval)
		}
		if device.Cdp != nil && device.Cdp.Enabled {
			timing.CDPIntervalSeconds = int(protocols.CDPAdvertiseInterval.Seconds())
			slowest = max(slowest, protocols.CDPAdvertiseInterval)
		}
		if device.Fdp != nil && device.Fdp.Enabled {
			timing.FDPIntervalSeconds = int(protocols.FDPAdvertiseInterval.Seconds())
			slowest = max(slowest, protocols.FDPAdvertiseInterval)
		}
	}
	timing.NeighborsStableAfterSeconds = int(slowest.Seconds()) * neighborSettleCycles

	return timing
}

func hashLines(lines []string) string {
	hash := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(hash[:])
}

func intList(values []int) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func siteIP(site Site, thirdOctet, host int) string {
	return fmt.Sprintf("10.%d.%d.%d", site.Octet, thirdOctet, host)
}

func siteNetworkName(site Site, slug string) string {
	return strings.ToLower(site.Code) + "-" + slug
}

func transit(site Site, kind string, siteIndex int) (string, string, int) {
	var base int
	switch kind {
	case "wan":
		base = 8
	case "security":
		base = 40
	case "core":
		base = 72
	default:
		panic("invalid transit kind: " + kind)
	}
	base += transitBlockSize * siteIndex
	return strings.ToLower(site.Code) + "-" + kind + "-transit",
		fmt.Sprintf("203.0.113.%d/29", base), base
}

// applyCongestion replaces the generated band on the interfaces a pack calls
// out as its trouble spots. An interface that does not exist is a typo, and a
// typo that quietly leaves the map healthy is the one outcome worth failing on:
// the whole point of the story is that an engineer finds it.
func applyCongestion(authored *converter.Config, links []CongestedLink) error {
	for _, link := range links {
		iface := findAuthoredInterface(authored, link.Device, link.Interface)
		if iface == nil {
			return fmt.Errorf("congested link %s %s does not exist", link.Device, link.Interface)
		}
		iface.InUtilization = link.InUtilization
		iface.OutUtilization = link.OutUtilization
	}

	return nil
}

func findAuthoredInterface(authored *converter.Config, device, name string) *converter.Interface {
	for deviceIndex := range authored.Devices {
		if authored.Devices[deviceIndex].Name != device {
			continue
		}
		for index := range authored.Devices[deviceIndex].Interfaces {
			if authored.Devices[deviceIndex].Interfaces[index].Name == name {
				return &authored.Devices[deviceIndex].Interfaces[index]
			}
		}
	}

	return nil
}
