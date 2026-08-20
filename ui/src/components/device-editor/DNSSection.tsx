import { Globe, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { DNSConfig, DNSRecord } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../form';
import type { ProtocolSectionProps } from './types';
import { smallInputClassName } from './types';

export const DnsSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getDnsConfig = (): DNSConfig => ({
    forwardRecords: [],
    reverseRecords: [],
    ...(device.dns ?? {}),
  });

  const updateDns = (config: DNSConfig | undefined) => {
    onUpdate('dns', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.dns.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={!!device.dns}
      onEnableChange={(enabled) => {
        if (enabled) {
          updateDns(getDnsConfig());
        } else {
          updateDns(undefined);
        }
      }}
    >
      {device.dns && (
        <div className="stack-xl">
          {/* Forward Records (A records) */}
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <Globe className={`${iconSizes.md} text-brand-accent`} />
              {t('editor.sections.dns.forwardRecordsTitle')}
            </h4>
            {(device.dns.forwardRecords || []).map((record: DNSRecord, index: number) => (
              <div
                key={`${record.name || record.ip || 'record'}`}
                className="flex gap-compact items-center"
              >
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = {
                      ...record,
                      name: e.target.value,
                    };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder={t('editor.sections.dns.hostnamePlaceholder')}
                  className={smallInputClassName}
                />
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = { ...record, ip: e.target.value };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder={t('editor.sections.dns.ipPlaceholder')}
                  className="w-40 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none font-mono"
                />
                <input
                  type="number"
                  value={record.ttl ?? 300}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = {
                      ...record,
                      ttl: Number.parseInt(e.target.value, 10),
                    };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder={t('editor.sections.dns.ttlPlaceholder')}
                  className="w-24 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const records = (device.dns?.forwardRecords || []).filter(
                      (_: DNSRecord, i: number) => i !== index,
                    );
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
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
                const records = [
                  ...(device.dns?.forwardRecords || []),
                  { name: '', ip: '', ttl: 300 } as DNSRecord,
                ];
                updateDns({ ...getDnsConfig(), forwardRecords: records });
              }}
            >
              {t('editor.sections.dns.addForwardButton')}
            </Button>
          </div>

          {/* Reverse Records (PTR records) */}
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <Globe className={`${iconSizes.md} text-brand-accent`} />
              {t('editor.sections.dns.reverseRecordsTitle')}
            </h4>
            {(device.dns.reverseRecords || []).map((record: DNSRecord, index: number) => (
              <div
                key={`${record.name || record.ip || 'record'}`}
                className="flex gap-compact items-center"
              >
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.reverseRecords || [])];
                    records[index] = { ...record, ip: e.target.value };
                    updateDns({ ...getDnsConfig(), reverseRecords: records });
                  }}
                  placeholder={t('editor.sections.dns.ipPlaceholder')}
                  className="w-40 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none font-mono"
                />
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.reverseRecords || [])];
                    records[index] = {
                      ...record,
                      name: e.target.value,
                    };
                    updateDns({ ...getDnsConfig(), reverseRecords: records });
                  }}
                  placeholder={t('editor.sections.dns.reverseHostnamePlaceholder')}
                  className={smallInputClassName}
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const records = (device.dns?.reverseRecords || []).filter(
                      (_: DNSRecord, i: number) => i !== index,
                    );
                    updateDns({ ...getDnsConfig(), reverseRecords: records });
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
                const records = [
                  ...(device.dns?.reverseRecords || []),
                  { ip: '', name: '', ttl: 300 } as DNSRecord,
                ];
                updateDns({ ...getDnsConfig(), reverseRecords: records });
              }}
            >
              {t('editor.sections.dns.addReverseButton')}
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
