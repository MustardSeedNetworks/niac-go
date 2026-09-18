import { isMap, isSeq, parseDocument } from 'yaml';
import { findDeviceFragment, spliceDeviceFragment } from '../../utils/device-fragment';

/** APs list their radios before their management uplink. Keep an existing
 * address/network on its interface; a new address belongs on a wired port. */
export function deviceAddressInterface(interfaces: unknown) {
  if (!isSeq(interfaces)) return undefined;
  const ports = interfaces.items.filter(isMap);
  return (
    ports.find((port) => typeof port.get('address') === 'string' && port.get('address') !== '') ??
    ports.find((port) => typeof port.get('network') === 'string' && port.get('network') !== '') ??
    ports.find((port) => port.get('type') !== 'ieee80211')
  );
}

/**
 * setDeviceAddress puts a device on a network at a given address.
 *
 * Only the one device's block is re-serialized: `findDeviceFragment` gives its
 * byte range, so every other device, the sections around it and the comments
 * between them are copied through untouched. Within the block the `yaml`
 * Document is edited in place, which keeps that device's own comments too.
 *
 * The address stays on the device's addressed interface, or its first wired
 * interface. A wired interface is created if it only has radios. `address` is expected in prefix
 * form -- the fabric compiler requires an interface address to carry its
 * network's prefix length.
 */
export function setDeviceAddress(
  configText: string,
  deviceName: string,
  networkName: string,
  address: string,
): string {
  const fragment = findDeviceFragment(configText, deviceName);
  if (!fragment) {
    return configText;
  }

  const doc = parseDocument(fragment.text);
  if (doc.errors.length > 0 || !isMap(doc.contents)) {
    return configText;
  }

  const interfaces = doc.get('interfaces');
  const target = deviceAddressInterface(interfaces);
  if (target) {
    target.set('network', networkName);
    target.set('address', address);
  } else {
    const wired = { name: 'Ethernet1/1', type: 'ethernet', network: networkName, address };
    if (isSeq(interfaces)) interfaces.add(doc.createNode(wired));
    else doc.set('interfaces', [wired]);
  }

  return spliceDeviceFragment(configText, fragment, String(doc));
}
