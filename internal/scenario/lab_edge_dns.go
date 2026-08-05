package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// setLabEdgeDNS publishes forward and reverse records for every generated
// device on the lab-edge resolver, so a tester can resolve any simulated host
// without per-device DNS configuration.
func setLabEdgeDNS(devices []converter.Device, domain string) {
	records := internetDNSRecords()
	for index := range devices {
		records = append(records, deviceDNSRecords(&devices[index], domain)...)
	}
	devices[0].DNS.ForwardRecords = records
	devices[0].DNS.ReverseRecords = records
}

func deviceDNSRecords(device *converter.Device, domain string) []converter.DNSRecord {
	records := make([]converter.DNSRecord, len(device.IPs))
	for index, address := range device.IPs {
		records[index] = converter.DNSRecord{
			Name: deviceDNSName(device.Name, domain, index), IP: address, TTL: dnsRecordTTL,
		}
	}
	return records
}

func deviceDNSName(device, domain string, addressIndex int) string {
	name := strings.ToLower(device)
	if addressIndex > 0 {
		name += fmt.Sprintf("-%d", addressIndex+1)
	}
	return name + "." + domain
}
