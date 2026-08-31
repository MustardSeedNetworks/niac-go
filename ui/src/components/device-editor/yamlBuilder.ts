import { stringify as stringifyYaml } from 'yaml';
import type { Device } from '../../api/types';
import { toDaemonDevice } from '../../utils/device-yaml';

/**
 * Build a YAML preview string from a Device object.
 *
 * The preview shows what the daemon would load, so it goes through the same
 * mapper as the export: `toDaemonDevice` owns the key names, and the `yaml`
 * package owns escaping. Hand-building this produced a document with the
 * wrong keys AND no escaping, so it was doubly unloadable.
 */
export const buildYamlPreview = (device: Device): string => {
  try {
    // lineWidth: 0 disables folding. A wrapped scalar is still valid YAML but
    // reads as a mangled value in a preview pane.
    return stringifyYaml({ devices: [toDaemonDevice(device)] }, { lineWidth: 0 }).trimEnd();
  } catch {
    return '# Error generating YAML preview';
  }
};
