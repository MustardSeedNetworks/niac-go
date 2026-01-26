import { Plus, X } from 'lucide-react';
import type { FC } from 'react';
import type { Device } from '../../api/types';
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
    <CollapsibleSection title="Additional IP Addresses" isExpanded={isExpanded} onToggle={onToggle}>
      <div className="space-y-4">
        <SmallText className="text-gray-400">
          Add secondary IP addresses for multi-homed or VLAN configurations.
        </SmallText>
        {(device.ips || []).map((ip, index) => (
          <div key={`${ip || 'ip'}-${index}`} className="flex gap-2">
            <input
              type="text"
              value={ip}
              onChange={(e) => handleIpChange(index, e.target.value)}
              placeholder="e.g., 192.168.2.1"
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
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={handleAddIp}
        >
          Add IP Address
        </Button>
      </div>
    </CollapsibleSection>
  );
};
