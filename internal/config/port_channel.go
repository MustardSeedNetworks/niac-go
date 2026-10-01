package config

import (
	"strconv"
	"strings"
)

// Name is how a trunk port names the bundle.
func (channel PortChannel) Name() string {
	return "port-channel" + strconv.Itoa(channel.ID)
}

// PortChannelFor returns the bundle an interface name refers to. A trunk may
// spell it Port-channel1 where the schema says port-channel1, so the match
// ignores case.
func PortChannelFor(device *Device, interfaceName string) (PortChannel, bool) {
	for _, channel := range device.PortChannels {
		if strings.EqualFold(interfaceName, channel.Name()) {
			return channel, true
		}
	}
	return PortChannel{}, false
}
