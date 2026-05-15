import { AlertCircle, Plus, Search, Server } from 'lucide-react';
import type { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { DeviceCardGridSkeleton, DeviceTableSkeleton } from '../../ui/Skeleton';
import { H2, P, SmallText } from '../../ui/Typography';

interface LoadingStateProps {
  viewMode: 'cards' | 'table';
}

export const DeviceListLoadingState: FC<LoadingStateProps> = ({ viewMode }) => {
  if (viewMode === 'table') {
    return (
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="p-0">
          {/* Table header skeleton */}
          <div className="flex items-center gap-4 border-b border-white/10 px-4 py-3 bg-gray-950/40">
            <div className="h-4 w-4 rounded bg-gray-700/50" />
            <div className="flex-1 grid grid-cols-12 gap-4 text-sm font-medium text-gray-400">
              <div className="col-span-3">Hostname</div>
              <div className="col-span-2">Type</div>
              <div className="col-span-2">IP Address</div>
              <div className="col-span-3">Protocols</div>
              <div className="col-span-2 text-right">Actions</div>
            </div>
          </div>
          <DeviceTableSkeleton rows={8} />
        </CardContent>
      </Card>
    );
  }

  return <DeviceCardGridSkeleton count={8} />;
};

interface ErrorStateProps {
  error: Error;
  onRetry: () => void;
}

export const DeviceListErrorState: FC<ErrorStateProps> = ({ error, onRetry }) => {
  return (
    <Card className="border-red-500/30 bg-red-900/20" role="alert" aria-live="assertive">
      <CardContent className="space-y-3">
        <div className="flex items-start gap-3">
          <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
          <div>
            <p className="font-semibold text-red-200">Failed to Load Devices</p>
            <SmallText className="text-red-300/90">{error.message}</SmallText>
            <Button variant="outline" size="sm" className="mt-3" onClick={onRetry}>
              Retry
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export const DeviceListEmptyState: FC = () => {
  const navigate = useNavigate();

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="py-12 text-center">
        <Server className="mx-auto h-12 w-12 text-gray-600" />
        <H2 className="mt-4 mb-2">No Devices Configured</H2>
        <P className="text-gray-400">
          Add your first device to start configuring your network simulation.
        </P>
        <Button
          tone="violet"
          className="mt-4"
          leftIcon={<Plus className={iconSizes.md} />}
          onClick={() => navigate('/device-config/new')}
        >
          Add Device
        </Button>
      </CardContent>
    </Card>
  );
};

interface NoResultsStateProps {
  onClearFilters: () => void;
}

export const DeviceListNoResultsState: FC<NoResultsStateProps> = ({ onClearFilters }) => {
  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="py-12 text-center">
        <Search className="mx-auto h-12 w-12 text-gray-600" />
        <H2 className="mt-4 mb-2">No Matching Devices</H2>
        <P className="text-gray-400">
          No devices match your current filters. Try adjusting your search.
        </P>
        <Button variant="outline" className="mt-4" onClick={onClearFilters}>
          Clear Filters
        </Button>
      </CardContent>
    </Card>
  );
};
