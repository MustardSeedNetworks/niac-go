import { Database, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { DHCPConfig, DHCPLease } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
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
  const { t } = useTranslation('devices');
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
      title={t('editor.sections.dhcp.title')}
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
        <div className="stack-xl">
          <div className="grid gap-comfortable md:grid-cols-2">
            <FormField
              label={t('editor.sections.dhcp.subnetMaskLabel')}
              helpText={t('editor.sections.dhcp.subnetMaskHelp')}
            >
              <input
                type="text"
                value={device.dhcp.subnetMask || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), subnetMask: e.target.value })}
                placeholder="255.255.255.0"
                className={monoInputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.dhcp.gatewayLabel')}
              helpText={t('editor.sections.dhcp.gatewayHelp')}
            >
              <input
                type="text"
                value={device.dhcp.router || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), router: e.target.value })}
                placeholder="192.168.1.1"
                className={monoInputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.dhcp.dnsServerLabel')}
              helpText={t('editor.sections.dhcp.dnsServerHelp')}
            >
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

            <FormField
              label={t('editor.sections.dhcp.domainNameLabel')}
              helpText={t('editor.sections.dhcp.domainNameHelp')}
            >
              <input
                type="text"
                value={device.dhcp.domainName || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), domainName: e.target.value })}
                placeholder="example.local"
                className={monoInputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.dhcp.poolStartLabel')}
              helpText={t('editor.sections.dhcp.poolStartHelp')}
            >
              <input
                type="text"
                value={device.dhcp.poolStart || ''}
                onChange={(e) => updateDhcp({ ...getDhcpConfig(), poolStart: e.target.value })}
                placeholder="192.168.1.100"
                className={monoInputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.dhcp.poolEndLabel')}
              helpText={t('editor.sections.dhcp.poolEndHelp')}
            >
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
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <Database className={`${iconSizes.md} text-brand-accent`} />
              {t('editor.sections.dhcp.staticLeasesTitle')}
            </h4>
            {(device.dhcp.clientLeases || []).map((lease: DHCPLease, index: number) => (
              <div
                key={`${lease.macAddress || lease.clientIp || 'lease'}`}
                className="flex gap-compact items-center"
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
                  placeholder={t('editor.sections.dhcp.macPlaceholder')}
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
                  placeholder={t('editor.sections.dhcp.ipPlaceholder')}
                  className={smallInputClassName}
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const leases = (device.dhcp?.clientLeases || []).filter(
                      (_: DHCPLease, i: number) => i !== index,
                    );
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
              leftIcon={<Plus className={iconSizes.md} />}
              onClick={() => {
                const leases = [
                  ...(device.dhcp?.clientLeases || []),
                  { macAddress: '', clientIp: '' } as DHCPLease,
                ];
                updateDhcp({ ...getDhcpConfig(), clientLeases: leases });
              }}
            >
              {t('editor.sections.dhcp.addLeaseButton')}
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
