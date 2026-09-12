/**
 * Operator-facing names for device types, in the viewer's language.
 *
 * The legend and the topology header's type filter both name these, and the
 * filter used to render the raw value under a `capitalize` class — which reads
 * correctly for "Router" and wrongly for "Voip-Phone".
 *
 * Every key is written out as a literal rather than interpolated from the
 * device type. The i18n extractor resolves keys statically: a computed key is
 * invisible to it, and the next extraction silently drops those entries from
 * the catalogue along with their Spanish translations. (It reads comments too,
 * so the counter-example is described rather than written out here.)
 * `Record<DeviceType, string>` keeps the set exhaustive, so a device type added
 * to the schema fails to compile here until it is named.
 */

import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceType } from '../api/device-config-types';

export function useDeviceTypeLabels(): Record<DeviceType, string> {
  const { t, i18n } = useTranslation('pages');

  return useMemo(
    () => ({
      router: t('topology.deviceTypes.router'),
      switch: t('topology.deviceTypes.switch'),
      'layer3-switch': t('topology.deviceTypes.layer3-switch'),
      ap: t('topology.deviceTypes.ap'),
      'access-point': t('topology.deviceTypes.access-point'),
      firewall: t('topology.deviceTypes.firewall'),
      server: t('topology.deviceTypes.server'),
      host: t('topology.deviceTypes.host'),
      workstation: t('topology.deviceTypes.workstation'),
      iot: t('topology.deviceTypes.iot'),
      printer: t('topology.deviceTypes.printer'),
      'voip-phone': t('topology.deviceTypes.voip-phone'),
      unknown: t('topology.deviceTypes.unknown'),
    }),
    // i18n.language is the dependency that matters: `t` is stable across a
    // language change, so keying on it alone would serve stale labels.
    [t, i18n.language],
  );
}
