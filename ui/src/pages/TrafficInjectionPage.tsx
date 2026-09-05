import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { ErrorInjectionPanel } from '../components/ErrorInjectionPanel';
import { ReplayControlPanel } from '../components/ReplayControlPanel';
import { H2, P } from '../ui/Typography';

export const TrafficInjectionPage: FC = () => {
  const { t } = useTranslation('pages');
  return (
    <div className="space-y-8">
      {/* Error Injection */}
      <div className="stack-lg">
        <div>
          <H2>{t('traffic.page.errorInjectionTitle')}</H2>
          <P className="text-text-muted">{t('traffic.page.errorInjectionDescription')}</P>
        </div>
        <ErrorInjectionPanel />
      </div>

      {/* PCAP Replay */}
      <div className="stack-lg">
        <div>
          <H2>{t('traffic.page.pcapReplayTitle')}</H2>
          <P className="text-text-muted">{t('traffic.page.pcapReplayDescription')}</P>
        </div>
        <ReplayControlPanel />
      </div>
    </div>
  );
};
