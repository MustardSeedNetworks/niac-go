import { useTranslation } from 'react-i18next';
import type { WalkCaptureCredentials } from '../../api/walk-profile-client';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Select } from '../../shared/Select';
import { ActionButton } from '../../ui/ActionButton';

interface Props {
  capture: WalkCaptureCredentials;
  setCapture: (capture: WalkCaptureCredentials) => void;
  captureName: string;
  setCaptureName: (name: string) => void;
  capturing: boolean;
  runCapture: () => Promise<void>;
  cancel: () => void;
}

export function WalkProfileCaptureForm({
  capture,
  setCapture,
  captureName,
  setCaptureName,
  capturing,
  runCapture,
  cancel,
}: Props) {
  const { t } = useTranslation('pages');
  const update = (field: keyof WalkCaptureCredentials, value: string | number) =>
    setCapture({ ...capture, [field]: value });

  return (
    <div className="stack-lg" data-testid="walk-profile-capture-form">
      <div className="grid gap-default md:grid-cols-3">
        <FormField label={t('walkAnalyzer.profile.captureName')}>
          {(control) => (
            <Input
              {...control}
              value={captureName}
              onChange={(event) => setCaptureName(event.target.value)}
            />
          )}
        </FormField>
        <FormField label={t('walkAnalyzer.profile.target')}>
          {(control) => (
            <Input
              {...control}
              placeholder="192.0.2.10"
              value={capture.target}
              onChange={(event) => update('target', event.target.value)}
            />
          )}
        </FormField>
        <FormField label={t('walkAnalyzer.profile.port')}>
          {(control) => (
            <Input
              {...control}
              type="number"
              min={1}
              max={65535}
              value={capture.port}
              onChange={(event) => update('port', Number(event.target.value))}
            />
          )}
        </FormField>
      </div>
      <FormField label={t('walkAnalyzer.profile.version')}>
        {(control) => (
          <Select
            {...control}
            value={capture.version}
            options={[
              { value: '2c', label: 'SNMPv2c' },
              { value: '3', label: 'SNMPv3' },
            ]}
            onValueChange={(value) => update('version', value)}
          />
        )}
      </FormField>
      {capture.version === '2c' ? (
        <FormField label={t('walkAnalyzer.profile.community')}>
          {(control) => (
            <Input
              {...control}
              type="password"
              autoComplete="off"
              value={capture.community}
              onChange={(event) => update('community', event.target.value)}
              data-testid="walk-profile-community"
            />
          )}
        </FormField>
      ) : (
        <div className="grid gap-default md:grid-cols-2">
          <FormField label={t('walkAnalyzer.profile.username')}>
            {(control) => (
              <Input
                {...control}
                autoComplete="off"
                value={capture.username}
                onChange={(event) => update('username', event.target.value)}
                data-testid="walk-profile-username"
              />
            )}
          </FormField>
          <FormField label={t('walkAnalyzer.profile.authProtocol')}>
            {(control) => (
              <Select
                {...control}
                value={capture.authProtocol}
                options={['none', 'sha256', 'sha', 'sha512', 'md5'].map((value) => ({
                  value,
                  label: value,
                }))}
                onValueChange={(value) => update('authProtocol', value)}
              />
            )}
          </FormField>
          <FormField label={t('walkAnalyzer.profile.authPassword')}>
            {(control) => (
              <Input
                {...control}
                type="password"
                autoComplete="off"
                value={capture.authPassword}
                onChange={(event) => update('authPassword', event.target.value)}
              />
            )}
          </FormField>
          <FormField label={t('walkAnalyzer.profile.privProtocol')}>
            {(control) => (
              <Select
                {...control}
                value={capture.privProtocol}
                options={['none', 'aes', 'aes192', 'aes256', 'des'].map((value) => ({
                  value,
                  label: value,
                }))}
                onValueChange={(value) => update('privProtocol', value)}
              />
            )}
          </FormField>
          <FormField label={t('walkAnalyzer.profile.privPassword')}>
            {(control) => (
              <Input
                {...control}
                type="password"
                autoComplete="off"
                value={capture.privPassword}
                onChange={(event) => update('privPassword', event.target.value)}
              />
            )}
          </FormField>
        </div>
      )}
      <p className="text-xs text-text-muted">{t('walkAnalyzer.profile.credentialsNotice')}</p>
      <div className="flex gap-default">
        <ActionButton
          variant="primary"
          action="start"
          type="button"
          disabled={!capture.target || capturing}
          loading={capturing}
          onClick={() => void runCapture()}
          data-testid="walk-profile-capture"
        >
          {t('walkAnalyzer.profile.captureAction')}
        </ActionButton>
        {capturing && (
          <Button variant="secondary" type="button" onClick={cancel}>
            {t('walkAnalyzer.profile.cancel')}
          </Button>
        )}
      </div>
    </div>
  );
}
