import { FileCog, Save } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ScenarioDraft } from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { YamlEditor } from '../config/YamlEditor';
import { DraftBehaviorComposer } from './DraftBehaviorComposer';
import { DraftTopologyComposer } from './DraftTopologyComposer';
import { parseDraftTopology } from './draft-topology';

interface DevicesStepProps {
  draftName: string;
  draft: ScenarioDraft;
  content: string;
  dirty: boolean;
  saving: boolean;
  onChange: (content: string) => void;
  onSave: () => Promise<boolean>;
  onDraftUpdate: (draft: ScenarioDraft) => void;
  onBusyChange: (busy: boolean) => void;
}

/** Step 2 edits the saved draft without touching the active runtime. */
export const DevicesStep: FC<DevicesStepProps> = ({
  draftName,
  draft,
  content,
  dirty,
  saving,
  onChange,
  onSave,
  onDraftUpdate,
  onBusyChange,
}) => {
  const { t } = useTranslation('pages');
  const visualSupported = useMemo(
    () => !parseDraftTopology(draft.content).configBacked,
    [draft.content],
  );
  const [view, setView] = useState<'visual' | 'behaviors' | 'yaml'>(() =>
    visualSupported ? 'visual' : 'yaml',
  );

  useEffect(() => {
    if (!visualSupported) setView('yaml');
  }, [visualSupported]);

  const switchView = async (next: 'visual' | 'behaviors' | 'yaml') => {
    if (next === view) return;
    if (next !== 'yaml' && !visualSupported) return;
    if (next !== 'yaml' && dirty && !(await onSave())) return;
    setView(next);
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-center gap-default">
          <FileCog className={`${iconSizes.lg} text-brand-accent`} />
          <div>
            <p className="font-medium text-text-primary">{draftName}</p>
            <SmallText className="text-text-muted">{t('newSimWizard.devices.help')}</SmallText>
          </div>
        </div>
        <div
          role="tablist"
          aria-label={t('newSimWizard.devices.viewLabel')}
          className="inline-flex w-fit rounded-lg border border-surface-border bg-bg-base/60 p-1"
        >
          {(['visual', 'behaviors', 'yaml'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              role="tab"
              data-testid={`wizard-view-${tab}`}
              aria-selected={view === tab}
              disabled={tab !== 'yaml' && !visualSupported}
              className={`min-h-11 rounded-md px-4 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary ${
                view === tab
                  ? 'bg-brand-primary text-on-brand'
                  : 'text-text-secondary hover:bg-surface-hover'
              }`}
              onClick={() => void switchView(tab)}
            >
              {t(`newSimWizard.devices.${tab}Label`)}
            </button>
          ))}
        </div>
        {!visualSupported && (
          <SmallText className="text-status-warning">
            {t('newSimWizard.topology.configBackedYamlOnly')}
          </SmallText>
        )}
        {view === 'visual' && visualSupported ? (
          <DraftTopologyComposer
            draft={draft}
            onDraftUpdate={onDraftUpdate}
            onBusyChange={onBusyChange}
          />
        ) : view === 'behaviors' && visualSupported ? (
          <DraftBehaviorComposer
            draft={draft}
            onDraftUpdate={onDraftUpdate}
            onBusyChange={onBusyChange}
          />
        ) : (
          <>
            <YamlEditor
              value={content}
              onChange={onChange}
              readOnly={saving}
              height="24rem"
              className="mt-heading"
            />
            <div>
              <Button
                tone="violet"
                leftIcon={<Save className={iconSizes.md} />}
                disabled={!dirty || saving}
                loading={saving}
                onClick={() => void onSave()}
              >
                {saving
                  ? t('newSimWizard.devices.savingLabel')
                  : t('newSimWizard.devices.saveLabel')}
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
};
