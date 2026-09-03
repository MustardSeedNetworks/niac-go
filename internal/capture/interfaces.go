package capture

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/gopacket/gopacket/pcap"
)

// PcapInterface is an alias for pcap.Interface for use in other packages.
type PcapInterface = pcap.Interface

// ErrInterfaceNotFound is returned when a network interface is missing.
var ErrInterfaceNotFound = errors.New("interface not found")

// InterfaceExists checks if a network interface exists.
func InterfaceExists(name string) bool {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		logger := slog.Default()
		logger.Error("Error finding devices", "error", err)

		return false
	}

	return slices.ContainsFunc(devices, func(device pcap.Interface) bool {
		return device.Name == name
	})
}

// ListInterfaces writes all available network interfaces to w.
func ListInterfaces(w io.Writer) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		logger := slog.Default()
		logger.Error("Error finding devices", "error", err)

		return
	}

	if len(devices) == 0 {
		_, _ = fmt.Fprintln(w, "  No interfaces found")

		return
	}

	for _, device := range devices {
		_, _ = fmt.Fprintf(w, "  %s", device.Name)
		if device.Description != "" {
			_, _ = fmt.Fprintf(w, " - %s", device.Description)
		}

		_, _ = fmt.Fprintln(w)

		if len(device.Addresses) > 0 {
			for _, addr := range device.Addresses {
				_, _ = fmt.Fprintf(w, "    IP: %s", addr.IP)
				if addr.Netmask != nil {
					_, _ = fmt.Fprintf(w, "  Netmask: %s", addr.Netmask)
				}

				_, _ = fmt.Fprintln(w)
			}
		}
	}
}

// GetInterface returns information about a specific interface.
func GetInterface(name string) (*pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w", err)
	}

	for _, device := range devices {
		if device.Name == name {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrInterfaceNotFound, name)
}

// GetAllInterfaces returns all available network interfaces.
func GetAllInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w", err)
	}

	return devices, nil
}

// getUsableInterfacePrefixes returns prefixes for usable network interfaces.
// These include ethernet, WiFi, and loopback interfaces.
func getUsableInterfacePrefixes() []string {
	return []string{
		"eth",  // Linux ethernet
		"en",   // macOS ethernet/WiFi (en0, en1, etc.)
		"wlan", // Linux WiFi
		"wlp",  // Linux WiFi (systemd naming)
		"lo",   // Loopback (lo, lo0)
	}
}

// IsUsableInterface checks if an interface name matches usable prefixes.
func IsUsableInterface(name string) bool {
	return slices.ContainsFunc(getUsableInterfacePrefixes(), func(prefix string) bool {
		return len(name) >= len(prefix) && name[:len(prefix)] == prefix
	})
}

// GetUsableInterfaces returns only usable network interfaces (ethernet, WiFi, loopback).
func GetUsableInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w", err)
	}

	result := make([]pcap.Interface, 0, len(devices))
	for _, device := range devices {
		if IsUsableInterface(device.Name) {
			result = append(result, device)
		}
	}

	return result, nil
}
