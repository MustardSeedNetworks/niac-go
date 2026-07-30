// Package drafttopology applies typed visual topology edits to an isolated configuration draft.
package drafttopology

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"net"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const (
	maxCoordinate = 1_000_000
	maxMACSuffix  = 0xFFFFFF
)

var (
	errInvalid  = errors.New("invalid topology mutation")
	errNotFound = errors.New("topology resource not found")
	errConflict = errors.New("topology mutation conflict")
)

type Operation string

const (
	AddDevice  Operation = "add_device"
	Connect    Operation = "connect"
	Disconnect Operation = "disconnect"
	MoveDevice Operation = "move_device"
	UpdateLink Operation = "update_link"
)

type Mutation struct {
	Operation Operation         `json:"operation"`
	Device    *DeviceMutation   `json:"device,omitempty"`
	Link      *LinkMutation     `json:"link,omitempty"`
	Position  *PositionMutation `json:"position,omitempty"`
}

type DeviceMutation struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Vendor      string            `json:"vendor,omitempty"`
	MAC         string            `json:"mac,omitempty"`
	MACSuffix   uint32            `json:"mac_suffix,omitempty"`
	SysObjectID string            `json:"sys_object_id,omitempty"`
	IPs         []string          `json:"ips,omitempty"`
	VLAN        int               `json:"vlan,omitempty"`
	Interfaces  []Interface       `json:"interfaces,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
	ProfileRole string            `json:"profile_role,omitempty"`
	WalkFile    string            `json:"-"`
}

type Interface struct {
	Name           string  `json:"name"`
	Type           string  `json:"type,omitempty"`
	Network        string  `json:"network,omitempty"`
	Address        string  `json:"address,omitempty"`
	MTU            int     `json:"mtu,omitempty"`
	Speed          int     `json:"speed,omitempty"`
	Duplex         string  `json:"duplex,omitempty"`
	AdminStatus    string  `json:"admin_status,omitempty"`
	OperStatus     string  `json:"oper_status,omitempty"`
	Description    string  `json:"description,omitempty"`
	InUtilization  float64 `json:"in_utilization,omitempty"`
	OutUtilization float64 `json:"out_utilization,omitempty"`
	VLANs          []int   `json:"vlans,omitempty"`
}

type Endpoint struct {
	Device    string `json:"device"`
	Interface string `json:"interface"`
}

type LinkEndpoints struct {
	Source Endpoint `json:"source"`
	Target Endpoint `json:"target"`
}

type LinkMutation struct {
	LinkEndpoints

	Properties LinkProperties `json:"properties"`
}

type LinkProperties struct {
	VLANs      []int `json:"vlans,omitempty"`
	NativeVLAN int   `json:"native_vlan,omitempty"`
	FDBOnly    bool  `json:"fdb_only,omitempty"`
}

type PositionMutation struct {
	Device string  `json:"device"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

func IsInvalid(err error) bool  { return errors.Is(err, errInvalid) }
func IsNotFound(err error) bool { return errors.Is(err, errNotFound) }
func IsConflict(err error) bool { return errors.Is(err, errConflict) }

// ValidateSource rejects source forms that canonical runtime serialization cannot preserve.
func ValidateSource(content string) error {
	var source struct {
		Segments []struct {
			Config string `yaml:"config"`
		} `yaml:"segments"`
	}
	if err := yaml.Unmarshal([]byte(content), &source); err != nil {
		return invalid("draft YAML is invalid")
	}
	for _, segment := range source.Segments {
		if strings.TrimSpace(segment.Config) != "" {
			return invalid("visual mutations cannot preserve config-backed segments")
		}
	}
	return nil
}

func Apply(cfg *config.Config, mutation Mutation) error {
	if cfg == nil {
		return invalid("configuration is required")
	}
	if err := validatePayload(mutation); err != nil {
		return err
	}

	switch mutation.Operation {
	case AddDevice:
		return addDevice(cfg, *mutation.Device)
	case Connect:
		return connect(cfg, *mutation.Link)
	case Disconnect:
		return disconnect(cfg, mutation.Link.LinkEndpoints)
	case MoveDevice:
		return moveDevice(cfg, *mutation.Position)
	case UpdateLink:
		return updateLink(cfg, *mutation.Link)
	default:
		return invalid("unsupported operation %q", mutation.Operation)
	}
}

