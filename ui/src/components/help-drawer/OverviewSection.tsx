/**
 * OverviewSection Component
 *
 * Pinned NIAC orientation: what the categories mean and which UI pages cover
 * what. Replaces the old "Features" tab as the landing surface.
 */

import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { categories, features } from '../../data/help-content';
import { badge, cn, layout } from '../../styles/theme';
import type { Feature } from './types';

interface OverviewSectionProps {
  searchQuery: string;
}

export function OverviewSection({ searchQuery }: OverviewSectionProps): ReactElement {
  const filteredFeatures = useMemo(() => {
    if (!searchQuery.trim()) return features;
    const query = searchQuery.toLowerCase();
    return features.filter(
      (f) => f.title.toLowerCase().includes(query) || f.description.toLowerCase().includes(query),
    );
  }, [searchQuery]);

  const filteredCategories = useMemo(() => {
    const all = Object.values(categories);
    if (!searchQuery.trim()) return all;
    const query = searchQuery.toLowerCase();
    return all.filter(
      (c) =>
        c.name.toLowerCase().includes(query) ||
        c.summary.toLowerCase().includes(query) ||
        c.fullName.toLowerCase().includes(query),
    );
  }, [searchQuery]);

  return (
    <div className="space-y-6">
      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">What NIAC does</h3>
        <p className="text-xs text-text-muted leading-relaxed">
          NIAC (Network In A Can) simulates one or more network devices on a real host interface. It
          answers ARP, ICMP, LLDP, CDP, SNMP, DHCP, DNS, HTTP, FTP and other protocols so a
          discovery tool, NMS, or monitoring platform sees the simulated devices as if they were
          physical hardware. Use it for tooling validation, training labs, and reproducer
          environments without standing up real gear.
        </p>
      </section>

      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">Help categories</h3>
        {filteredCategories.length === 0 ? (
          <p className="text-sm text-text-muted py-2">No categories match your search.</p>
        ) : (
          <div className="space-y-2">
            {filteredCategories.map((cat) => (
              <div key={cat.id} className="bg-surface-hover rounded-lg p-3">
                <div className={layout.flex.between}>
                  <h4 className="text-sm font-medium text-text-primary">{cat.name}</h4>
                  <code className="text-xs text-text-muted">{cat.items.length} items</code>
                </div>
                <p className="text-xs text-text-muted mt-1">{cat.summary}</p>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">Web UI quick reference</h3>
        {filteredFeatures.length === 0 ? (
          <p className="text-sm text-text-muted py-2">No features match your search.</p>
        ) : (
          <div className="space-y-2">
            {filteredFeatures.map((feature) => (
              <FeatureCard key={feature.path} feature={feature} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

interface FeatureCardProps {
  feature: Feature;
}

function FeatureCard({ feature }: FeatureCardProps): ReactElement {
  return (
    <div className="bg-surface-hover rounded-lg p-3 hover:bg-surface-hover transition-colors">
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          <h4 className="text-sm font-medium text-text-primary">{feature.title}</h4>
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
        <code className="text-xs text-text-muted">{feature.path}</code>
      </div>
      <p className="text-xs text-text-muted mt-1">{feature.description}</p>
    </div>
  );
}
