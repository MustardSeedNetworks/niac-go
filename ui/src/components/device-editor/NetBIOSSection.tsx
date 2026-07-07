import { Network } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { NetBIOSConfig, NetBIOSService } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { InfoPopover } from '../../ui/InfoPopover';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const NetBiosSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('common');
  const { t: tHelp } = useTranslation('help');
  const getNetbiosConfig = (): NetBIOSConfig => ({
    enabled: true,
    nodeType: 'B',
    services: [],
    ...(device.netbios ?? {}),
  });

  const updateNetBios = (config: NetBIOSConfig | undefined) => {
    onUpdate('netbios', config);
  };

  return (
    <CollapsibleSection
      title="NetBIOS"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.netbios?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateNetBios(enabled ? getNetbiosConfig() : undefined);
      }}
    >
      {device.netbios?.enabled && (
        <div className="stack-xl">
          <div className="grid gap-comfortable md:grid-cols-2">
            <FormField label="NetBIOS Name" helpText="NetBIOS computer name (max 15 chars)">
              <input
                type="text"
                value={device.netbios.name || ''}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    name: e.target.value.toUpperCase().slice(0, 15),
                  })
                }
                placeholder="FILESERVER"
                maxLength={15}
                className={`${inputClassName} uppercase`}
              />
            </FormField>

            <FormField label="Workgroup" helpText="NetBIOS workgroup/domain">
              <input
                type="text"
                value={device.netbios.workgroup || ''}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    workgroup: e.target.value.toUpperCase(),
                  })
                }
                placeholder="WORKGROUP"
                className={`${inputClassName} uppercase`}
              />
            </FormField>

            <FormField
              label={
                <>
                  Node Type
                  <InfoPopover
                    label={t('jargon.ariaLabel', { term: 'NetBIOS node type' })}
                    title="NetBIOS node type"
                  >
                    {tHelp('jargon.netbiosNodeType')}
                  </InfoPopover>
                </>
              }
              helpText="NetBIOS node type"
            >
              <select
                value={device.netbios.nodeType || 'B'}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    nodeType: e.target.value as NetBIOSConfig['nodeType'],
                  })
                }
                className={inputClassName}
              >
                <option value="B">B-node (Broadcast)</option>
                <option value="P">P-node (Point-to-Point)</option>
                <option value="M">M-node (Mixed)</option>
                <option value="H">H-node (Hybrid)</option>
              </select>
            </FormField>

            <FormField label="TTL (seconds)" helpText="Time-to-live for NetBIOS announcements">
              <input
                type="number"
                value={device.netbios.ttl ?? 300000}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    ttl: Number.parseInt(e.target.value, 10),
                  })
                }
                min={60000}
                max={604800000}
                className={inputClassName}
              />
            </FormField>
          </div>

          {/* Services */}
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <Network className={`${iconSizes.md} text-brand-accent`} />
              NetBIOS Services
            </h4>
            <div className="flex flex-wrap gap-comfortable">
              {(['workstation', 'fileserver', 'messenger'] as NetBIOSService[]).map((service) => (
                <label key={service} className="flex items-center gap-compact cursor-pointer">
                  <input
                    type="checkbox"
                    checked={(device.netbios?.services || []).includes(service)}
                    onChange={(e) => {
                      const services = device.netbios?.services || [];
                      if (e.target.checked) {
                        updateNetBios({
                          ...getNetbiosConfig(),
                          services: [...services, service],
                        });
                      } else {
                        updateNetBios({
                          ...getNetbiosConfig(),
                          services: services.filter((s: NetBIOSService) => s !== service),
                        });
                      }
                    }}
                    className="w-4 h-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
                  />
                  <span className="text-sm text-text-secondary capitalize">{service}</span>
                </label>
              ))}
            </div>

            <div className="flex items-center gap-1 mt-inline">
              <label className="flex items-center gap-compact cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.netbios.msbrowse ?? false}
                  onChange={(e) =>
                    updateNetBios({
                      ...getNetbiosConfig(),
                      msbrowse: e.target.checked,
                    })
                  }
                  className="w-4 h-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
                />
                <span className="text-sm text-text-secondary">Master Browser (MSBROWSE)</span>
              </label>
              <InfoPopover label={t('jargon.ariaLabel', { term: 'MSBROWSE' })} title="MSBROWSE">
                {tHelp('jargon.msbrowse')}
              </InfoPopover>
            </div>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
