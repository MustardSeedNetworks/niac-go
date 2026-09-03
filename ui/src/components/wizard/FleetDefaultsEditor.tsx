import { type FC, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { spliceConfigSection } from '../../utils/config-section';
import { FormField } from '../form/FormField';
import {
  type CapturePlayback,
  type DiscoveryDefaults,
  parseFleetDefaults,
  serializeCapturePlaybacks,
  serializeDiscoveryProtocols,
} from './fleet-defaults';

interface FleetDefaultsEditorProps {
  content: string;
  onChange: (content: string) => void;
}

const inputClassName =
  'w-full rounded border border-surface-border bg-bg-surface px-3 py-row text-sm text-text-primary';

const DISCOVERY_PROTOCOLS = ['lldp', 'cdp', 'edp', 'fdp'] as const;

/**
 * The two config sections that had no control anywhere in the UI: the
 * fleet-wide discovery defaults, and the capture played back alongside the
 * simulated devices.
 *
 * Neither is a derived field, so the parity gate could not honestly allow-list
 * them; they were simply unreachable except by editing YAML by hand.
 */
export const FleetDefaultsEditor: FC<FleetDefaultsEditorProps> = ({ content, onChange }) => {
  const { t } = useTranslation('pages');
  const model = useMemo(() => parseFleetDefaults(content), [content]);

  const writeDiscovery = (discovery: DiscoveryDefaults) =>
    onChange(
      spliceConfigSection(content, 'discovery_protocols', serializeDiscoveryProtocols(discovery)),
    );

  const writePlaybacks = (playbacks: CapturePlayback[]) =>
    onChange(
      spliceConfigSection(content, 'capture_playbacks', serializeCapturePlaybacks(playbacks)),
    );

  // At most one playback: the runtime plays exactly one capture, and extra
  // entries were dropped on load and deleted from disk on the next save.
  const playback = model.capturePlaybacks[0] ?? null;

  const updatePlayback = (patch: Partial<CapturePlayback>) => {
    const next = { ...(playback ?? { fileName: '' }), ...patch };
    writePlaybacks(next.fileName.trim() === '' ? [] : [next]);
  };

  return (
    <div className="stack">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <SmallText className="font-medium text-text-primary">
            {t('newSimWizard.fleetDefaults.discoveryTitle')}
          </SmallText>
          <SmallText className="text-text-muted">
            {t('newSimWizard.fleetDefaults.discoveryHelp')}
          </SmallText>

          {DISCOVERY_PROTOCOLS.map((protocol) => {
            const current = model.discoveryProtocols[protocol];
            return (
              <div key={protocol} className="grid gap-comfortable md:grid-cols-3 items-end">
                <label
                  htmlFor={`discovery-${protocol}-enabled`}
                  className="flex items-center gap-compact text-sm text-text-secondary"
                >
                  <input
                    id={`discovery-${protocol}-enabled`}
                    type="checkbox"
                    className="w-4 h-4 rounded border-border-muted bg-bg-elevated text-brand-primary"
                    checked={current?.enabled === true}
                    onChange={(event) =>
                      writeDiscovery({
                        ...model.discoveryProtocols,
                        [protocol]: event.target.checked
                          ? { ...current, enabled: true }
                          : undefined,
                      })
                    }
                  />
                  {protocol.toUpperCase()}
                </label>
                <FormField
                  label={t('newSimWizard.fleetDefaults.interval')}
                  htmlFor={`discovery-${protocol}-interval`}
                >
                  <input
                    id={`discovery-${protocol}-interval`}
                    className={inputClassName}
                    inputMode="numeric"
                    disabled={current?.enabled !== true}
                    value={current?.interval ?? ''}
                    onChange={(event) =>
                      writeDiscovery({
                        ...model.discoveryProtocols,
                        [protocol]: {
                          enabled: true,
                          interval: event.target.value ? Number(event.target.value) : undefined,
                        },
                      })
                    }
                  />
                </FormField>
              </div>
            );
          })}
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <SmallText className="font-medium text-text-primary">
            {t('newSimWizard.fleetDefaults.captureTitle')}
          </SmallText>
          <SmallText className="text-text-muted">
            {t('newSimWizard.fleetDefaults.captureHelp')}
          </SmallText>

          <div className="grid gap-comfortable md:grid-cols-3 items-end">
            <FormField
              label={t('newSimWizard.fleetDefaults.fileName')}
              htmlFor="capture-playback-file-name"
            >
              <input
                id="capture-playback-file-name"
                className={inputClassName}
                value={playback?.fileName ?? ''}
                onChange={(event) => updatePlayback({ fileName: event.target.value })}
              />
            </FormField>
            <FormField
              label={t('newSimWizard.fleetDefaults.loopTime')}
              htmlFor="capture-playback-loop-time"
            >
              <input
                id="capture-playback-loop-time"
                className={inputClassName}
                inputMode="numeric"
                disabled={!playback}
                value={playback?.loopTime ?? ''}
                onChange={(event) =>
                  updatePlayback({
                    loopTime: event.target.value ? Number(event.target.value) : undefined,
                  })
                }
              />
            </FormField>
            <FormField
              label={t('newSimWizard.fleetDefaults.scaleTime')}
              htmlFor="capture-playback-scale-time"
            >
              <input
                id="capture-playback-scale-time"
                className={inputClassName}
                inputMode="decimal"
                disabled={!playback}
                value={playback?.scaleTime ?? ''}
                onChange={(event) =>
                  updatePlayback({
                    scaleTime: event.target.value ? Number(event.target.value) : undefined,
                  })
                }
              />
            </FormField>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
