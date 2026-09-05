import { AlertTriangle, Network, RefreshCw } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';

/**
 * CanvasState renders what the topology canvas shows when it has no graph to
 * draw: still loading, failed to load, or loaded and genuinely empty.
 *
 * The three were inline branches on one page and the middle one did not exist,
 * so a daemon the UI could not reach fell through to "no topology data" and a
 * failed load was indistinguishable from a network with nothing in it. Keeping
 * them together is what makes that omission visible next time.
 */
export const CanvasState: FC<{
  loading: boolean;
  error: Error | null;
  onRetry: () => void;
}> = ({ loading, error, onRetry }) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');

  if (loading) {
    return (
      <Centered>
        <div className="flex flex-col items-center gap-default">
          <RefreshCw className="w-8 h-8 text-brand-accent animate-spin" />
          <SmallText className="text-text-muted">{t('topology.page.loadingTopology')}</SmallText>
        </div>
      </Centered>
    );
  }

  if (error) {
    return (
      <Centered testId="topology-load-error">
        <div className="text-center">
          <AlertTriangle className="w-16 h-16 text-status-error mx-auto mb-content" />
          <p className="text-status-error mb-2" role="alert">
            {t('topology.page.loadFailed')}
          </p>
          <SmallText className="text-text-muted">{error.message}</SmallText>
          <div className="mt-content">
            <Button variant="secondary" size="sm" onClick={onRetry}>
              {tCommon('buttons.refresh')}
            </Button>
          </div>
        </div>
      </Centered>
    );
  }

  return (
    <Centered>
      <div className="text-center">
        <Network className="w-16 h-16 text-text-disabled mx-auto mb-content" />
        <p className="text-text-muted mb-2">{t('topology.page.noTopologyData')}</p>
        <SmallText className="text-text-muted">{t('topology.page.noConnectionsHint')}</SmallText>
      </div>
    </Centered>
  );
};

const Centered: FC<{ children: React.ReactNode; testId?: string }> = ({ children, testId }) => (
  <div className="absolute inset-0 flex-center" data-testid={testId}>
    {children}
  </div>
);
