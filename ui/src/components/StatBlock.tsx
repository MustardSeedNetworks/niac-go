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
  // Values are not always short numbers: the running-simulation card puts a
  // config filename here and its full path in the helper, neither of which has
  // a space to wrap on. Without break-words they set the card's width and push
  // the whole page into horizontal scroll on a phone (#1483).
  <div className="min-w-0">
    <SmallText className="uppercase tracking-wide text-text-muted">{label}</SmallText>
    <p className="text-3xl font-bold text-text-primary break-words">{value}</p>
    <SmallText className="text-text-secondary break-words">{helper}</SmallText>
  </div>
));

StatBlock.displayName = 'StatBlock';
