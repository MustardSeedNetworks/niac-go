import type { ChangeEvent, MutableRefObject } from 'react';
import { useTranslation } from 'react-i18next';
import type { WalkCaptureCredentials } from '../../api/walk-profile-client';
import { Button } from '../../shared/Button';
import { ActionButton } from '../../ui/ActionButton';
import { WalkProfileCaptureForm } from './WalkProfileCaptureForm';

interface Props {
  mode: 'import' | 'capture';
  setMode: (mode: 'import' | 'capture') => void;
  file: File | null;
  setFile: (file: File | null) => void;
  fileInput: MutableRefObject<HTMLInputElement | null>;
  capture: WalkCaptureCredentials;
  setCapture: (capture: WalkCaptureCredentials) => void;
  captureName: string;
  setCaptureName: (name: string) => void;
  busy: 'idle' | 'reading' | 'capturing' | 'creating';
  resetResult: () => void;
  runImport: () => Promise<void>;
  runCapture: () => Promise<void>;
  cancel: () => void;
}

export function WalkProfileSource(props: Props) {
  const { t } = useTranslation('pages');
  const chooseFile = (event: ChangeEvent<HTMLInputElement>) => {
    props.setFile(event.target.files?.[0] ?? null);
    props.resetResult();
  };

  return (
    <>
      <div className="flex gap-default">
        {(['import', 'capture'] as const).map((mode) => (
          <Button
            key={mode}
            type="button"
            // A segmented control, not an action: the selected segment is
            // filled surface, not the page's primary. It used to render
            // solid, which made two primaries on this page.
            variant={props.mode === mode ? 'secondary' : 'ghost'}
            aria-pressed={props.mode === mode}
            size="sm"
            onClick={() => props.setMode(mode)}
          >
            {t(`walkAnalyzer.profile.${mode}Tab`)}
          </Button>
        ))}
      </div>
      {props.mode === 'import' ? (
        <div className="stack">
          <label
            className="block text-sm font-medium text-text-secondary"
            htmlFor="walk-profile-file"
          >
            {t('walkAnalyzer.profile.walkFile')}
          </label>
          <input
            id="walk-profile-file"
            data-testid="walk-profile-file"
            ref={props.fileInput}
            type="file"
            accept=".walk,.snmpwalk,.txt,text/plain"
            onChange={chooseFile}
            className="block w-full rounded-lg border border-surface-border bg-bg-base/60 px-4 py-row text-sm text-text-primary"
          />
          <div className="flex gap-default">
            <ActionButton
              action="upload"
              type="button"
              disabled={!props.file || props.busy !== 'idle'}
              loading={props.busy === 'reading'}
              onClick={() => void props.runImport()}
              data-testid="walk-profile-import"
            >
              {t('walkAnalyzer.profile.importAction')}
            </ActionButton>
            {props.busy === 'reading' && (
              <Button variant="secondary" type="button" onClick={props.cancel}>
                {t('walkAnalyzer.profile.cancel')}
              </Button>
            )}
          </div>
        </div>
      ) : (
        <WalkProfileCaptureForm
          capture={props.capture}
          setCapture={props.setCapture}
          captureName={props.captureName}
          setCaptureName={props.setCaptureName}
          capturing={props.busy === 'capturing'}
          runCapture={props.runCapture}
          cancel={props.cancel}
        />
      )}
    </>
  );
}
