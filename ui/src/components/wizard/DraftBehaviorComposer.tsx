import { Activity, Plus, Save, Trash2 } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  type DraftBehaviorPhase,
  type DraftBehaviorTimeline,
  replaceScenarioDraftBehaviors,
  type ScenarioDraft,
} from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import { useErrorToast } from '../../hooks/useErrorToast';
import { Button } from '../../ui/Button';
import { Input } from '../../ui/Input';
import { SmallText } from '../../ui/Typography';
import { BehaviorPhaseActions } from './BehaviorPhaseActions';
import { parseDraftBehaviorTimelines } from './behavior-timeline';
import { parseDraftTopology } from './draft-topology';

interface DraftBehaviorComposerProps {
  draft: ScenarioDraft;
  onDraftUpdate: (draft: ScenarioDraft) => void;
  onBusyChange: (busy: boolean) => void;
}

function milliseconds(seconds: string) {
  return Math.round(Number(seconds) * 1000);
}

function seconds(value: number) {
  return String(value / 1000);
}

export const DraftBehaviorComposer: FC<DraftBehaviorComposerProps> = ({
  draft,
  onDraftUpdate,
  onBusyChange,
}) => {
  const { t } = useTranslation('pages');
  const showError = useErrorToast();
  const topology = useMemo(() => parseDraftTopology(draft.content), [draft.content]);
  const [timelines, setTimelines] = useState<DraftBehaviorTimeline[]>(() =>
    parseDraftBehaviorTimelines(draft.content),
  );
  const [busy, setBusy] = useState(false);

  useEffect(() => setTimelines(parseDraftBehaviorTimelines(draft.content)), [draft.content]);

  const deviceOptions = topology.devices.map((device) => ({
    value: device.name,
    label: device.name,
  }));
  // The first device is not necessarily a usable one: the Start-empty seed puts
  // an interfaceless host at position 0, which left this step permanently
  // disabled however many switches were added afterwards (#1491). Seed from the
  // first device that actually has an interface, since that is what a timeline
  // targets.
  const seedDevice = topology.devices.find(
    (device) => (topology.interfaces[device.name] ?? []).length > 0,
  );
  const firstDevice = seedDevice?.name ?? '';
  const firstInterface = firstDevice ? (topology.interfaces[firstDevice]?.[0]?.name ?? '') : '';
  const interfaceOptions = (device: string) =>
    (topology.interfaces[device] ?? []).map((iface) => ({ value: iface.name, label: iface.name }));

  const updateTimeline = (
    index: number,
    update: (current: DraftBehaviorTimeline) => DraftBehaviorTimeline,
  ) =>
    setTimelines((current) =>
      current.map((timeline, item) => (item === index ? update(timeline) : timeline)),
    );

  const updatePhase = (
    timelineIndex: number,
    phaseIndex: number,
    update: (current: DraftBehaviorPhase) => DraftBehaviorPhase,
  ) =>
    updateTimeline(timelineIndex, (timeline) => ({
      ...timeline,
      phases: timeline.phases.map((phase, item) => (item === phaseIndex ? update(phase) : phase)),
    }));

  const addPhase = (timelineIndex: number) =>
    updateTimeline(timelineIndex, (timeline) => ({
      ...timeline,
      phases: [
        ...timeline.phases,
        {
          name: t('newSimWizard.behaviors.defaultPhase'),
          startOffsetMs: timeline.phases.reduce(
            (end, phase) => Math.max(end, phase.startOffsetMs + phase.durationMs),
            0,
          ),
          durationMs: 30_000,
          reset: true,
          traffic: [],
          faults: [],
        },
      ],
    }));

  const addTimeline = () =>
    setTimelines((current) => [
      ...current,
      {
        name: t('newSimWizard.behaviors.defaultTimeline'),
        startOffsetMs: 0,
        repeatCount: 1,
        phases: [
          {
            name: t('newSimWizard.behaviors.defaultPhase'),
            startOffsetMs: 0,
            durationMs: 30_000,
            reset: true,
            traffic: firstDevice
              ? [{ device: firstDevice, interface: firstInterface, utilization: 75 }]
              : [],
            faults: [],
          },
        ],
      },
    ]);

  const valid = timelines.every(
    (timeline) =>
      timeline.name.trim() &&
      timeline.repeatCount >= 1 &&
      timeline.repeatCount <= 1000 &&
      timeline.startOffsetMs >= 0 &&
      timeline.phases.length > 0 &&
      timeline.phases.every(
        (phase) =>
          phase.name.trim() &&
          phase.startOffsetMs >= 0 &&
          phase.durationMs > 0 &&
          phase.traffic.length + phase.faults.length > 0 &&
          phase.traffic.every(
            (action) =>
              action.device &&
              action.interface &&
              action.utilization >= 1 &&
              action.utilization <= 100,
          ) &&
          phase.faults.every(
            (action) =>
              action.device && action.interface && action.value >= 1 && action.value <= 100,
          ),
      ),
  );

  const save = async () => {
    setBusy(true);
    onBusyChange(true);
    try {
      const updated = await replaceScenarioDraftBehaviors(draft.name, draft.revision, timelines);
      onDraftUpdate(updated);
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
      onBusyChange(false);
    }
  };

  return (
    <div className="stack" data-testid="behavior-composer">
      <div className="flex flex-wrap items-center justify-between gap-default">
        <SmallText className="text-text-muted">{t('newSimWizard.behaviors.help')}</SmallText>
        <Button
          variant="outline"
          leftIcon={<Plus className={iconSizes.md} />}
          disabled={busy || !firstDevice || !firstInterface}
          // A timeline targets a device interface, so there is nothing to
          // schedule until one exists. Without this the empty state told the
          // user to click a permanently disabled button (D4).
          title={
            !firstDevice || !firstInterface ? t('newSimWizard.behaviors.needsInterface') : undefined
          }
          onClick={addTimeline}
        >
          {t('newSimWizard.behaviors.addTimeline')}
        </Button>
      </div>

      {timelines.length === 0 && (
        <div className="rounded-lg border border-dashed border-surface-border p-6 text-center">
          <Activity className={`${iconSizes.xl} mx-auto text-brand-accent`} />
          <SmallText className="mt-tight text-text-muted">
            {t('newSimWizard.behaviors.empty')}
          </SmallText>
          {(!firstDevice || !firstInterface) && (
            <SmallText className="mt-tight block text-text-muted">
              {t('newSimWizard.behaviors.needsInterface')}
            </SmallText>
          )}
        </div>
      )}

      {timelines.map((timeline, timelineIndex) => (
        <section
          key={`${timelineIndex}-${timeline.name}`}
          className="stack rounded-lg border border-surface-border bg-bg-base/40 p-4"
        >
          <div className="grid gap-default md:grid-cols-[2fr_1fr_1fr_auto] md:items-end">
            <Input
              label={t('newSimWizard.behaviors.timelineName')}
              value={timeline.name}
              onChange={(event) =>
                updateTimeline(timelineIndex, (current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
            />
            <Input
              label={t('newSimWizard.behaviors.startsAfter')}
              type="number"
              min={0}
              step="0.1"
              value={seconds(timeline.startOffsetMs)}
              onChange={(event) =>
                updateTimeline(timelineIndex, (current) => ({
                  ...current,
                  startOffsetMs: milliseconds(event.target.value),
                }))
              }
            />
            <Input
              label={t('newSimWizard.behaviors.repetitions')}
              type="number"
              min={1}
              max={1000}
              value={timeline.repeatCount}
              onChange={(event) =>
                updateTimeline(timelineIndex, (current) => ({
                  ...current,
                  repeatCount: Number(event.target.value),
                }))
              }
            />
            <Button
              variant="outline"
              tone="red"
              aria-label={t('newSimWizard.behaviors.removeTimeline')}
              onClick={() =>
                setTimelines((current) => current.filter((_, item) => item !== timelineIndex))
              }
            >
              <Trash2 className={iconSizes.md} />
            </Button>
          </div>

          {timeline.phases.map((phase, phaseIndex) => (
            <div
              key={`${phaseIndex}-${phase.name}`}
              className="stack border-l-2 border-brand-primary/40 pl-4"
            >
              <div className="grid gap-default md:grid-cols-4">
                <Input
                  label={t('newSimWizard.behaviors.phaseName')}
                  value={phase.name}
                  onChange={(event) =>
                    updatePhase(timelineIndex, phaseIndex, (current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
                <Input
                  label={t('newSimWizard.behaviors.phaseOffset')}
                  type="number"
                  min={0}
                  step="0.1"
                  value={seconds(phase.startOffsetMs)}
                  onChange={(event) =>
                    updatePhase(timelineIndex, phaseIndex, (current) => ({
                      ...current,
                      startOffsetMs: milliseconds(event.target.value),
                    }))
                  }
                />
                <Input
                  label={t('newSimWizard.behaviors.duration')}
                  type="number"
                  min={0.1}
                  step="0.1"
                  value={seconds(phase.durationMs)}
                  onChange={(event) =>
                    updatePhase(timelineIndex, phaseIndex, (current) => ({
                      ...current,
                      durationMs: milliseconds(event.target.value),
                    }))
                  }
                />
                <div className="flex items-end justify-between gap-default">
                  <label className="flex min-h-11 items-center gap-tight text-sm text-text-secondary">
                    <input
                      type="checkbox"
                      checked={phase.reset}
                      onChange={(event) =>
                        updatePhase(timelineIndex, phaseIndex, (current) => ({
                          ...current,
                          reset: event.target.checked,
                        }))
                      }
                    />
                    {t('newSimWizard.behaviors.reset')}
                  </label>
                  <Button
                    variant="outline"
                    tone="red"
                    aria-label={t('newSimWizard.behaviors.removePhase')}
                    onClick={() =>
                      updateTimeline(timelineIndex, (current) => ({
                        ...current,
                        phases: current.phases.filter((_, item) => item !== phaseIndex),
                      }))
                    }
                  >
                    <Trash2 className={iconSizes.md} />
                  </Button>
                </div>
              </div>

              <BehaviorPhaseActions
                phase={phase}
                deviceOptions={deviceOptions}
                interfaceOptions={interfaceOptions}
                firstDevice={firstDevice}
                firstInterface={firstInterface}
                onChange={(next) => updatePhase(timelineIndex, phaseIndex, () => next)}
              />
            </div>
          ))}

          <Button size="sm" variant="outline" onClick={() => addPhase(timelineIndex)}>
            {t('newSimWizard.behaviors.addPhase')}
          </Button>
        </section>
      ))}

      <div>
        <Button
          tone="violet"
          leftIcon={<Save className={iconSizes.md} />}
          disabled={busy || !valid || timelines.length === 0}
          loading={busy}
          data-testid="save-behaviors"
          onClick={() => void save()}
        >
          {t('newSimWizard.behaviors.save')}
        </Button>
      </div>
    </div>
  );
};
