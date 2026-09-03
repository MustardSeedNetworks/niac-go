import { type FC, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { parseAuthoredDevice, serializeAuthoredDevice } from '../../utils/authored-device-yaml';
import { findDeviceFragment, spliceDeviceFragment } from '../../utils/device-fragment';
import type {
  AuthoredDevice,
  AuthoredValue,
} from '../device-editor/generated/authored-device.generated';
import { DEVICE_SECTIONS } from '../device-editor/generated/sections.generated';
import { SchemaSectionBody } from '../device-editor/SchemaFields';
import { CollapsibleSection } from '../form/CollapsibleSection';

interface DeviceProtocolsEditorProps {
  content: string;
  onChange: (content: string) => void;
  /** Device names in the draft, in authored order. */
  devices: readonly string[];
}

/**
 * Every device's protocol blocks, authored with the same generated sections
 * the device editor uses.
 *
 * The wizard and the device editor render the identical `DEVICE_SECTIONS`
 * manifest, so a field one can set the other can too -- which is the point of
 * the parity work. Sections start collapsed: there are 28 of them per device
 * and expanding all of them for every device at once would render thousands
 * of inputs nobody asked for.
 *
 * Edits go back through the device's byte range, so the rest of the config --
 * other devices, the networks section, operator comments -- is untouched.
 */
export const DeviceProtocolsEditor: FC<DeviceProtocolsEditorProps> = ({
  content,
  onChange,
  devices,
}) => {
  const { t } = useTranslation('pages');
  const [expanded, setExpanded] = useState<string | null>(null);

  const authored = useMemo(() => {
    const byName = new Map<string, AuthoredDevice>();
    for (const name of devices) {
      const fragment = findDeviceFragment(content, name);
      // No fragment means the config did not parse or the device is not in
      // it; either way there is nothing to author, and a device that is
      // present always yields at least an empty document.
      if (!fragment) continue;
      byName.set(name, parseAuthoredDevice(fragment.text));
    }
    return byName;
  }, [content, devices]);

  const updateSection = (name: string, sectionKey: string, value: AuthoredValue) => {
    const fragment = findDeviceFragment(content, name);
    const device = authored.get(name);
    if (!fragment || !device) return;
    const next: AuthoredDevice = { ...device, [sectionKey]: value };
    onChange(spliceDeviceFragment(content, fragment, serializeAuthoredDevice(next)));
  };

  if (devices.length === 0) {
    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          <SmallText className="text-text-muted">{t('newSimWizard.networks.noDevices')}</SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="stack" data-testid="wizard-protocols-editor">
      {devices.map((name) => {
        const device = authored.get(name);
        if (!device) return null;
        return (
          <Card key={name} className="border-surface-border bg-bg-surface/70">
            <CardContent className="stack">
              <SmallText className="font-mono font-medium text-text-primary">{name}</SmallText>
              {DEVICE_SECTIONS.map((section) => {
                const key = `${name}:${section.key}`;
                return (
                  <CollapsibleSection
                    key={key}
                    title={section.title}
                    isExpanded={expanded === key}
                    onToggle={() => setExpanded(expanded === key ? null : key)}
                  >
                    <SchemaSectionBody
                      section={section}
                      value={device[section.key as keyof AuthoredDevice] as AuthoredValue}
                      onChange={(value) => updateSection(name, section.key, value)}
                    />
                  </CollapsibleSection>
                );
              })}
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
};
