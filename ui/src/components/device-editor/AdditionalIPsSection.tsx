import { Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import { CollapsibleSection } from '../form/CollapsibleSection';
import type { AuthoredDevice, AuthoredValue } from './generated/authored-device.generated';
import { smallInputClassName } from './types';
import type { DeviceFieldErrors } from './useDeviceEditor';

export interface AdditionalIPsSectionProps {
  device: AuthoredDevice;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: (key: keyof AuthoredDevice, value: AuthoredValue) => void;
  errors?: DeviceFieldErrors;
}

/**
 * The device's addresses.
 *
 * `ips` is the whole list — the authored device has no separate primary
 * address, and the editor's old "Primary IP" field wrote a camelCase `ip` the
 * daemon's YAML has no key for.
 */
export const AdditionalIPsSection: FC<AdditionalIPsSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  errors,
}) => {
  const { t } = useTranslation('devices');
  const ips = device.ips ?? [];
  const replace = (next: readonly string[]) => onUpdate('ips', next.length > 0 ? next : undefined);

  return (
    <CollapsibleSection
      title={t('editor.sections.additionalIps.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
    >
      <div className="stack-lg">
        <SmallText className="text-text-muted">
          {t('editor.sections.additionalIps.description')}
        </SmallText>
        {errors?.ips && <SmallText className="text-status-error">{errors.ips}</SmallText>}
        {ips.map((ip, index) => (
          // Positional and reorderable by removal, and an entry being edited is
          // not unique, so the index is the identity.
          <div key={index} className="flex gap-compact">
            <input
              type="text"
              value={ip}
              aria-label={t('editor.sections.additionalIps.entryLabel', { index: index + 1 })}
              onChange={(e) =>
                replace(ips.map((entry, i) => (i === index ? e.target.value : entry)))
              }
              placeholder={t('editor.sections.additionalIps.placeholder')}
              className={smallInputClassName}
            />
            <Button
              variant="ghost"
              tone="red"
              size="sm"
              aria-label={t('editor.sections.additionalIps.removeLabel', { index: index + 1 })}
              onClick={() => replace(ips.filter((_, i) => i !== index))}
            >
              <X className={iconSizes.sm} />
            </Button>
          </div>
        ))}
        <Button
          variant="outline"
          size="sm"
          leftIcon={<Plus className={iconSizes.md} />}
          onClick={() => replace([...ips, ''])}
        >
          {t('editor.sections.additionalIps.addButton')}
        </Button>
      </div>
    </CollapsibleSection>
  );
};
