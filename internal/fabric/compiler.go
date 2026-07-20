package fabric

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// Compile validates and canonicalizes a routed scenario without side effects.
func Compile(cfg *config.Config, binding Binding) Report {
	compiler := scenarioCompiler{
		cfg:       cfg,
		binding:   binding,
		networks:  make(map[string]Network),
		devices:   make(map[string]struct{}),
		addresses: make(map[netip.Addr]string),
	}
	compiler.compileBinding()
	compiler.compileNetworks()
	compiler.compileDevices()
	compiler.report.Safe = len(compiler.report.Diagnostics) == 0
	return compiler.report
}

type scenarioCompiler struct {
	cfg       *config.Config
	binding   Binding
	networks  map[string]Network
	devices   map[string]struct{}
	addresses map[netip.Addr]string
	report    Report
}

func (c *scenarioCompiler) compileBinding() {
	c.validateBindingMode()
	if c.cfg == nil {
		c.add(CodeUnknownAttachment, "attachment", "scenario is required")
		return
	}
	for _, attachment := range c.cfg.Attachments {
		if attachment.Name != c.binding.Attachment {
			continue
		}
		c.report.Topology.Binding = CompiledBinding{
			Binding: c.binding, Network: attachment.Network, WireTagged: false,
		}
		return
	}
	c.add(CodeUnknownAttachment, "attachment", "logical attachment does not exist")
}

func (c *scenarioCompiler) validateBindingMode() {
	switch c.binding.Mode {
	case ModeDirect:
		if !c.binding.Dedicated {
			c.add(CodeDedicatedRequired, "dedicated", "direct mode requires a dedicated interface")
		}
		if c.binding.AccessVLAN != 0 {
			c.add(CodeInvalidAccessVLAN, "accessVlan", "direct mode does not use an access VLAN")
		}
	case ModeAccess:
		if c.binding.AccessVLAN < 1 || c.binding.AccessVLAN > 4094 {
			c.add(CodeInvalidAccessVLAN, "accessVlan", "access VLAN must be between 1 and 4094")
		}
	default:
		c.add(CodeInvalidAttachmentMode, "mode", "mode must be direct or access")
	}
}

func (c *scenarioCompiler) compileNetworks() {
	if c.cfg == nil {
		return
	}
	for i, source := range c.cfg.Networks {
		field := fmt.Sprintf("networks[%d]", i)
		prefix, err := parseCanonicalPrefix(source.Subnet)
		if err != nil {
			c.add(CodeInvalidNetwork, field+".subnet", err.Error())
			continue
		}
		if _, exists := c.networks[source.Name]; exists {
			c.add(CodeDuplicateNetwork, field+".name", "network name must be unique")
			continue
		}
		if source.VirtualVLAN < 0 || source.VirtualVLAN > 4094 {
			c.add(
				CodeInvalidVirtualVLAN,
				field+".virtual_vlan",
				"virtual VLAN must be between 1 and 4094",
			)
			continue
		}
		network := Network{
			Name: source.Name, Prefix: prefix, VirtualVLAN: safeconv.Uint16(source.VirtualVLAN),
		}
		c.checkOverlap(field, network)
		c.networks[source.Name] = network
		c.report.Topology.Networks = append(c.report.Topology.Networks, network)
	}
	c.validateAttachmentNetwork()
}

func (c *scenarioCompiler) checkOverlap(field string, candidate Network) {
	for _, network := range c.report.Topology.Networks {
		if network.Prefix.Overlaps(candidate.Prefix) {
			c.add(CodeOverlappingNetworks, field+".subnet", "network prefixes must not overlap")
			return
		}
	}
}

func (c *scenarioCompiler) validateAttachmentNetwork() {
	name := c.report.Topology.Binding.Network
	if name == "" {
		return
	}
	if _, exists := c.networks[name]; !exists {
		c.add(CodeUnknownNetwork, "attachment", "attachment references an unknown network")
	}
}

func (c *scenarioCompiler) compileDevices() {
	if c.cfg == nil {
		return
	}
	interfacesByDevice := make([]map[string]Interface, len(c.cfg.Devices))
	for i := range c.cfg.Devices {
		device := &c.cfg.Devices[i]
		if _, exists := c.devices[device.Name]; exists {
			c.add(CodeDuplicateDevice, fmt.Sprintf("devices[%d].name", i), "device name must be unique")
			continue
		}
		c.devices[device.Name] = struct{}{}
		interfacesByDevice[i] = c.compileInterfaces(device)
	}
	for i := range c.cfg.Devices {
		if interfacesByDevice[i] == nil {
			continue
		}
		device := &c.cfg.Devices[i]
		c.compileRoutes(device, interfacesByDevice[i])
		c.compileDHCP(device, interfacesByDevice[i])
	}
}

func (c *scenarioCompiler) add(code DiagnosticCode, field, message string) {
	c.report.Diagnostics = append(c.report.Diagnostics, Diagnostic{
		Code: code, Field: field, Message: message,
	})
}

func parseCanonicalPrefix(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil || !prefix.Addr().Is4() {
		return netip.Prefix{}, errors.New("must be an IPv4 prefix")
	}
	if prefix != prefix.Masked() {
		return netip.Prefix{}, errors.New("prefix must use its network address")
	}
	return prefix, nil
}
