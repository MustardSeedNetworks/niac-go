package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

func internetDNSRecords() []converter.DNSRecord {
	hostnames := []string{
		"google.com", "www.google.com", "cloudflare.com",
		"www.msftconnecttest.com", "connectivitycheck.gstatic.com",
	}
	records := make([]converter.DNSRecord, len(hostnames))
	for index, hostname := range hostnames {
		records[index] = converter.DNSRecord{Name: hostname, IP: internetLoopback, TTL: dnsRecordTTL}
	}
	return records
}
