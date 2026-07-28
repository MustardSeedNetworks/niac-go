package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

const (
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

	return Result{YAML: data, Manifest: buildManifest(&authored)}, nil
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
		DeviceCount: len(authored.Devices), NetworkCount: len(authored.Networks), LinkCount: len(edges),
		DeviceNamesSHA256: hashLines(names), NetworksSHA256: hashLines(networks), LinksSHA256: hashLines(edges),
	}
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
