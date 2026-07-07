import { Radio } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { TrafficConfig } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';

export const TrafficSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getTrafficConfig = (): TrafficConfig => ({
    enabled: true,
    arpAnnouncements: {
      enabled: device.traffic?.arpAnnouncements?.enabled ?? false,
      interval: device.traffic?.arpAnnouncements?.interval ?? 60,
    },
    periodicPings: {
      enabled: device.traffic?.periodicPings?.enabled ?? false,
      interval: device.traffic?.periodicPings?.interval ?? 30,
      payloadSize: device.traffic?.periodicPings?.payloadSize ?? 56,
    },
    randomTraffic: {
      enabled: device.traffic?.randomTraffic?.enabled ?? false,
      interval: device.traffic?.randomTraffic?.interval ?? 60,
      packetCount: device.traffic?.randomTraffic?.packetCount ?? 5,
      patterns: device.traffic?.randomTraffic?.patterns ?? [],
    },
    ...(device.traffic ?? {}),
  });

  const updateTraffic = (config: TrafficConfig | undefined) => {
    onUpdate('traffic', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.traffic.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.traffic?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateTraffic(enabled ? getTrafficConfig() : undefined);
      }}
    >
      {device.traffic?.enabled && (
        <div className="stack-xl">
          {/* ARP Announcements */}
          <div className="rounded-lg border border-surface-border bg-bg-base/40 pad stack">
            <div className="flex-between">
              <h4 className="label flex items-center gap-compact">
                <Radio className={`${iconSizes.md} text-brand-accent`} />
                {t('editor.sections.traffic.arpAnnouncementsTitle')}
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.arpAnnouncements?.enabled ?? false}
                  onChange={(e) => {
                    const baseConfig = getTrafficConfig();
                    updateTraffic({
                      ...baseConfig,
                      arpAnnouncements: {
                        ...baseConfig.arpAnnouncements,
                        enabled: e.target.checked,
                      },
                    });
                  }}
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-primary peer-focus:ring-2 peer-focus:ring-brand-primary transition-colors">
                  <div
                    className={`absolute top-0.5 left-0.5 w-4 h-4 bg-knob rounded-full transition-transform ${device.traffic.arpAnnouncements?.enabled ? 'translate-x-4' : ''}`}
                  />
                </div>
              </label>
            </div>
            {device.traffic.arpAnnouncements?.enabled && (
              <FormField
                label={t('editor.sections.traffic.intervalLabel')}
                helpText={t('editor.sections.traffic.arpIntervalHelp')}
              >
                <input
                  type="number"
                  value={device.traffic.arpAnnouncements.interval ?? 60}
                  onChange={(e) => {
                    const baseConfig = getTrafficConfig();
                    updateTraffic({
                      ...baseConfig,
                      arpAnnouncements: {
                        enabled: baseConfig.arpAnnouncements?.enabled ?? true,
                        interval: Number.parseInt(e.target.value, 10),
                      },
                    });
                  }}
                  min={1}
                  className="w-32 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                />
              </FormField>
            )}
          </div>

          {/* Periodic Pings */}
          <div className="rounded-lg border border-surface-border bg-bg-base/40 pad stack">
            <div className="flex-between">
              <h4 className="label flex items-center gap-compact">
                <Radio className={`${iconSizes.md} text-brand-accent`} />
                {t('editor.sections.traffic.periodicPingsTitle')}
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.periodicPings?.enabled ?? false}
                  onChange={(e) => {
                    const baseConfig = getTrafficConfig();
                    updateTraffic({
                      ...baseConfig,
                      periodicPings: {
                        ...baseConfig.periodicPings,
                        enabled: e.target.checked,
                      },
                    });
                  }}
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-primary peer-focus:ring-2 peer-focus:ring-brand-primary transition-colors">
                  <div
                    className={`absolute top-0.5 left-0.5 w-4 h-4 bg-knob rounded-full transition-transform ${device.traffic.periodicPings?.enabled ? 'translate-x-4' : ''}`}
                  />
                </div>
              </label>
            </div>
            {device.traffic.periodicPings?.enabled && (
              <div className="grid gap-comfortable md:grid-cols-2">
                <FormField
                  label={t('editor.sections.traffic.intervalLabel')}
                  helpText={t('editor.sections.traffic.pingIntervalHelp')}
                >
                  <input
                    type="number"
                    value={device.traffic.periodicPings.interval ?? 30}
                    onChange={(e) => {
                      const baseConfig = getTrafficConfig();
                      updateTraffic({
                        ...baseConfig,
                        periodicPings: {
                          enabled: baseConfig.periodicPings?.enabled ?? true,
                          interval: Number.parseInt(e.target.value, 10),
                          payloadSize: baseConfig.periodicPings?.payloadSize ?? 56,
                        },
                      });
                    }}
                    min={1}
                    className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                  />
                </FormField>
                <FormField
                  label={t('editor.sections.traffic.payloadSizeLabel')}
                  helpText={t('editor.sections.traffic.payloadSizeHelp')}
                >
                  <input
                    type="number"
                    value={device.traffic.periodicPings.payloadSize ?? 56}
                    onChange={(e) => {
                      const baseConfig = getTrafficConfig();
                      updateTraffic({
                        ...baseConfig,
                        periodicPings: {
                          enabled: baseConfig.periodicPings?.enabled ?? true,
                          interval: baseConfig.periodicPings?.interval ?? 30,
                          payloadSize: Number.parseInt(e.target.value, 10),
                        },
                      });
                    }}
                    min={0}
                    max={65507}
                    className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                  />
                </FormField>
              </div>
            )}
          </div>

          {/* Random Traffic */}
          <div className="rounded-lg border border-surface-border bg-bg-base/40 pad stack">
            <div className="flex-between">
              <h4 className="label flex items-center gap-compact">
                <Radio className={`${iconSizes.md} text-brand-accent`} />
                {t('editor.sections.traffic.randomTrafficTitle')}
              </h4>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={device.traffic.randomTraffic?.enabled ?? false}
                  onChange={(e) => {
                    const baseConfig = getTrafficConfig();
                    updateTraffic({
                      ...baseConfig,
                      randomTraffic: {
                        ...baseConfig.randomTraffic,
                        enabled: e.target.checked,
                      },
                    });
                  }}
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-primary peer-focus:ring-2 peer-focus:ring-brand-primary transition-colors">
                  <div
                    className={`absolute top-0.5 left-0.5 w-4 h-4 bg-knob rounded-full transition-transform ${device.traffic.randomTraffic?.enabled ? 'translate-x-4' : ''}`}
                  />
                </div>
              </label>
            </div>
            {device.traffic.randomTraffic?.enabled && (
              <div className="stack-lg">
                <div className="grid gap-comfortable md:grid-cols-2">
                  <FormField
                    label={t('editor.sections.traffic.intervalLabel')}
                    helpText={t('editor.sections.traffic.randomIntervalHelp')}
                  >
                    <input
                      type="number"
                      value={device.traffic.randomTraffic.interval ?? 60}
                      onChange={(e) => {
                        const baseConfig = getTrafficConfig();
                        updateTraffic({
                          ...baseConfig,
                          randomTraffic: {
                            enabled: baseConfig.randomTraffic?.enabled ?? true,
                            interval: Number.parseInt(e.target.value, 10),
                            packetCount: baseConfig.randomTraffic?.packetCount ?? 5,
                            patterns: baseConfig.randomTraffic?.patterns ?? [],
                          },
                        });
                      }}
                      min={1}
                      className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                    />
                  </FormField>
                  <FormField
                    label={t('editor.sections.traffic.packetCountLabel')}
                    helpText={t('editor.sections.traffic.packetCountHelp')}
                  >
                    <input
                      type="number"
                      value={device.traffic.randomTraffic.packetCount ?? 5}
                      onChange={(e) => {
                        const baseConfig = getTrafficConfig();
                        updateTraffic({
                          ...baseConfig,
                          randomTraffic: {
                            enabled: baseConfig.randomTraffic?.enabled ?? true,
                            interval: baseConfig.randomTraffic?.interval ?? 60,
                            packetCount: Number.parseInt(e.target.value, 10),
                            patterns: baseConfig.randomTraffic?.patterns ?? [],
                          },
                        });
                      }}
                      min={1}
                      max={100}
                      className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                    />
                  </FormField>
                </div>
                <div className="stack-sm">
                  <h5 className="text-xs font-medium text-text-muted">
                    {t('editor.sections.traffic.patternsLabel')}
                  </h5>
                  <div className="flex flex-wrap gap-comfortable">
                    {(['broadcast_arp', 'multicast', 'udp'] as const).map((pattern) => (
                      <label key={pattern} className="flex items-center gap-compact cursor-pointer">
                        <input
                          type="checkbox"
                          checked={(device.traffic?.randomTraffic?.patterns || []).includes(
                            pattern,
                          )}
                          onChange={(e) => {
                            const baseConfig = getTrafficConfig();
                            const patterns = baseConfig.randomTraffic?.patterns ?? [];
                            if (e.target.checked) {
                              updateTraffic({
                                ...baseConfig,
                                randomTraffic: {
                                  enabled: baseConfig.randomTraffic?.enabled ?? true,
                                  interval: baseConfig.randomTraffic?.interval ?? 60,
                                  packetCount: baseConfig.randomTraffic?.packetCount ?? 5,
                                  patterns: [...patterns, pattern],
                                },
                              });
                            } else {
                              updateTraffic({
                                ...baseConfig,
                                randomTraffic: {
                                  enabled: baseConfig.randomTraffic?.enabled ?? true,
                                  interval: baseConfig.randomTraffic?.interval ?? 60,
                                  packetCount: baseConfig.randomTraffic?.packetCount ?? 5,
                                  patterns: patterns.filter((p) => p !== pattern),
                                },
                              });
                            }
                          }}
                          className="w-4 h-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
                        />
                        <span className="text-sm text-text-secondary">
                          {pattern === 'broadcast_arp'
                            ? t('editor.sections.traffic.broadcastArp')
                            : pattern === 'multicast'
                              ? t('editor.sections.traffic.multicast')
                              : t('editor.sections.traffic.udpPattern')}
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
