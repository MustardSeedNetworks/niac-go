import { type FC, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { type ApiErrorDetail, isApiError } from '../../api/errors';
import {
  captureWalkProfile,
  createCapturedProfile,
  importWalkProfile,
  type WalkCaptureCredentials,
  type WalkProfileReview,
} from '../../api/walk-profile-client';
import { ApiErrorMessage } from '../../ui/ApiErrorMessage';
import { Card, CardContent } from '../../ui/Card';
import { WalkProfileReviewForm } from './WalkProfileReviewForm';
import { WalkProfileSource } from './WalkProfileSource';

const maximumWalkBytes = 16 * 1024 * 1024;

const emptyCapture = (): WalkCaptureCredentials => ({
  target: '',
  port: 161,
  version: '2c',
  community: '',
  username: '',
  authProtocol: 'sha256',
  authPassword: '',
  privProtocol: 'aes',
  privPassword: '',
  timeoutSeconds: 20,
});

const walkFilename = (name: string) => {
  const stem = name.replace(/\.(?:snmpwalk|txt|walk)$/i, '');
  return `${stem}.walk`;
};

export const WalkProfileCreator: FC = () => {
  const { t } = useTranslation('pages');
  const [mode, setMode] = useState<'import' | 'capture'>('import');
  const [file, setFile] = useState<File | null>(null);
  const [captureName, setCaptureName] = useState('captured-switch.walk');
  const [capture, setCapture] = useState<WalkCaptureCredentials>(emptyCapture);
  const [review, setReview] = useState<WalkProfileReview | null>(null);
  const [busy, setBusy] = useState<'idle' | 'reading' | 'capturing' | 'creating'>('idle');
  const [error, setError] = useState<string | null>(null);
  // The server enumerates why a capture failed (#1488); keeping only
  // err.message discarded it at the last step (#1499).
  const [errorDetails, setErrorDetails] = useState<readonly ApiErrorDetail[]>([]);
  const [createdRole, setCreatedRole] = useState<string | null>(null);
  const controller = useRef<AbortController | null>(null);
  const fileInput = useRef<HTMLInputElement | null>(null);

  const resetResult = () => {
    setReview(null);
    setCreatedRole(null);
    setError(null);
    setErrorDetails([]);
  };

  const runImport = async () => {
    if (!file) return;
    if (file.size > maximumWalkBytes) {
      setError(t('walkAnalyzer.profile.fileTooLarge'));
      return;
    }
    setBusy('reading');
    setError(null);
    setErrorDetails([]);
    setCreatedRole(null);
    controller.current = new AbortController();
    try {
      setReview(
        await importWalkProfile(
          walkFilename(file.name),
          await file.text(),
          controller.current.signal,
        ),
      );
    } catch (caught) {
      if ((caught as Error).name !== 'AbortError') {
        setError((caught as Error).message);
        setErrorDetails(isApiError(caught) ? caught.details : []);
      }
    } finally {
      controller.current = null;
      setFile(null);
      if (fileInput.current) fileInput.current.value = '';
      setBusy('idle');
    }
  };

  const runCapture = async () => {
    setBusy('capturing');
    setError(null);
    setErrorDetails([]);
    setCreatedRole(null);
    controller.current = new AbortController();
    try {
      setReview(
        await captureWalkProfile(walkFilename(captureName), capture, controller.current.signal),
      );
    } catch (caught) {
      if ((caught as Error).name !== 'AbortError') {
        setError((caught as Error).message);
        setErrorDetails(isApiError(caught) ? caught.details : []);
      }
    } finally {
      controller.current = null;
      setCapture((current) => ({
        ...current,
        community: '',
        username: '',
        authPassword: '',
        privPassword: '',
      }));
      setBusy('idle');
    }
  };

  const createProfile = async () => {
    if (!review) return;
    setBusy('creating');
    setError(null);
    setErrorDetails([]);
    try {
      const created = await createCapturedProfile({
        role: review.profile.role,
        deviceType: review.profile.deviceType,
        vendor: review.profile.vendor,
        model: review.profile.model,
        platform: review.profile.platform,
        software: review.profile.software,
        walkName: review.walkName,
      });
      setCreatedRole(created.role);
      setReview(null);
    } catch (caught) {
      setError((caught as Error).message);
    } finally {
      setBusy('idle');
    }
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70" data-testid="walk-profile-creator">
      <CardContent className="stack-lg">
        <header>
          <h2 className="heading-3 text-text-primary">{t('walkAnalyzer.profile.title')}</h2>
          <p className="text-sm text-text-muted">{t('walkAnalyzer.profile.description')}</p>
        </header>
        <WalkProfileSource
          mode={mode}
          setMode={setMode}
          file={file}
          setFile={setFile}
          fileInput={fileInput}
          capture={capture}
          setCapture={setCapture}
          captureName={captureName}
          setCaptureName={setCaptureName}
          busy={busy}
          resetResult={resetResult}
          runImport={runImport}
          runCapture={runCapture}
          cancel={() => controller.current?.abort()}
        />
        {error && <ApiErrorMessage message={error} details={errorDetails} />}
        {createdRole && (
          <p role="status" className="text-sm text-status-success">
            {t('walkAnalyzer.profile.created', { role: createdRole })}
          </p>
        )}
        {review && (
          <WalkProfileReviewForm
            review={review}
            setReview={setReview}
            creating={busy === 'creating'}
            createProfile={createProfile}
          />
        )}
      </CardContent>
    </Card>
  );
};
