import { AlertCircle } from 'lucide-react';
import type { FC } from 'react';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';

export interface DeviceStatusViewProps {
  isNewDevice: boolean;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
  navigate: (path: string) => void;
}

/**
 * Displays loading and error states for the device editor
 * Returns null when device data is ready to be edited
 */
export const DeviceStatusView: FC<DeviceStatusViewProps> = ({
  isNewDevice,
  loading,
  error,
  refetch,
  navigate,
}) => {
  if (!isNewDevice && loading) {
    return (
      <div className="space-y-6">
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="flex items-center justify-center py-12">
            <div className="flex items-center gap-3 text-gray-400">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
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
        <Card className="border-red-500/30 bg-red-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
              <div>
                <p className="font-semibold text-red-200">Failed to Load Device</p>
                <SmallText className="text-red-300/90">{error.message}</SmallText>
                <div className="flex gap-2 mt-3">
                  <Button variant="outline" size="sm" onClick={() => refetch()}>
                    Retry
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => navigate('/device-config')}>
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