func validatePayload(mutation Mutation) error {
	device, link, position := mutation.Device != nil, mutation.Link != nil, mutation.Position != nil
	switch mutation.Operation {
	case AddDevice:
		if !device || link || position {
			return invalid("add_device requires only device")
		}
	case Connect, Disconnect, UpdateLink:
		if device || !link || position {
			return invalid("%s requires only link", mutation.Operation)
		}
	case MoveDevice:
		if device || link || !position {
			return invalid("move_device requires only position")
		}
	default:
		return invalid("unsupported operation %q", mutation.Operation)
	}
	return nil
}

func addDevice(cfg *config.Config, input DeviceMutation) error {
	if len(cfg.Segments) != 0 {
		return invalid("add_device is not supported for segmented drafts")
	}
	if strings.TrimSpace(input.Name) == "" || input.Name != strings.TrimSpace(input.Name) {
		return invalid("device name is required without surrounding whitespace")
	}
	if strings.TrimSpace(input.Type) == "" {
		return invalid("device type is required")
	}
	if input.MAC != "" && input.Vendor != "" {
		return invalid("device identity must use either mac or vendor")
	}
	if input.MAC == "" && input.Vendor == "" {
		return invalid("device identity requires mac or vendor")
	}
	if input.MACSuffix > maxMACSuffix {
		return invalid("MAC suffix must fit within 24 bits")
	}
	if findDevice(cfg, input.Name) != nil {
		return conflict("device %q already exists", input.Name)
	}

	var mac net.HardwareAddr
	var err error
	if input.MAC != "" {
		mac, err = net.ParseMAC(input.MAC)
		if err != nil {
			return invalid("invalid MAC address")
		}
	}
	ips := make([]net.IP, len(input.IPs))
	for index, raw := range input.IPs {
		ips[index] = net.ParseIP(raw)
		if ips[index] == nil {
			return invalid("invalid IP address at index %d", index)
		}
	}
	interfaces := make([]config.Interface, len(input.Interfaces))
	for index, iface := range input.Interfaces {
		interfaces[index] = config.Interface{
			Name: iface.Name, Type: iface.Type, Network: iface.Network, Address: iface.Address,
			MTU: iface.MTU, Speed: iface.Speed, Duplex: iface.Duplex,
			AdminStatus: iface.AdminStatus, OperStatus: iface.OperStatus,
			Description: iface.Description, InUtilization: iface.InUtilization,
			OutUtilization: iface.OutUtilization, VLANs: slices.Clone(iface.VLANs),
		}
	}
	properties := cloneMap(input.Properties)
	if input.SysObjectID != "" {
		if properties == nil {
			properties = make(map[string]string)
		}
		properties["sysObjectID"] = input.SysObjectID
	}
	device := config.Device{
		Name: input.Name, Type: input.Type, MACAddress: mac,
		MACVendor: input.Vendor, MACSuffix: input.MACSuffix,
		IPAddresses: ips, VLAN: input.VLAN, Interfaces: interfaces,
		Properties: properties, SNMPConfig: capturedSNMPConfig(input),
	}
	cfg.Devices = append(cfg.Devices, device)
	return nil
}

func capturedSNMPConfig(input DeviceMutation) config.SNMPConfig {
	if input.WalkFile == "" {
		return config.SNMPConfig{}
	}
	enabled := true
	return config.SNMPConfig{
		Enabled: &enabled, Community: "public", SysName: input.Name, WalkFile: input.WalkFile,
	}
}

func connect(cfg *config.Config, link LinkMutation) error {
	left, right, err := linkDevicesAndInterfaces(cfg, link.LinkEndpoints)
	if err != nil {
		return err
	}
	if deviceSegment(cfg, left.endpoint.Device) != deviceSegment(cfg, right.endpoint.Device) {
		return invalid("a link cannot cross isolated segments")
	}
	if endpointOccupied(cfg, left.endpoint) || endpointOccupied(cfg, right.endpoint) {
		return conflict("one or both interfaces are already connected")
	}
	if validationErr := validateLinkProperties(link.Properties); validationErr != nil {
		return validationErr
	}
	appendTrunk(left, right, link.Properties)
	appendTrunk(right, left, link.Properties)
	return nil
}

