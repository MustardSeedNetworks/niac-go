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
  const { t: tDevices } = useTranslation('devices');
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
      title={tDevices('editor.sections.netbios.title')}
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
            <FormField
              label={tDevices('editor.sections.netbios.nameLabel')}
              helpText={tDevices('editor.sections.netbios.nameHelp')}
            >
              <input
                type="text"
                value={device.netbios.name || ''}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    name: e.target.value.toUpperCase().slice(0, 15),
                  })
                }
                placeholder={tDevices('editor.sections.netbios.namePlaceholder')}
                maxLength={15}
                className={`${inputClassName} uppercase`}
              />
            </FormField>

            <FormField
              label={tDevices('editor.sections.netbios.workgroupLabel')}
              helpText={tDevices('editor.sections.netbios.workgroupHelp')}
            >
              <input
                type="text"
                value={device.netbios.workgroup || ''}
                onChange={(e) =>
                  updateNetBios({
                    ...getNetbiosConfig(),
                    workgroup: e.target.value.toUpperCase(),
                  })
                }
                placeholder={tDevices('editor.sections.netbios.workgroupPlaceholder')}
                className={`${inputClassName} uppercase`}
              />
            </FormField>

            <FormField
              label={
                <>
                  {tDevices('editor.sections.netbios.nodeTypeLabel')}
                  <InfoPopover
                    label={t('jargon.ariaLabel', { term: 'NetBIOS node type' })}
                    title="NetBIOS node type"
                  >
                    {tHelp('jargon.netbiosNodeType')}
                  </InfoPopover>
                </>
              }
              helpText={tDevices('editor.sections.netbios.nodeTypeHelp')}
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
                <option value="B">{tDevices('editor.sections.netbios.nodeTypeB')}</option>
                <option value="P">{tDevices('editor.sections.netbios.nodeTypeP')}</option>
                <option value="M">{tDevices('editor.sections.netbios.nodeTypeM')}</option>
                <option value="H">{tDevices('editor.sections.netbios.nodeTypeH')}</option>
              </select>
            </FormField>

            <FormField
              label={tDevices('editor.sections.netbios.ttlLabel')}
              helpText={tDevices('editor.sections.netbios.ttlHelp')}
            >
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
              {tDevices('editor.sections.netbios.servicesTitle')}
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
                <span className="text-sm text-text-secondary">
                  {tDevices('editor.sections.netbios.masterBrowserLabel')}
                </span>
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
