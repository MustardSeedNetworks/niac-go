import { Plus, X } from 'lucide-react';
import { type FC, useRef } from 'react';
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
  const ips = device.ips ?? [];

  // Track a stable id per row so React reconciles inputs correctly when the
  // user removes a middle row (otherwise an index-keyed list would let the
  // wrong <input> keep focus). The ref is grown/shrunk in step with `ips`
  // length; on add we mint a new id, on remove we drop the id at the same
  // index the caller removed.
  const rowIdsRef = useRef<string[]>([]);
  while (rowIdsRef.current.length < ips.length) {
    rowIdsRef.current.push(crypto.randomUUID());
  }
  if (rowIdsRef.current.length > ips.length) {
    rowIdsRef.current = rowIdsRef.current.slice(0, ips.length);
  }

  const handleIpChange = (index: number, value: string) => {
    const newIps = [...ips];
    newIps[index] = value;
    onUpdate('ips', newIps);
  };

  const handleRemoveIp = (index: number) => {
    rowIdsRef.current = rowIdsRef.current.filter((_, i) => i !== index);
    onUpdate(
      'ips',
      ips.filter((_, i) => i !== index),
    );
  };

  const handleAddIp = () => {
    rowIdsRef.current.push(crypto.randomUUID());
    onUpdate('ips', [...ips, '']);
  };

  return (
    <CollapsibleSection title="Additional IP Addresses" isExpanded={isExpanded} onToggle={onToggle}>
      <div className="space-y-4">
        <SmallText className="text-gray-400">
          Add secondary IP addresses for multi-homed or VLAN configurations.
        </SmallText>
        {ips.map((ip, index) => (
          <div key={rowIdsRef.current[index]} className="flex gap-2">
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
