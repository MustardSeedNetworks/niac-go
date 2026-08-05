package linklive

func interfaceFinding(
	kind FindingKind,
	device, iface, expected, observed string,
) Finding {
	return Finding{
		Kind: kind, Device: device, Interface: iface, Expected: expected, Observed: observed,
	}
}
