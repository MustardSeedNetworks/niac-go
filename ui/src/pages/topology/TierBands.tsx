/**
 * TierBands draws the Topology archetype's labelled core → access bands.
 *
 * The bands render through ReactFlow's ViewportPortal so they live in the
 * graph's coordinate space — panning and zooming carries them with the
 * devices — without entering node state, where they would have to be filtered
 * out of drag persistence, selection and opacity handling on every pass.
 *
 * They are chrome, not data: `aria-hidden` and pointer-events-none, so a band
 * can never absorb a click meant for a device.
 */

import { ViewportPortal } from '@xyflow/react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { Tier } from './tiers';

/** i18n key per band, so the label is translated rather than drawn from the enum. */
const LABEL_KEYS = {
  Core: 'topology.tiers.core',
  Distribution: 'topology.tiers.distribution',
  Access: 'topology.tiers.access',
} as const satisfies Record<Tier['label'], string>;

interface TierBandProps {
  tier: Tier;
  /** Canvas x of the band's left edge. */
  left: number;
  /** Width the band spans. */
  width: number;
}

/**
 * TierBand is one band. Split out from the portal wrapper so its markup can be
 * asserted directly — ViewportPortal resolves its target from ReactFlow's
 * store, which a unit test has no reason to stand up.
 */
export const TierBand: FC<TierBandProps> = ({ tier, left, width }) => {
  const { t } = useTranslation('pages');

  return (
    <div
      data-testid="topology-tier-band"
      data-tier={tier.label}
      aria-hidden="true"
      className="pointer-events-none absolute rounded-2xl border border-border-muted"
      style={{
        transform: `translate(${left}px, ${tier.y}px)`,
        width,
        height: tier.height,
        backgroundColor: 'var(--color-bg-subtle)',
        zIndex: -1,
      }}
    >
      <span className="absolute left-4 top-3 text-xs uppercase tracking-wide text-fg-muted">
        {t(LABEL_KEYS[tier.label])}
        {' · '}
        <span className="figure">{tier.deviceCount}</span>
      </span>
    </div>
  );
};

interface TierBandsProps {
  tiers: Tier[];
  /** Canvas x of the leftmost device, so bands start where the graph does. */
  left: number;
  /** Width the bands span. */
  width: number;
}

export const TierBands: FC<TierBandsProps> = ({ tiers, left, width }) => {
  if (tiers.length === 0) {
    return null;
  }

  return (
    <ViewportPortal>
      {tiers.map((tier, index) => (
        // Bands are positional, not identified — two Distribution bands are
        // distinguished only by rank, so the index is the honest key.
        <TierBand key={`${tier.label}-${index}`} tier={tier} left={left} width={width} />
      ))}
    </ViewportPortal>
  );
};
