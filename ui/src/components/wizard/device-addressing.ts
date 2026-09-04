import { isMap, isSeq, parseDocument } from 'yaml';
import { findDeviceFragment, spliceDeviceFragment } from '../../utils/device-fragment';

/**
 * setDeviceAddress puts a device on a network at a given address.
 *
 * Only the one device's block is re-serialized: `findDeviceFragment` gives its
 * byte range, so every other device, the sections around it and the comments
 * between them are copied through untouched. Within the block the `yaml`
 * Document is edited in place, which keeps that device's own comments too.
 *
 * The address is written to the device's first interface, creating the
 * interface list when the device has none. `address` is expected in prefix
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
  if (!isSeq(interfaces) || interfaces.items.length === 0) {
    doc.set('interfaces', [
      { name: 'Ethernet1/1', type: 'ethernet', network: networkName, address },
    ]);
  } else {
    const first = interfaces.items[0];
    if (!isMap(first)) {
      return configText;
    }
    first.set('network', networkName);
    first.set('address', address);
  }

  return spliceDeviceFragment(configText, fragment, String(doc));
}
