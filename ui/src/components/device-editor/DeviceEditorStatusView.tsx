import { AlertCircle } from 'lucide-react';
import type { FC } from 'react';
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
  if (!isNewDevice && loading) {
    return (
      <div className="space-y-6">
        <Card className="border-white/5 bg-bg-surface/70">
          <CardContent className="flex items-center justify-center py-12">
            <div className="flex items-center gap-3 text-text-muted">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-brand-500 border-t-transparent" />
              <span>Loading device...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isNewDevice && error) {
    return (
      <div className="space-y-6">
        <Card className="border-status-error/30 bg-status-error/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-status-error" />
              <div>
                <p className="font-semibold text-status-error">Failed to Load Device</p>
                <SmallText className="text-status-error/90">{error.message}</SmallText>
                <div className="flex gap-2 mt-3">
                  <Button variant="outline" size="sm" onClick={onRetry}>
                    Retry
                  </Button>
                  <Button variant="outline" size="sm" onClick={onNavigateBack}>
                    Back to List
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
