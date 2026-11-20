package scenario

import (
	"fmt"
	"strings"
)

func numberedNames(prefix string, count int) []string {
	if count < 1 || count > maxRedundantPeers {
		return nil
	}

	names := make([]string, 0, maxRedundantPeers)
	for index := 1; index <= count; index++ {
		names = append(names, numberedName(prefix, index))
	}
	return names
}

func numberedName(prefix string, index int) string {
	return fmt.Sprintf("%s%02d", prefix, index)
}

func wirelessControllerName(site Site, index int) string {
	return numberedName(site.Code+"-WLC", index)
}

func accessName(site Site, index int) string {
	return numberedName(site.Code+"-ACC-SW", index)
}

func location(index int) (int, int) {
	return (index-1)/4 + 1, (index-1)%4 + 1
}

func accessPointName(site Site, accessIndex, slot int) string {
	building, floor := location(accessIndex)
	return fmt.Sprintf("%s-WAP-B%02d-F%02d-%02d", site.Code, building, floor, slot)
}

// roleOfDevice names the role of a switching, routing or access-point device
// from the name this file gave it, so a link's far end can be classified
// without a second registry of devices. The infixes are the ones numberedName
// and accessPointName produce, plus the two fixed lab-edge and provider names.
// A name it cannot place is a generator bug, so it panics like roleCode does.
func roleOfDevice(name string) string {
	for _, known := range []struct{ infix, role string }{
		{"-CORE-SW", roleCoreSwitch},
		{"-DIST-SW", roleDistribution},
		{"-ACC-SW", roleAccessSwitch},
		{"-SRV-SW", roleServerSwitch},
		{"-WAP-", roleAccessPoint},
		{"-WAN-R", roleWAN},
		{"-FW", roleFirewall},
	} {
		if strings.Contains(name, known.infix) {
			return known.role
		}
	}
	switch {
	case strings.HasPrefix(name, "WAN-R"):
		return roleWAN
	case strings.HasPrefix(name, "LAB-EDGE"):
		return roleLabEdge
	}
	panic("unplaceable scenario device: " + name)
}
