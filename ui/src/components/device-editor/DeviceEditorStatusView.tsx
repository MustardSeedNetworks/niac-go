import { AlertCircle } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';

export interface DeviceEditorStatusViewProps {
  isNewDevice: boolean;
  loading: boolean;
  error: Error | null;
  onRetry: () => void;
  onNavigateBack: () => void;
}

export const DeviceEditorStatusView: FC<DeviceEditorStatusViewProps> = ({
  isNewDevice,
  loading,
  error,
  onRetry,
  onNavigateBack,
}) => {
  const { t } = useTranslation('devices');
  if (!isNewDevice && loading) {
    return (
      <div className="stack-xl">
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="flex-center py-centered">
            <div className="flex items-center gap-default text-text-muted">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-brand-primary border-t-transparent" />
              <span>{t('editor.statusView.loadingDevice')}</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isNewDevice && error) {
    return (
      <div className="stack-xl">
        <Card className="border-status-error/30 bg-status-error/20">
          <CardContent className="stack">
            <div className="flex items-start gap-default">
              <AlertCircle className="mt-tight h-5 w-5 text-status-error" />
              <div>
                <p className="font-semibold text-status-error">
                  {t('editor.statusView.failedTitle')}
                </p>
                <SmallText className="text-status-error/90">{error.message}</SmallText>
                <div className="flex gap-compact mt-heading">
                  <Button variant="outline" size="sm" onClick={onRetry}>
                    {t('list.states.retry')}
                  </Button>
                  <Button variant="outline" size="sm" onClick={onNavigateBack}>
                    {t('editor.statusView.backToList')}
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return null;
};
