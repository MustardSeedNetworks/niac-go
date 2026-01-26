import { Database, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import type { DHCPConfig, DHCPLease } from '../../api/types';
import { Button } from '../../ui/Button';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { monoInputClassName, smallInputClassName } from './types';

export const DhcpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const getDhcpConfig = (): DHCPConfig => ({
    subnetMask: '255.255.255.0',
    clientLeases: [],
    ...(device.dhcp ?? {}),
  });

  const updateDhcp = (config: DHCPConfig | undefined) => {
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
          updateDhcp(getDhcpConfig());
        } else {
          updateDhcp(undefined);
        }
      }}
    >
      {device.dhcp && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Subnet Mask" helpText="DHCP subnet mask">
              <input
                type="text"
                value={device.dhcp.subnetMask || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), subnetMask: e.target.value })}
                placeholder="255.255.255.0"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Default Gateway" helpText="Router/gateway for clients">
              <input
                type="text"
                value={device.dhcp.router || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), router: e.target.value })}
                placeholder="192.168.1.1"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="DNS Server" helpText="Domain name server for clients">
              <input
                type="text"
                value={device.dhcp.domainNameServer || ''}
                onChange={(e) =>
                  updateDhcp({
                    ...getDhcpConfig(),
                    domainNameServer: e.target.value,
                  })
                }
                placeholder="8.8.8.8"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Domain Name" helpText="Domain suffix for clients">
              <input
                type="text"
                value={device.dhcp.domainName || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), domainName: e.target.value })}
                placeholder="example.local"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Pool Start" helpText="Start of DHCP address pool">
              <input
                type="text"
                value={device.dhcp.poolStart || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), poolStart: e.target.value })}
                placeholder="192.168.1.100"
                className={monoInputClassName}
              />
            </FormField>

            <FormField label="Pool End" helpText="End of DHCP address pool">
              <input
                type="text"
                value={device.dhcp.poolEnd || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), poolEnd: e.target.value })}
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
            {(device.dhcp.clientLeases || []).map((lease: DHCPLease, index: number) => (
              <div
                key={`${lease.macAddress || lease.clientIp || 'lease'}`}
                className="flex gap-2 items-center"
              >
                <input
                  type="text"
                  value={lease.macAddress || ''}
                  onChange={(e) => {
                    const leases = [...(device.dhcp?.clientLeases || [])];
                    leases[index] = {
                      ...leases[index],
                      macAddress: e.target.value,
                    };
                    updateDhcp({ ...getDhcpConfig(), clientLeases: leases });
                  }}
                  placeholder="MAC Address"
                  className={smallInputClassName}
                />
                <input
                  type="text"
                  value={lease.clientIp || ''}
                  onChange={(e) => {
                    const leases = [...(device.dhcp?.clientLeases || [])];
                    leases[index] = {
                      ...leases[index],
                      clientIp: e.target.value,
                    };
                    updateDhcp({ ...getDhcpConfig(), clientLeases: leases });
                  }}
                  placeholder="IP Address"
                  className={smallInputClassName}
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const leases = (device.dhcp?.clientLeases || []).filter((_: DHCPLease, i: number) => i !== index);
                    updateDhcp({ ...getDhcpConfig(), clientLeases: leases });
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
                const leases = [
                  ...(device.dhcp?.clientLeases || []),
                  { macAddress: '', clientIp: '' } as DHCPLease,
                ];
                updateDhcp({ ...getDhcpConfig(), clientLeases: leases });
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
