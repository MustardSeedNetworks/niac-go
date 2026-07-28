import type { FC } from 'react';
import { SelectedNetworkPreview } from '../../pages/runtime/SelectedNetworkPreview';

interface ReviewStepProps {
  name: string;
  content: string;
}

/**
 * Step 4 reviews the exact revisioned draft that will be sent to preflight.
 */
export const ReviewStep: FC<ReviewStepProps> = ({ name, content }) => (
  <SelectedNetworkPreview source="upload" name={name} content={content} />
);
