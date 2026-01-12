import { type FC } from 'react';
import { Radio } from 'lucide-react';
import { CollapsibleSection, FormField } from '../form';
import type { TrafficConfig } from '../../api/types';
import type { ProtocolSectionProps } from './types';

export const TrafficSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateTraffic = (config: TrafficConfig | undefined) => {
    onUpdate('traffic', config);
  };

  return (
    <CollapsibleSection
      title="Traffic Patterns"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.traffic?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateTraffic(enabled ? { enabled: true } as TrafficConfig : undefined);
      }}
    >
      {device.traffic?.enabled && (
        <div className="space-y-6">
          {/* ARP Announcements */}
          <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Radio className="h-4 w-4 text-violet-400" />
                ARP Announcements
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.arp_announcements?.enabled ?? false}
                  onChange={(e) =>
                    updateTraffic({
                      ...device.traffic!,
                      arp_announcements: { ...device.traffic!.arp_announcements, enabled: e.target.checked }
                    })
                  }
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.arp_announcements?.enabled ? 'translate-x-4' : ''}`} />
                </div>
              </label>
            </div>
            {device.traffic.arp_announcements?.enabled && (
              <FormField label="Interval (seconds)" helpText="Time between ARP announcements">
                <input
                  type="number"
                  value={device.traffic.arp_announcements.interval ?? 60}
                  onChange={(e) =>
                    updateTraffic({
                      ...device.traffic!,
                      arp_announcements: { ...device.traffic!.arp_announcements!, interval: parseInt(e.target.value) }
                    })
                  }
                  min={1}
                  className="w-32 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>
            )}
          </div>

          {/* Periodic Pings */}
          <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Radio className="h-4 w-4 text-violet-400" />
                Periodic Pings
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.periodic_pings?.enabled ?? false}
                  onChange={(e) =>
                    updateTraffic({
                      ...device.traffic!,
                      periodic_pings: { ...device.traffic!.periodic_pings, enabled: e.target.checked }
                    })
                  }
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.periodic_pings?.enabled ? 'translate-x-4' : ''}`} />
                </div>
              </label>
            </div>
            {device.traffic.periodic_pings?.enabled && (
              <div className="grid gap-4 md:grid-cols-2">
                <FormField label="Interval (seconds)" helpText="Time between pings">
                  <input
                    type="number"
                    value={device.traffic.periodic_pings.interval ?? 30}
                    onChange={(e) =>
                      updateTraffic({
                        ...device.traffic!,
                        periodic_pings: { ...device.traffic!.periodic_pings!, interval: parseInt(e.target.value) }
                      })
                    }
                    min={1}
                    className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                </FormField>
                <FormField label="Payload Size (bytes)" helpText="Size of ICMP payload">
                  <input
                    type="number"
                    value={device.traffic.periodic_pings.payload_size ?? 56}
                    onChange={(e) =>
                      updateTraffic({
                        ...device.traffic!,
                        periodic_pings: { ...device.traffic!.periodic_pings!, payload_size: parseInt(e.target.value) }
                      })
                    }
                    min={0}
                    max={65507}
                    className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                </FormField>
              </div>
            )}
          </div>

          {/* Random Traffic */}
          <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Radio className="h-4 w-4 text-violet-400" />
                Random Traffic
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.random_traffic?.enabled ?? false}
                  onChange={(e) =>
                    updateTraffic({
                      ...device.traffic!,
                      random_traffic: { ...device.traffic!.random_traffic, enabled: e.target.checked }
                    })
                  }
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.random_traffic?.enabled ? 'translate-x-4' : ''}`} />
                </div>
              </label>
            </div>
            {device.traffic.random_traffic?.enabled && (
              <div className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <FormField label="Interval (seconds)" helpText="Time between traffic bursts">
                    <input
                      type="number"
                      value={device.traffic.random_traffic.interval ?? 60}
                      onChange={(e) =>
                        updateTraffic({
                          ...device.traffic!,
                          random_traffic: { ...device.traffic!.random_traffic!, interval: parseInt(e.target.value) }
                        })
                      }
                      min={1}
                      className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                  </FormField>
                  <FormField label="Packet Count" helpText="Packets per burst">
                    <input
                      type="number"
                      value={device.traffic.random_traffic.packet_count ?? 5}
                      onChange={(e) =>
                        updateTraffic({
                          ...device.traffic!,
                          random_traffic: { ...device.traffic!.random_traffic!, packet_count: parseInt(e.target.value) }
                        })
                      }
                      min={1}
                      max={100}
                      className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                  </FormField>
                </div>
                <div className="space-y-2">
                  <h5 className="text-xs font-medium text-gray-400">Traffic Patterns</h5>
                  <div className="flex flex-wrap gap-4">
                    {(['broadcast_arp', 'multicast', 'udp'] as const).map((pattern) => (
                      <label key={pattern} className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={(device.traffic!.random_traffic?.patterns || []).includes(pattern)}
                          onChange={(e) => {
                            const patterns = device.traffic!.random_traffic?.patterns || [];
                            if (e.target.checked) {
                              updateTraffic({
                                ...device.traffic!,
                                random_traffic: { ...device.traffic!.random_traffic!, patterns: [...patterns, pattern] }
                              });
                            } else {
                              updateTraffic({
                                ...device.traffic!,
                                random_traffic: { ...device.traffic!.random_traffic!, patterns: patterns.filter(p => p !== pattern) }
                              });
                            }
                          }}
                          className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-violet-600 focus:ring-violet-500"
                        />
                        <span className="text-sm text-gray-300">
                          {pattern === 'broadcast_arp' ? 'Broadcast ARP' : pattern === 'multicast' ? 'Multicast' : 'UDP'}
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
