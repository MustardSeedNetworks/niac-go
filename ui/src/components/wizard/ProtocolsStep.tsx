import type { FC } from 'react';
import { SelectedNetworkPreview } from '../../pages/runtime/SelectedNetworkPreview';

interface ProtocolsStepProps {
  name: string;
  content: string;
}

/**
 * Step 3 summarizes identities and configured services from draft content.
 * It cannot accidentally report devices from a different running scenario.
 */
export const ProtocolsStep: FC<ProtocolsStepProps> = ({ name, content }) => (
  <SelectedNetworkPreview source="upload" name={name} content={content} view="protocols" />
);
