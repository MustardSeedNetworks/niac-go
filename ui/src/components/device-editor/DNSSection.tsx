import { Globe, Plus, X } from 'lucide-react';
import type { FC } from 'react';
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
      title="DNS Server"
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
        <div className="space-y-6">
          {/* Forward Records (A records) */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Globe className={`${iconSizes.md} text-violet-400`} />
              Forward Records (A Records)
            </h4>
            {(device.dns.forwardRecords || []).map((record: DNSRecord, index: number) => (
              <div
                key={`${record.name || record.ip || 'record'}`}
                className="flex gap-2 items-center"
              >
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = {
                      ...records[index],
                      name: e.target.value,
                    };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder="Hostname (e.g., www.example.com)"
                  className={smallInputClassName}
                />
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = { ...records[index], ip: e.target.value };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder="IP Address"
                  className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
                <input
                  type="number"
                  value={record.ttl ?? 300}
                  onChange={(e) => {
                    const records = [...(device.dns?.forwardRecords || [])];
                    records[index] = {
                      ...records[index],
                      ttl: Number.parseInt(e.target.value, 10),
                    };
                    updateDns({ ...getDnsConfig(), forwardRecords: records });
                  }}
                  placeholder="TTL"
                  className="w-24 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
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
              Add Forward Record
            </Button>
          </div>

          {/* Reverse Records (PTR records) */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Globe className={`${iconSizes.md} text-violet-400`} />
              Reverse Records (PTR Records)
            </h4>
            {(device.dns.reverseRecords || []).map((record: DNSRecord, index: number) => (
              <div
                key={`${record.name || record.ip || 'record'}`}
                className="flex gap-2 items-center"
              >
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.reverseRecords || [])];
                    records[index] = { ...records[index], ip: e.target.value };
                    updateDns({ ...getDnsConfig(), reverseRecords: records });
                  }}
                  placeholder="IP Address"
                  className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns?.reverseRecords || [])];
                    records[index] = {
                      ...records[index],
                      name: e.target.value,
                    };
                    updateDns({ ...getDnsConfig(), reverseRecords: records });
                  }}
                  placeholder="Hostname (e.g., server.example.com)"
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
              Add Reverse Record
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
