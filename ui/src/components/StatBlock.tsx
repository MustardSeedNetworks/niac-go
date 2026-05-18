import { memo } from 'react';
import { SmallText } from '../ui/Typography';

interface StatBlockProps {
  label: string;
  value: string;
  helper: string;
}

/**
 * Stat block component for displaying key metrics
 */
export const StatBlock = memo(({ label, value, helper }: StatBlockProps) => (
  <div>
    <SmallText className="uppercase tracking-wide text-text-muted">{label}</SmallText>
    <p className="text-3xl font-bold text-text-primary">{value}</p>
    <SmallText className="text-text-secondary">{helper}</SmallText>
  </div>
));

StatBlock.displayName = 'StatBlock';
