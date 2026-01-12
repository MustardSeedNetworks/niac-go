import { type FC } from 'react';
import { Globe, Plus, X } from 'lucide-react';
import { Button } from '../../ui';
import { CollapsibleSection } from '../form';
import type { DNSConfig, DNSRecord } from '../../api/types';
import type { ProtocolSectionProps } from './types';
import { smallInputClassName } from './types';

export const DNSSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateDNS = (config: DNSConfig | undefined) => {
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
          updateDNS({ forward_records: [], reverse_records: [] } as DNSConfig);
        } else {
          updateDNS(undefined);
        }
      }}
    >
      {device.dns && (
        <div className="space-y-6">
          {/* Forward Records (A records) */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Globe className="h-4 w-4 text-violet-400" />
              Forward Records (A Records)
            </h4>
            {(device.dns.forward_records || []).map((record, index) => (
              <div key={index} className="flex gap-2 items-center">
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns!.forward_records || [])];
                    records[index] = { ...records[index], name: e.target.value };
                    updateDNS({ ...device.dns!, forward_records: records });
                  }}
                  placeholder="Hostname (e.g., www.example.com)"
                  className={smallInputClassName}
                />
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns!.forward_records || [])];
                    records[index] = { ...records[index], ip: e.target.value };
                    updateDNS({ ...device.dns!, forward_records: records });
                  }}
                  placeholder="IP Address"
                  className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
                <input
                  type="number"
                  value={record.ttl ?? 300}
                  onChange={(e) => {
                    const records = [...(device.dns!.forward_records || [])];
                    records[index] = { ...records[index], ttl: parseInt(e.target.value) };
                    updateDNS({ ...device.dns!, forward_records: records });
                  }}
                  placeholder="TTL"
                  className="w-24 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const records = (device.dns!.forward_records || []).filter((_, i) => i !== index);
                    updateDNS({ ...device.dns!, forward_records: records });
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
                const records = [...(device.dns!.forward_records || []), { name: '', ip: '', ttl: 300 } as DNSRecord];
                updateDNS({ ...device.dns!, forward_records: records });
              }}
            >
              Add Forward Record
            </Button>
          </div>

          {/* Reverse Records (PTR records) */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Globe className="h-4 w-4 text-violet-400" />
              Reverse Records (PTR Records)
            </h4>
            {(device.dns.reverse_records || []).map((record, index) => (
              <div key={index} className="flex gap-2 items-center">
                <input
                  type="text"
                  value={record.ip || ''}
                  onChange={(e) => {
                    const records = [...(device.dns!.reverse_records || [])];
                    records[index] = { ...records[index], ip: e.target.value };
                    updateDNS({ ...device.dns!, reverse_records: records });
                  }}
                  placeholder="IP Address"
                  className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
                <input
                  type="text"
                  value={record.name || ''}
                  onChange={(e) => {
                    const records = [...(device.dns!.reverse_records || [])];
                    records[index] = { ...records[index], name: e.target.value };
                    updateDNS({ ...device.dns!, reverse_records: records });
                  }}
                  placeholder="Hostname (e.g., server.example.com)"
                  className={smallInputClassName}
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const records = (device.dns!.reverse_records || []).filter((_, i) => i !== index);
                    updateDNS({ ...device.dns!, reverse_records: records });
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
                const records = [...(device.dns!.reverse_records || []), { ip: '', name: '', ttl: 300 } as DNSRecord];
                updateDNS({ ...device.dns!, reverse_records: records });
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
