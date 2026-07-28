package scenario_test

import (
	"math"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestGenerateRejectsOversizedRepeatedFields(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		mutate func(*scenario.Request)
	}{
		{"domain", "domain", func(r *scenario.Request) { r.Domain = strings.Repeat("a.", 118) + "aa" }},
		{"SNMP community", "SNMP community", func(r *scenario.Request) { r.SNMPCommunity = strings.Repeat("c", 256) }},
		{"attachment name", "attachment name", func(r *scenario.Request) {
			r.AttachmentName = strings.Repeat("a", 65)
		}},
		{"site location", "location", func(r *scenario.Request) { r.Sites[0].Location = strings.Repeat("l", 129) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := scenario.EnterpriseReferenceRequest()
			test.mutate(&request)
			if _, err := scenario.Generate(request); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Generate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGenerateRejectsAmbiguousSiteIdentity(t *testing.T) {
	request := scenario.EnterpriseReferenceRequest()
	request.Sites[1].Code = request.Sites[0].Code
	if _, err := scenario.Generate(request); err == nil || !strings.Contains(err.Error(), "site code") {
		t.Fatalf("Generate() error = %v, want duplicate site code", err)
	}
	request = scenario.EnterpriseReferenceRequest()
	request.Sites[1].Octet = request.Sites[0].Octet
	if _, err := scenario.Generate(request); err == nil || !strings.Contains(err.Error(), "site octet") {
		t.Fatalf("Generate() error = %v, want duplicate site octet", err)
	}
}

func TestGenerateRejectsImpossibleFleetCounts(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		mutate func(*scenario.Request)
	}{
		{"site limit", "sites must contain 1 to 4 entries", func(r *scenario.Request) {
			r.Sites = append(r.Sites, scenario.Site{Code: "NYC", Octet: 244, Location: "New York, NY"})
		}},
		{"unequal redundancy", "WAN, firewall, and core counts must match", func(r *scenario.Request) {
			r.Counts.Firewalls = 1
		}},
		{"peer limit", "WAN, firewall, and core counts must be 1 or 2", func(r *scenario.Request) {
			r.Counts.SiteWANRouters, r.Counts.Firewalls, r.Counts.CoreSwitches = 3, 3, 3
		}},
		{"excessive peer count", "WAN, firewall, and core counts must be 1 or 2", func(r *scenario.Request) {
			r.Counts.SiteWANRouters, r.Counts.Firewalls, r.Counts.CoreSwitches = math.MaxInt, math.MaxInt, math.MaxInt
		}},
		{"distribution pairs", "distribution switch count must be even", func(r *scenario.Request) {
			r.Counts.DistributionSwitches = 3
		}},
		{"access limit", "access switch count must be between 1 and 20", func(r *scenario.Request) {
			r.Counts.AccessSwitches = 21
		}},
		{"AP port limit", "access points per access switch must be between 0 and 9", func(r *scenario.Request) {
			r.Counts.AccessPointsPerAccess = 10
		}},
		{"workstation pool", "site workstation count must not exceed 79", func(r *scenario.Request) {
			r.Counts.WorkstationsPerAccess = 5
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := scenario.EnterpriseReferenceRequest()
			test.mutate(&request)
			if _, err := scenario.Generate(request); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Generate() error = %v, want %q", err, test.want)
			}
		})
	}
}
