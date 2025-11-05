package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

func TestNewPacket(t *testing.T) {
	pkt := NewPacket(1500)
	if pkt == nil {
		t.Fatal("NewPacket returned nil")
	}
	if len(pkt.Buffer) != 1500 {
		t.Errorf("Expected buffer size 1500, got %d", len(pkt.Buffer))
	}
	if pkt.VLAN != -1 {
		t.Errorf("Expected VLAN -1, got %d", pkt.VLAN)
	}
}

func TestPacketClone(t *testing.T) {
	pkt := NewPacket(100)
	pkt.Buffer[0] = 0xAA
	pkt.Length = 50
	pkt.SerialNumber = 123

	clone := pkt.Clone()
	if clone == nil {
		t.Fatal("Clone returned nil")
	}

	// Verify values copied
	if clone.Buffer[0] != 0xAA {
		t.Error("Buffer not cloned correctly")
	}
	if clone.Length != 50 {
		t.Error("Length not cloned correctly")
	}
	if clone.SerialNumber != 123 {
		t.Error("SerialNumber not cloned correctly")
	}

	// Verify deep copy (modifying clone doesn't affect original)
	clone.Buffer[0] = 0xBB
	if pkt.Buffer[0] != 0xAA {
		t.Error("Clone is not a deep copy")
	}
}

func TestPacket16BitOperations(t *testing.T) {
	pkt := NewPacket(100)

	// Test Put16 and Get16
	pkt.Put16(0x1234, 0)
	val := pkt.Get16(0)
	if val != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%x", val)
	}

	pkt.Put16(0xABCD, 10)
	val = pkt.Get16(10)
	if val != 0xABCD {
		t.Errorf("Expected 0xABCD, got 0x%x", val)
	}
}

func TestPacket32BitOperations(t *testing.T) {
	pkt := NewPacket(100)

	// Test Put32 and Get32
	pkt.Put32(0x12345678, 0)
	val := pkt.Get32(0)
	if val != 0x12345678 {
		t.Errorf("Expected 0x12345678, got 0x%x", val)
	}
}

func TestPacketMACOperations(t *testing.T) {
	pkt := NewPacket(100)

	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// Test dest MAC (offset 0)
	pkt.PutDestMAC(dstMAC)
	gotDst := pkt.GetDestMAC()
	if !macEqual(gotDst, dstMAC) {
		t.Errorf("Dest MAC mismatch: got %s, want %s", gotDst, dstMAC)
	}

	// Test source MAC (offset 6)
	pkt.PutSourceMAC(srcMAC)
	gotSrc := pkt.GetSourceMAC()
	if !macEqual(gotSrc, srcMAC) {
		t.Errorf("Source MAC mismatch: got %s, want %s", gotSrc, srcMAC)
	}

	// Test CopySourceMACToDest
	pkt.CopySourceMACToDest()
	gotDst = pkt.GetDestMAC()
	if !macEqual(gotDst, srcMAC) {
		t.Errorf("CopySourceMACToDest failed: got %s, want %s", gotDst, srcMAC)
	}
}

func TestPacketIPOperations(t *testing.T) {
	pkt := NewPacket(100)

	ip := net.ParseIP("192.168.1.1")

	pkt.PutIP(ip, 14)
	gotIP := pkt.GetIP(14)

	if !gotIP.Equal(ip.To4()) {
		t.Errorf("IP mismatch: got %s, want %s", gotIP, ip)
	}
}

func TestBuildEthernetHeader(t *testing.T) {
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	header := BuildEthernetHeader(dstMAC, srcMAC, EtherTypeIP)

	if len(header) != 14 {
		t.Errorf("Expected header length 14, got %d", len(header))
	}

	// Check dest MAC
	if !macEqual(header[0:6], dstMAC) {
		t.Error("Dest MAC mismatch in header")
	}

	// Check src MAC
	if !macEqual(header[6:12], srcMAC) {
		t.Error("Src MAC mismatch in header")
	}

	// Check EtherType
	etherType := uint16(header[12])<<8 | uint16(header[13])
	if etherType != EtherTypeIP {
		t.Errorf("EtherType mismatch: got 0x%x, want 0x%x", etherType, EtherTypeIP)
	}
}

func TestDeviceTable(t *testing.T) {
	dt := NewDeviceTable()

	// Create test device
	device := &config.Device{
		Name:        "TestDevice",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	// Add device
	dt.AddByMAC(device.MACAddress, device)
	dt.AddByIP(device.IPAddresses[0], device)

	// Test lookup by MAC
	found := dt.GetByMAC(device.MACAddress)
	if found == nil {
		t.Fatal("Device not found by MAC")
	}
	if found.Name != "TestDevice" {
		t.Errorf("Wrong device found: %s", found.Name)
	}

	// Test lookup by IP
	foundDevices := dt.GetByIP(device.IPAddresses[0])
	if len(foundDevices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(foundDevices))
	}
	if foundDevices[0].Name != "TestDevice" {
		t.Errorf("Wrong device found: %s", foundDevices[0].Name)
	}

	// Test count
	if dt.Count() != 1 {
		t.Errorf("Expected count 1, got %d", dt.Count())
	}

	// Test remove
	dt.Remove(device)
	if dt.Count() != 0 {
		t.Errorf("Expected count 0 after remove, got %d", dt.Count())
	}

	found = dt.GetByMAC(device.MACAddress)
	if found != nil {
		t.Error("Device should not be found after removal")
	}
}

