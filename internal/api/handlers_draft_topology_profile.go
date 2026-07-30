package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/drafttopology"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const (
	bitsPerMegabit      = 1_000_000
	capturedDefaultMbps = 1_000
	capturedStandardMTU = 1_500
)

func (s *Server) enrichCapturedDraftProfile(
	cfg *config.Config,
	mutation *drafttopology.Mutation,
) error {
	if mutation.Operation != drafttopology.AddDevice || mutation.Device == nil ||
		mutation.Device.ProfileRole == "" {
		return nil
	}
	profiles, err := scenario.CustomProfiles(s.library.Root())
	if err != nil {
		return errors.New("captured profile catalog is unavailable")
	}
	profile, ok := capturedProfileByRole(profiles, mutation.Device.ProfileRole)
	if !ok {
		return nil
	}
	walkRoot := s.library.SubDir(library.KindWalks)
	if !capturedWalkAvailable(walkRoot, profile.WalkName) {
		return errors.New("captured profile walk is unavailable")
	}
	if err = validateDraftWalkRoot(cfg, walkRoot); err != nil {
		return err
	}
	cfg.IncludePath = walkRoot
	applyCapturedProfile(mutation.Device, profile)
	return nil
}

func capturedProfileByRole(
	profiles []scenario.DeviceProfile,
	role string,
) (scenario.DeviceProfile, bool) {
	for _, profile := range profiles {
		if profile.Role == role {
			return profile, true
		}
	}
	return scenario.DeviceProfile{}, false
}

func capturedWalkAvailable(walkRoot, walkName string) bool {
	info, err := os.Lstat(filepath.Join(walkRoot, filepath.FromSlash(walkName)))
	return err == nil && info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular()
}

func validateDraftWalkRoot(cfg *config.Config, walkRoot string) error {
	for _, device := range cfg.Devices {
		for _, existing := range configuredWalks(device.SNMPConfig) {
			resolved := existing
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(cfg.IncludePath, resolved)
			}
			relative, err := filepath.Rel(walkRoot, resolved)
			if err != nil || strings.HasPrefix(relative, "..") {
				return errors.New("draft already uses walks from another content location")
			}
		}
	}
	return nil
}

func configuredWalks(snmpConfig config.SNMPConfig) []string {
	walks := append([]string{}, snmpConfig.WalkFiles...)
	if snmpConfig.WalkFile != "" {
		walks = append(walks, snmpConfig.WalkFile)
	}
	for _, include := range snmpConfig.CommunityIncludes {
		if include.WalkFile != "" {
			walks = append(walks, include.WalkFile)
		}
	}
	return walks
}

func applyCapturedProfile(device *drafttopology.DeviceMutation, profile scenario.DeviceProfile) {
	device.Type = profile.DeviceType
	device.Vendor = profile.Vendor
	device.SysObjectID = profile.SysObjectID
	device.WalkFile = profile.WalkName
	if len(profile.Interfaces) > 0 {
		device.Interfaces = capturedProfileInterfaces(profile.Interfaces)
	}
	if device.Properties == nil {
		device.Properties = make(map[string]string)
	}
	device.Properties["role"] = profile.Role
	device.Properties["model"] = profile.Model
	device.Properties["platform"] = profile.Platform
	device.Properties["software"] = profile.Software
}

func capturedProfileInterfaces(interfaces []scenario.ProfileInterface) []drafttopology.Interface {
	result := make([]drafttopology.Interface, 0, len(interfaces))
	for _, iface := range interfaces {
		result = append(result, capturedProfileInterface(iface))
	}
	return result
}

func capturedProfileInterface(iface scenario.ProfileInterface) drafttopology.Interface {
	speed, mtu := capturedDefaultMbps, iface.MTU
	if iface.Speed > 0 {
		speed = int(iface.Speed / bitsPerMegabit)
	}
	if mtu == 0 {
		mtu = capturedStandardMTU
	}
	adminStatus := capturedAdminStatus(iface.AdminStatus)
	operStatus := capturedOperStatus(iface.OperStatus)
	return drafttopology.Interface{
		Name: iface.Name, Type: iface.Type, Speed: speed, MTU: mtu,
		Duplex: "full", AdminStatus: adminStatus, OperStatus: operStatus,
	}
}

func capturedAdminStatus(status string) string {
	switch status {
	case "down", "testing":
		return status
	default:
		return "up"
	}
}

func capturedOperStatus(status string) string {
	switch status {
	case "up", "testing":
		return status
	default:
		return "down"
	}
}
