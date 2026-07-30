import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { WalkCaptureCredentials } from '../../api/walk-profile-client';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';

interface Props {
  capture: WalkCaptureCredentials;
  setCapture: (capture: WalkCaptureCredentials) => void;
  captureName: string;
  setCaptureName: (name: string) => void;
  capturing: boolean;
  runCapture: () => Promise<void>;
  cancel: () => void;
}

export const WalkProfileCaptureForm: FC<Props> = ({
  capture,
  setCapture,
  captureName,
  setCaptureName,
  capturing,
  runCapture,
  cancel,
}) => {
  const { t } = useTranslation('pages');
  const update = (field: keyof WalkCaptureCredentials, value: string | number) =>
    setCapture({ ...capture, [field]: value });

  return (
    <div className="stack-lg" data-testid="walk-profile-capture-form">
      <div className="grid gap-default md:grid-cols-3">
        <Input
          label={t('walkAnalyzer.profile.captureName')}
          value={captureName}
          onChange={(event) => setCaptureName(event.target.value)}
        />
        <Input
          label={t('walkAnalyzer.profile.target')}
          placeholder="192.0.2.10"
          value={capture.target}
          onChange={(event) => update('target', event.target.value)}
        />
        <Input
          label={t('walkAnalyzer.profile.port')}
          type="number"
          min={1}
          max={65535}
          value={capture.port}
          onChange={(event) => update('port', Number(event.target.value))}
        />
      </div>
      <Select
        label={t('walkAnalyzer.profile.version')}
        value={capture.version}
        options={[
          { value: '2c', label: 'SNMPv2c' },
          { value: '3', label: 'SNMPv3' },
        ]}
        onChange={(value) => update('version', value)}
      />
      {capture.version === '2c' ? (
        <Input
          label={t('walkAnalyzer.profile.community')}
          type="password"
          autoComplete="off"
          value={capture.community}
          onChange={(event) => update('community', event.target.value)}
          data-testid="walk-profile-community"
        />
      ) : (
        <div className="grid gap-default md:grid-cols-2">
          <Input
            label={t('walkAnalyzer.profile.username')}
            autoComplete="off"
            value={capture.username}
            onChange={(event) => update('username', event.target.value)}
            data-testid="walk-profile-username"
          />
          <Select
            label={t('walkAnalyzer.profile.authProtocol')}
            value={capture.authProtocol}
            options={['none', 'sha256', 'sha', 'sha512', 'md5'].map((value) => ({
              value,
              label: value,
            }))}
            onChange={(value) => update('authProtocol', value)}
          />
          <Input
            label={t('walkAnalyzer.profile.authPassword')}
            type="password"
            autoComplete="off"
            value={capture.authPassword}
            onChange={(event) => update('authPassword', event.target.value)}
          />
          <Select
            label={t('walkAnalyzer.profile.privProtocol')}
            value={capture.privProtocol}
            options={['none', 'aes', 'aes192', 'aes256', 'des'].map((value) => ({
              value,
              label: value,
            }))}
            onChange={(value) => update('privProtocol', value)}
          />
          <Input
            label={t('walkAnalyzer.profile.privPassword')}
            type="password"
            autoComplete="off"
            value={capture.privPassword}
            onChange={(event) => update('privPassword', event.target.value)}
          />
        </div>
      )}
      <p className="text-xs text-text-muted">{t('walkAnalyzer.profile.credentialsNotice')}</p>
      <div className="flex gap-default">
        <Button
          type="button"
          tone="blue"
          disabled={!capture.target || capturing}
          loading={capturing}
          onClick={() => void runCapture()}
          data-testid="walk-profile-capture"
        >
          {t('walkAnalyzer.profile.captureAction')}
        </Button>
        {capturing && (
          <Button type="button" variant="outline" tone="gray" onClick={cancel}>
            {t('walkAnalyzer.profile.cancel')}
          </Button>
        )}
      </div>
    </div>
  );
};