func TestDeviceTableMultiple(t *testing.T) {
	dt := NewDeviceTable()

	// Add multiple devices
	for i := 0; i < 10; i++ {
		device := &config.Device{
			Name:        string(rune('A' + i)),
			MACAddress:  net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, byte(i)},
			IPAddresses: []net.IP{net.IPv4(192, 168, 1, byte(i+1))},
		}
		dt.AddByMAC(device.MACAddress, device)
		dt.AddByIP(device.IPAddresses[0], device)
	}

	if dt.Count() != 10 {
		t.Errorf("Expected count 10, got %d", dt.Count())
	}

	// Test AllDevices
	all := dt.AllDevices()
	if len(all) != 10 {
		t.Errorf("Expected 10 devices from AllDevices, got %d", len(all))
	}
}

func TestStatistics(t *testing.T) {
	stats := &Statistics{}

	stats.mu.Lock()
	stats.PacketsReceived = 100
	stats.PacketsSent = 50
	stats.ARPRequests = 10
	stats.mu.Unlock()

	stats.mu.RLock()
	if stats.PacketsReceived != 100 {
		t.Errorf("Expected 100 packets received, got %d", stats.PacketsReceived)
	}
	if stats.PacketsSent != 50 {
		t.Errorf("Expected 50 packets sent, got %d", stats.PacketsSent)
	}
	stats.mu.RUnlock()
}

// Helper function to compare MAC addresses
func macEqual(a, b net.HardwareAddr) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func BenchmarkPacketClone(b *testing.B) {
	pkt := NewPacket(1500)
	pkt.Length = 1500

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pkt.Clone()
	}
}

func BenchmarkPacket16BitOps(b *testing.B) {
	pkt := NewPacket(1500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkt.Put16(0x1234, 0)
		_ = pkt.Get16(0)
	}
}

func BenchmarkDeviceTableLookup(b *testing.B) {
	dt := NewDeviceTable()

	// Add 100 devices
	devices := make([]*config.Device, 100)
	for i := 0; i < 100; i++ {
		device := &config.Device{
			Name:        string(rune('A' + i)),
			MACAddress:  net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, byte(i)},
			IPAddresses: []net.IP{net.IPv4(192, 168, 1, byte(i+1))},
		}
		devices[i] = device
		dt.AddByMAC(device.MACAddress, device)
		dt.AddByIP(device.IPAddresses[0], device)
	}

	// Benchmark lookup
	testMAC := devices[50].MACAddress
	testIP := devices[50].IPAddresses[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dt.GetByMAC(testMAC)
		_ = dt.GetByIP(testIP)
	}
}

func TestParsePacketVLAN(t *testing.T) {
	// Create a packet with VLAN tag
	pkt := NewPacket(100)

	// Ethernet header with VLAN
	pkt.PutDestMAC(net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	pkt.PutSourceMAC(net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	pkt.Put16(EtherTypeVLAN, 12)  // VLAN tag
	pkt.Put16(0x0064, 14)          // VLAN ID 100
	pkt.Put16(EtherTypeIP, 16)     // Actual EtherType after VLAN

	parsed, err := ParsePacket(pkt.Buffer, 1)
	if err != nil {
		t.Fatalf("ParsePacket failed: %v", err)
	}

	if parsed.VLAN != 100 {
		t.Errorf("Expected VLAN 100, got %d", parsed.VLAN)
	}
}

func TestCalculateIPChecksum(t *testing.T) {
	// Simple IP header (version 4, IHL 5, no options)
	header := []byte{
		0x45, 0x00, // Version/IHL, TOS
		0x00, 0x14, // Total length (20 bytes)
		0x00, 0x00, // ID
		0x00, 0x00, // Flags/Fragment
		0x40, 0x01, // TTL, Protocol (ICMP)
		0x00, 0x00, // Checksum (will be calculated)
		0xC0, 0xA8, 0x01, 0x01, // Source IP 192.168.1.1
		0xC0, 0xA8, 0x01, 0x02, // Dest IP 192.168.1.2
	}

	checksum := CalculateIPChecksum(header)

	// Checksum should not be zero (this is a basic sanity check)
	if checksum == 0 {
		t.Error("Checksum should not be zero")
	}

	// Put checksum in header and recalculate - should be 0xFFFF
	header[10] = byte(checksum >> 8)
	header[11] = byte(checksum)

	verify := CalculateIPChecksum(header)
	if verify != 0xFFFF && verify != 0x0000 {
		t.Errorf("Checksum verification failed: got 0x%04x", verify)
	}
}

func init() {
	// Set shorter timeout for tests
	_ = time.Second
}
