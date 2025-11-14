package capture

import (
	"fmt"
	"log"

	"github.com/google/gopacket/pcap"
)

// InterfaceExists checks if a network interface exists
func InterfaceExists(name string) bool {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Printf("Error finding devices: %v", err)
		return false
	}

	for _, device := range devices {
		if device.Name == name {
			return true
		}
	}
	return false
}

// ListInterfaces prints all available network interfaces
func ListInterfaces() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Printf("Error finding devices: %v", err)
		return
	}

	if len(devices) == 0 {
		fmt.Println("  No interfaces found")
		return
	}

	for _, device := range devices {
		fmt.Printf("  %s", device.Name)
		if device.Description != "" {
			fmt.Printf(" - %s", device.Description)
		}
		fmt.Println()

		if len(device.Addresses) > 0 {
			for _, addr := range device.Addresses {
				fmt.Printf("    IP: %s", addr.IP)
				if addr.Netmask != nil {
					fmt.Printf("  Netmask: %s", addr.Netmask)
				}
				fmt.Println()
			}
		}
	}
}

// GetInterface returns information about a specific interface
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

	return nil, fmt.Errorf("interface %s not found", name)
}

// GetAllInterfaces returns all available network interfaces
func GetAllInterfaces() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w", err)
	}
	return devices, nil
}
