import { type FC } from 'react';
import { H2, P } from '@krisarmstrong/web-foundation';
import { ErrorInjectionPanel } from '../components/ErrorInjectionPanel';
import { ReplayControlPanel } from '../components/ReplayControlPanel';

export const TrafficInjectionPage: FC = () => {
  return (
    <div className="space-y-8">
      {/* Error Injection Section */}
      <div className="space-y-4">
        <div>
          <H2>Error Injection</H2>
          <P className="text-gray-400">
            Inject network errors on device interfaces for testing and simulation scenarios.
          </P>
        </div>
        <ErrorInjectionPanel />
      </div>

      {/* PCAP Replay Section */}
      <div className="space-y-4">
        <div>
          <H2>PCAP Replay</H2>
          <P className="text-gray-400">
            Replay captured packet traffic with loop and timing controls for testing.
          </P>
        </div>
        <ReplayControlPanel />
      </div>
    </div>
  );
};
