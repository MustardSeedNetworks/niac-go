package capture

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/gopacket/pcap"
)

// Sentinel errors for interfaces.
var ErrInterfaceNotFound = errors.New("interface not found")

// InterfaceExists checks if a network interface exists.
func InterfaceExists(name string) bool {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		slog.Error("Error finding devices", "error", err)

		return false
	}

	for _, device := range devices {
		if device.Name == name {
			return true
		}
	}

	return false
}

// ListInterfaces prints all available network interfaces.
func ListInterfaces() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		slog.Error("Error finding devices", "error", err)

		return
	}

	if len(devices) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "  No interfaces found")

		return
	}

	for _, device := range devices {
		_, _ = fmt.Fprintf(os.Stdout, "  %s", device.Name)
		if device.Description != "" {
			_, _ = fmt.Fprintf(os.Stdout, " - %s", device.Description)
		}

		_, _ = fmt.Fprintln(os.Stdout)

		if len(device.Addresses) > 0 {
			for _, addr := range device.Addresses {
				_, _ = fmt.Fprintf(os.Stdout, "    IP: %s", addr.IP)
				if addr.Netmask != nil {
					_, _ = fmt.Fprintf(os.Stdout, "  Netmask: %s", addr.Netmask)
				}

				_, _ = fmt.Fprintln(os.Stdout)
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
