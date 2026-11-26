import { AlertCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { SmallText } from '../../ui/Typography';

export interface DeviceEditorStatusViewProps {
  isNewDevice: boolean;
  loading: boolean;
  error: Error | null;
  onRetry: () => void;
  onNavigateBack: () => void;
}

export function DeviceEditorStatusView({
  isNewDevice,
  loading,
  error,
  onRetry,
  onNavigateBack,
}: DeviceEditorStatusViewProps) {
  const { t } = useTranslation('devices');
  if (!isNewDevice && loading) {
    return (
      <div className="stack-xl">
        <Card>
          <div className="flex-center">
            <div className="flex items-center gap-default text-text-muted">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-brand-primary border-t-transparent" />
              <span>{t('editor.statusView.loadingDevice')}</span>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!isNewDevice && error) {
    return (
      <div className="stack-xl">
        <Card>
          <div className="stack">
            <div className="flex items-start gap-default">
              <AlertCircle className="mt-tight h-5 w-5 text-status-error" />
              <div>
                <p className="font-semibold text-status-error-strong">
                  {t('editor.statusView.failedTitle')}
                </p>
                <SmallText className="text-status-error-strong">{error.message}</SmallText>
                <div className="flex gap-compact mt-heading">
                  <Button variant="secondary" size="sm" onClick={onRetry}>
                    {t('list.states.retry')}
                  </Button>
                  <Button variant="secondary" size="sm" onClick={onNavigateBack}>
                    {t('editor.statusView.backToList')}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return null;
}
