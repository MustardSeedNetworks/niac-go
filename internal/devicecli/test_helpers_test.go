package devicecli_test

import "net/netip"

func mustPrefix(value string) netip.Prefix {
	return netip.MustParsePrefix(value)
}

func mustAddr(value string) netip.Addr {
	return netip.MustParseAddr(value)
}
