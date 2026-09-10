package converter

import (
	"bytes"
	"encoding/binary"
	"net"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// MaxDHCPv4PoolSize bounds allocation for an authored DHCPv4 pool.
const MaxDHCPv4PoolSize = 65536

func validateDHCPv4Options(validation validator.StructLevel) {
	server, ok := reflect.TypeAssert[DhcpServer](validation.Current())
	if !ok {
		return
	}
	mask := net.ParseIP(server.SubnetMask).To4()
	if mask != nil && !ValidDHCPv4Mask(net.IPMask(mask)) {
		validation.ReportError(server.SubnetMask, "subnet_mask", "SubnetMask", "ipv4_mask", "")
	}
	start, end := net.ParseIP(server.PoolStart), net.ParseIP(server.PoolEnd)
	if start.To4() != nil && end.To4() != nil && !ValidDHCPv4Pool(start, end) {
		validation.ReportError(server.PoolEnd, "pool_end", "PoolEnd", "dhcp_pool", "")
	}
}

// ValidDHCPv4Mask accepts contiguous IPv4 masks, including an omitted mask.
func ValidDHCPv4Mask(mask net.IPMask) bool {
	if len(mask) == 0 {
		return true
	}
	_, bits := mask.Size()
	return bits == net.IPv4len*8
}

// ValidDHCPv4Pool requires paired, ordered, bounded IPv4 endpoints or an omitted pool.
func ValidDHCPv4Pool(start, end net.IP) bool {
	if start == nil && end == nil {
		return true
	}
	start4, end4 := start.To4(), end.To4()
	if start4 == nil || end4 == nil || bytes.Compare(start4, end4) > 0 {
		return false
	}
	size := uint64(binary.BigEndian.Uint32(end4)) - uint64(binary.BigEndian.Uint32(start4)) + 1
	return size <= MaxDHCPv4PoolSize
}
