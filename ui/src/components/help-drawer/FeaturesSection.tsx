// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * FeaturesSection Component
 *
 * Displays feature quick reference cards with search filtering.
 */

import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { badge, cn, layout } from '../../styles/theme';
import { FEATURES } from './data';
import type { Feature } from './types';

interface FeaturesSectionProps {
  searchQuery: string;
}

export function FeaturesSection({ searchQuery }: FeaturesSectionProps): ReactElement {
  const filteredFeatures = useMemo(() => {
    if (!searchQuery.trim()) return FEATURES;
    const query = searchQuery.toLowerCase();
    return FEATURES.filter(
      (f) => f.title.toLowerCase().includes(query) || f.description.toLowerCase().includes(query),
    );
  }, [searchQuery]);

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-white">Quick Reference</h3>
      {filteredFeatures.length === 0 ? (
        <p className="text-sm text-gray-500 py-4 text-center">No features match your search.</p>
      ) : (
        <div className="space-y-2">
          {filteredFeatures.map((feature) => (
            <FeatureCard key={feature.path} feature={feature} />
          ))}
        </div>
      )}
    </div>
  );
}

interface FeatureCardProps {
  feature: Feature;
}

function FeatureCard({ feature }: FeatureCardProps): ReactElement {
  return (
    <div className="bg-white/5 rounded-lg p-3 hover:bg-white/10 transition-colors">
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          <h4 className="text-sm font-medium text-white">{feature.title}</h4>
          {feature.badge && (
            <span
              className={cn(
                badge.base,
                feature.badge === 'New' ? badge.variant.new : badge.variant.beta,
                badge.size.xs,
              )}
            >
              {feature.badge}
            </span>
          )}
        </div>
        <code className="text-xs text-gray-500">{feature.path}</code>
      </div>
      <p className="text-xs text-gray-400 mt-1">{feature.description}</p>
    </div>
  );
}