func disconnect(cfg *config.Config, endpoints LinkEndpoints) error {
	left, right, err := linkDevicesAndInterfaces(cfg, endpoints)
	if err != nil {
		return err
	}
	leftIndex, rightIndex := reciprocalIndices(left, right)
	if leftIndex < 0 || rightIndex < 0 {
		return conflict("reciprocal link does not exist")
	}
	left.device.TrunkPorts = slices.Delete(left.device.TrunkPorts, leftIndex, leftIndex+1)
	right.device.TrunkPorts = slices.Delete(right.device.TrunkPorts, rightIndex, rightIndex+1)
	clearInterfaceLink(left.iface, right.endpoint)
	clearInterfaceLink(right.iface, left.endpoint)
	return nil
}

func updateLink(cfg *config.Config, link LinkMutation) error {
	left, right, err := linkDevicesAndInterfaces(cfg, link.LinkEndpoints)
	if err != nil {
		return err
	}
	leftIndex, rightIndex := reciprocalIndices(left, right)
	if leftIndex < 0 || rightIndex < 0 {
		return conflict("reciprocal link does not exist")
	}
	if !sameTrunkProperties(
		&left.device.TrunkPorts[leftIndex],
		&right.device.TrunkPorts[rightIndex],
	) {
		return conflict("asymmetric link properties require YAML editing")
	}
	if validationErr := validateLinkProperties(link.Properties); validationErr != nil {
		return validationErr
	}
	setTrunkProperties(&left.device.TrunkPorts[leftIndex], link.Properties)
	setTrunkProperties(&right.device.TrunkPorts[rightIndex], link.Properties)
	left.iface.VLANs = slices.Clone(link.Properties.VLANs)
	right.iface.VLANs = slices.Clone(link.Properties.VLANs)
	return nil
}

func moveDevice(cfg *config.Config, position PositionMutation) error {
	if math.IsNaN(position.X) || math.IsNaN(position.Y) || math.IsInf(position.X, 0) ||
		math.IsInf(position.Y, 0) ||
		math.Abs(position.X) > maxCoordinate ||
		math.Abs(position.Y) > maxCoordinate {
		return invalid("position coordinates must be finite and within bounds")
	}
	device := findDevice(cfg, position.Device)
	if device == nil {
		return notFound("device %q", position.Device)
	}
	if device.Properties == nil {
		device.Properties = make(map[string]string)
	}
	device.Properties["topology_x"] = strconv.FormatFloat(position.X, 'f', -1, 64)
	device.Properties["topology_y"] = strconv.FormatFloat(position.Y, 'f', -1, 64)
	return nil
}

type resolvedEndpoint struct {
	endpoint Endpoint
	device   *config.Device
	iface    *config.Interface
}

func linkDevicesAndInterfaces(
	cfg *config.Config,
	endpoints LinkEndpoints,
) (resolvedEndpoint, resolvedEndpoint, error) {
	if endpoints.Source.Device == endpoints.Target.Device {
		return resolvedEndpoint{}, resolvedEndpoint{}, invalid(
			"a link requires two different devices",
		)
	}
	left, err := resolveEndpoint(cfg, endpoints.Source)
	if err != nil {
		return resolvedEndpoint{}, resolvedEndpoint{}, err
	}
	right, err := resolveEndpoint(cfg, endpoints.Target)
	if err != nil {
		return resolvedEndpoint{}, resolvedEndpoint{}, err
	}
	return left, right, nil
}

func resolveEndpoint(cfg *config.Config, endpoint Endpoint) (resolvedEndpoint, error) {
	device := findDevice(cfg, endpoint.Device)
	if device == nil {
		return resolvedEndpoint{}, notFound("device %q", endpoint.Device)
	}
	for index := range device.Interfaces {
		iface := &device.Interfaces[index]
		if iface.Name != endpoint.Interface {
			continue
		}
		if iface.Type != "" && iface.Type != "ethernet" && iface.Type != "other" {
			return resolvedEndpoint{}, invalid(
				"interface %q on %q cannot carry a physical link",
				endpoint.Interface,
				endpoint.Device,
			)
		}
		return resolvedEndpoint{endpoint: endpoint, device: device, iface: iface}, nil
	}
	return resolvedEndpoint{}, notFound(
		"interface %q on device %q",
		endpoint.Interface,
		endpoint.Device,
	)
}

