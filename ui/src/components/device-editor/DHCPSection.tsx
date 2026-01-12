import { type FC } from 'react';
import { Database, Plus, X } from 'lucide-react';
import { Button } from '../../ui';
import { CollapsibleSection, FormField } from '../form';
import type { DHCPConfig, DHCPLease } from '../../api/types';
import type { ProtocolSectionProps } from './types';
import { monoInputClassName, smallInputClassName } from './types';

export const DHCPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateDHCP = (config: DHCPConfig | undefined) => {
    onUpdate('dhcp', config);
  };

  return (
    <CollapsibleSection
      title="DHCP Server"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={!!device.dhcp}
      onEnableChange={(enabled) => {
        if (enabled) {
          updateDHCP({ subnet_mask: '255.255.255.0' } as DHCPConfig);
        } else {
          updateDHCP(undefined);
        }
      }}
    >
      {device.dhcp && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Subnet Mask" helpText="DHCP subnet mask">
              <input
                type="text"
                value={device.dhcp.subnet_mask || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, subnet_mask: e.target.value })
                }
                placeholder="255.255.255.0"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Default Gateway" helpText="Router/gateway for clients">
              <input
                type="text"
                value={device.dhcp.router || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, router: e.target.value })
                }
                placeholder="192.168.1.1"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="DNS Server" helpText="Domain name server for clients">
              <input
                type="text"
                value={device.dhcp.domain_name_server || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, domain_name_server: e.target.value })
                }
                placeholder="8.8.8.8"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Domain Name" helpText="Domain suffix for clients">
              <input
                type="text"
                value={device.dhcp.domain_name || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, domain_name: e.target.value })
                }
                placeholder="example.local"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Pool Start" helpText="Start of DHCP address pool">
              <input
                type="text"
                value={device.dhcp.pool_start || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, pool_start: e.target.value })
                }
                placeholder="192.168.1.100"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Pool End" helpText="End of DHCP address pool">
              <input
                type="text"
                value={device.dhcp.pool_end || ''}
                onChange={(e) =>
                  updateDHCP({ ...device.dhcp!, pool_end: e.target.value })
                }
                placeholder="192.168.1.200"
                className={monoInputClassName}
              />
            </FormField>
          </div>

          {/* Static Leases */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Database className="h-4 w-4 text-violet-400" />
              Static Leases
            </h4>
            {(device.dhcp.client_leases || []).map((lease, index) => (
              <div key={index} className="flex gap-2 items-center">
                <input
                  type="text"
                  value={lease.mac_address || ''}
                  onChange={(e) => {
                    const leases = [...(device.dhcp!.client_leases || [])];
                    leases[index] = { ...leases[index], mac_address: e.target.value };
                    updateDHCP({ ...device.dhcp!, client_leases: leases });
                  }}
                  placeholder="MAC Address"
                  className={smallInputClassName}
                />
                <input
                  type="text"
                  value={lease.client_ip || ''}
                  onChange={(e) => {
                    const leases = [...(device.dhcp!.client_leases || [])];
                    leases[index] = { ...leases[index], client_ip: e.target.value };
                    updateDHCP({ ...device.dhcp!, client_leases: leases });
                  }}
                  placeholder="IP Address"
                  className={smallInputClassName}
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const leases = (device.dhcp!.client_leases || []).filter((_, i) => i !== index);
                    updateDHCP({ ...device.dhcp!, client_leases: leases });
                  }}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                const leases = [...(device.dhcp!.client_leases || []), { mac_address: '', client_ip: '' } as DHCPLease];
                updateDHCP({ ...device.dhcp!, client_leases: leases });
              }}
            >
              Add Static Lease
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
