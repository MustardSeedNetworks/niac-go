import { Plus, X } from 'lucide-react';
import type { FC, MutableRefObject } from 'react';
import type { Device } from '../../api/types';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import { CollapsibleSection } from '../form/CollapsibleSection';
import { monoInputClassName } from './types';

export interface DeviceIpAddressesProps {
  device: Device;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: <K extends keyof Device>(field: K, value: Device[K]) => void;
  ipKeysRef: MutableRefObject<number[]>;
  ipKeyCounterRef: MutableRefObject<number>;
}

/**
 * Section for managing additional IP addresses
 */
export const DeviceIpAddresses: FC<DeviceIpAddressesProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  ipKeysRef,
  ipKeyCounterRef,
}) => {
  const ips = device.ips || [];

  // Ensure we have stable keys for all IPs
  while (ipKeysRef.current.length < ips.length) {
    ipKeysRef.current.push(ipKeyCounterRef.current++);
  }

  const handleIpChange = (index: number, value: string) => {
    const newIps = [...ips];
    newIps[index] = value;
    onUpdate('ips', newIps);
  };

  const handleRemoveIp = (index: number) => {
    const newIps = ips.filter((_, idx) => idx !== index);
    ipKeysRef.current = ipKeysRef.current.filter((_, idx) => idx !== index);
    onUpdate('ips', newIps);
  };

  const handleAddIp = () => {
    onUpdate('ips', [...ips, '']);
  };

  return (
    <CollapsibleSection title="Additional IP Addresses" isExpanded={isExpanded} onToggle={onToggle}>
      <div className="space-y-4">
        <SmallText className="text-gray-400">
          Add secondary IP addresses for multi-homed or VLAN configurations.
        </SmallText>
        {ips.map((ip, i) => (
          <div key={ipKeysRef.current[i]} className="flex gap-2">
            <input
              type="text"
              value={ip}
              onChange={(e) => handleIpChange(i, e.target.value)}
              placeholder="e.g., 192.168.2.1"
              className={`flex-1 ${monoInputClassName}`}
            />
            <Button variant="ghost" tone="red" size="sm" onClick={() => handleRemoveIp(i)}>
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