func validateLinkProperties(properties LinkProperties) error {
	seen := make(map[int]struct{}, len(properties.VLANs))
	for _, vlan := range properties.VLANs {
		if vlan < 1 || vlan > 4094 {
			return invalid("VLAN %d is outside 1-4094", vlan)
		}
		if _, found := seen[vlan]; found {
			return invalid("VLAN %d is duplicated", vlan)
		}
		seen[vlan] = struct{}{}
	}
	if properties.NativeVLAN != 0 {
		if _, found := seen[properties.NativeVLAN]; !found {
			return invalid("native VLAN must be in the allowed VLAN list")
		}
	}
	return nil
}

func appendTrunk(local, remote resolvedEndpoint, properties LinkProperties) {
	local.device.TrunkPorts = append(local.device.TrunkPorts, config.TrunkPort{
		Interface: local.endpoint.Interface, RemoteDevice: remote.endpoint.Device,
		RemoteInterface: remote.endpoint.Interface,
	})
	setTrunkProperties(&local.device.TrunkPorts[len(local.device.TrunkPorts)-1], properties)
	local.iface.Description = "to " + remote.endpoint.Device + " " + remote.endpoint.Interface
	local.iface.VLANs = slices.Clone(properties.VLANs)
}

func setTrunkProperties(trunk *config.TrunkPort, properties LinkProperties) {
	trunk.VLANs = slices.Clone(properties.VLANs)
	trunk.NativeVLAN = properties.NativeVLAN
	trunk.FDBOnly = properties.FDBOnly
}

func sameTrunkProperties(left, right *config.TrunkPort) bool {
	leftVLANs := slices.Clone(left.VLANs)
	rightVLANs := slices.Clone(right.VLANs)
	slices.Sort(leftVLANs)
	slices.Sort(rightVLANs)
	return slices.Equal(leftVLANs, rightVLANs) && left.NativeVLAN == right.NativeVLAN &&
		left.FDBOnly == right.FDBOnly
}

func reciprocalIndices(left, right resolvedEndpoint) (int, int) {
	return exactTrunkIndex(left.device, left.endpoint.Interface, right.endpoint),
		exactTrunkIndex(right.device, right.endpoint.Interface, left.endpoint)
}

func endpointOccupied(cfg *config.Config, endpoint Endpoint) bool {
	for _, segment := range cfg.NormalizedSegments() {
		for deviceIndex := range segment.Devices {
			device := &segment.Devices[deviceIndex]
			for _, trunk := range device.TrunkPorts {
				if (device.Name == endpoint.Device && trunk.Interface == endpoint.Interface) ||
					(trunk.RemoteDevice == endpoint.Device && trunk.RemoteInterface == endpoint.Interface) {
					return true
				}
			}
		}
	}
	return false
}

func exactTrunkIndex(device *config.Device, iface string, remote Endpoint) int {
	for index := range device.TrunkPorts {
		trunk := &device.TrunkPorts[index]
		if trunk.Interface == iface && trunk.RemoteDevice == remote.Device &&
			trunk.RemoteInterface == remote.Interface {
			return index
		}
	}
	return -1
}

func clearInterfaceLink(iface *config.Interface, remote Endpoint) {
	if iface.Description == "to "+remote.Device+" "+remote.Interface {
		iface.Description = ""
	}
	iface.VLANs = nil
}

func findDevice(cfg *config.Config, name string) *config.Device {
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == name {
			return &cfg.Devices[index]
		}
	}
	for segmentIndex := range cfg.Segments {
		for deviceIndex := range cfg.Segments[segmentIndex].Devices {
			device := &cfg.Segments[segmentIndex].Devices[deviceIndex]
			if device.Name == name {
				return device
			}
		}
	}
	return nil
}

func deviceSegment(cfg *config.Config, name string) int {
	if len(cfg.Segments) == 0 {
		return 0
	}
	for segmentIndex := range cfg.Segments {
		for deviceIndex := range cfg.Segments[segmentIndex].Devices {
			if cfg.Segments[segmentIndex].Devices[deviceIndex].Name == name {
				return segmentIndex
			}
		}
	}
	return -1
}

func cloneMap(source map[string]string) map[string]string {
	return maps.Clone(source)
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalid, fmt.Sprintf(format, args...))
}

func notFound(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errNotFound, fmt.Sprintf(format, args...))
}

func conflict(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errConflict, fmt.Sprintf(format, args...))
}
