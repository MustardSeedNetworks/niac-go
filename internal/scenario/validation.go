package scenario

import (
	"errors"
	"fmt"
	"strings"
)

func validateRequest(request Request) error {
	if len(request.Sites) == 0 || len(request.Sites) > maxSites {
		return fmt.Errorf("sites must contain 1 to %d entries", maxSites)
	}
	if !validDomainSuffix(request.Domain) {
		return fmt.Errorf("domain must be a valid DNS suffix no longer than %d bytes", maxScenarioDomainLength)
	}
	if strings.TrimSpace(request.SNMPCommunity) == "" || len(request.SNMPCommunity) > maxSNMPCommunityLength {
		return fmt.Errorf("SNMP community is required and must not exceed %d bytes", maxSNMPCommunityLength)
	}
	if strings.TrimSpace(request.AttachmentName) == "" || len(request.AttachmentName) > maxAttachmentNameLength {
		return fmt.Errorf("attachment name is required and must not exceed %d bytes", maxAttachmentNameLength)
	}
	if err := validateSites(request.Sites); err != nil {
		return err
	}
	return validateCounts(request.Counts)
}

func validateSites(sites []Site) error {
	codes := make(map[string]bool, len(sites))
	octets := make(map[int]bool, len(sites))
	for _, site := range sites {
		if !validSiteCode(site.Code) {
			return fmt.Errorf("site code %q must be 2 to 8 uppercase letters or digits", site.Code)
		}
		if codes[site.Code] {
			return fmt.Errorf("site code %q is duplicated", site.Code)
		}
		codes[site.Code] = true
		if site.Octet < 1 || site.Octet > 253 {
			return fmt.Errorf("site octet %d must be between 1 and 253", site.Octet)
		}
		if octets[site.Octet] {
			return fmt.Errorf("site octet %d is duplicated", site.Octet)
		}
		octets[site.Octet] = true
		if strings.TrimSpace(site.Location) == "" || len(site.Location) > maxSiteLocationLength {
			return fmt.Errorf(
				"site %s location is required and must not exceed %d bytes",
				site.Code, maxSiteLocationLength,
			)
		}
	}
	return nil
}

func validateCounts(c Counts) error {
	if c.SiteWANRouters != c.Firewalls || c.SiteWANRouters != c.CoreSwitches {
		return errors.New("WAN, firewall, and core counts must match")
	}
	if c.SiteWANRouters < 1 || c.SiteWANRouters > maxRedundantPeers {
		return errors.New("WAN, firewall, and core counts must be 1 or 2")
	}
	if c.DistributionSwitches < maxRedundantPeers || c.DistributionSwitches > maxDistributionSwitches {
		return errors.New("distribution switch count must be between 2 and 8")
	}
	if c.DistributionSwitches%maxRedundantPeers != 0 {
		return errors.New("distribution switch count must be even")
	}
	if c.AccessSwitches < 1 || c.AccessSwitches > maxAccessSwitches {
		return errors.New("access switch count must be between 1 and 20")
	}
	if c.ServerSwitches < 1 || c.ServerSwitches > maxServerSwitches {
		return errors.New("server switch count must be between 1 and 8")
	}
	if c.AccessPointsPerAccess < 0 || c.WorkstationsPerAccess < 0 || c.WirelessControllers < 0 {
		return errors.New("endpoint counts cannot be negative")
	}
	if c.AccessPointsPerAccess > maxAccessPointsPerAccess {
		return errors.New("access points per access switch must be between 0 and 9")
	}
	if c.WorkstationsPerAccess > maxWorkstationsPerAccess {
		return errors.New("workstations per access switch must be between 0 and 39")
	}
	if c.WirelessControllers > maxWirelessControllers {
		return errors.New("wireless controller count must be between 0 and 8")
	}
	if c.AccessSwitches*c.AccessPointsPerAccess > maxSiteAccessPoints {
		return fmt.Errorf("site access point count must not exceed %d", maxSiteAccessPoints)
	}
	if c.AccessSwitches*c.WorkstationsPerAccess > maxSiteWorkstations {
		return fmt.Errorf("site workstation count must not exceed %d", maxSiteWorkstations)
	}
	return nil
}

func validSiteCode(code string) bool {
	if len(code) < 2 || len(code) > 8 || code[0] < 'A' || code[0] > 'Z' {
		return false
	}
	for index := 1; index < len(code); index++ {
		if (code[index] < 'A' || code[index] > 'Z') && (code[index] < '0' || code[index] > '9') {
			return false
		}
	}
	return true
}

func validDomainSuffix(domain string) bool {
	if domain == "" || len(domain) > maxScenarioDomainLength || strings.TrimSpace(domain) != domain {
		return false
	}
	for label := range strings.SplitSeq(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for index := range len(label) {
			character := label[index]
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}
