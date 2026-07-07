import { Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { Device } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import { CollapsibleSection } from '../form/CollapsibleSection';
import { smallInputClassName } from './types';

export interface AdditionalIPsSectionProps {
  device: Device;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: <K extends keyof Device>(field: K, value: Device[K]) => void;
}

export const AdditionalIPsSection: FC<AdditionalIPsSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const handleIpChange = (index: number, value: string) => {
    const newIps = [...(device.ips || [])];
    newIps[index] = value;
    onUpdate('ips', newIps);
  };

  const handleRemoveIp = (index: number) => {
    const newIps = (device.ips || []).filter((_, i) => i !== index);
    onUpdate('ips', newIps);
  };

  const handleAddIp = () => {
    onUpdate('ips', [...(device.ips || []), '']);
  };

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
        {(device.ips || []).map((ip, index) => (
          <div key={`${ip || 'ip'}-${index}`} className="flex gap-compact">
            <input
              type="text"
              value={ip}
              onChange={(e) => handleIpChange(index, e.target.value)}
              placeholder={t('editor.sections.additionalIps.placeholder')}
              className={smallInputClassName}
            />
            <Button variant="ghost" tone="red" size="sm" onClick={() => handleRemoveIp(index)}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        ))}
        <Button
          variant="outline"
          size="sm"
          leftIcon={<Plus className={iconSizes.md} />}
          onClick={handleAddIp}
        >
          {t('editor.sections.additionalIps.addButton')}
        </Button>
      </div>
    </CollapsibleSection>
  );
};
