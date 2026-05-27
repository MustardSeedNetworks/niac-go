/**
 * OverviewSection Component
 *
 * Pinned NIAC orientation: what the categories mean and which UI pages cover
 * what. Replaces the old "Features" tab as the landing surface.
 */

import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { categories, features } from '../../data/help-content';
import { badge, cn, layout } from '../../styles/theme';
import type { Feature } from './types';

interface OverviewSectionProps {
  searchQuery: string;
}

export function OverviewSection({ searchQuery }: OverviewSectionProps): ReactElement {
  const { t } = useTranslation('help');
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
    <div className="stack-xl">
      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">
          {t('overview.whatNiacDoesTitle')}
        </h3>
        <p className="text-xs text-text-muted leading-relaxed">{t('overview.whatNiacDoesBody')}</p>
      </section>

      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">
          {t('overview.categoriesTitle')}
        </h3>
        {filteredCategories.length === 0 ? (
          <p className="text-sm text-text-muted py-row">{t('overview.categoriesEmpty')}</p>
        ) : (
          <div className="stack-sm">
            {filteredCategories.map((cat) => (
              <div key={cat.id} className="bg-surface-hover rounded-lg pad-sm">
                <div className={layout.flex.between}>
                  <h4 className="label">{cat.name}</h4>
                  <code className="text-xs text-text-muted">
                    {t('overview.categoryItems', { count: cat.items.length })}
                  </code>
                </div>
                <p className="text-xs text-text-muted mt-tight">{cat.summary}</p>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">
          {t('overview.webUiQuickRefTitle')}
        </h3>
        {filteredFeatures.length === 0 ? (
          <p className="text-sm text-text-muted py-row">{t('overview.featuresEmpty')}</p>
        ) : (
          <div className="stack-sm">
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
    <div className="bg-surface-hover rounded-lg pad-sm hover:bg-surface-hover transition-colors">
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          <h4 className="label">{feature.title}</h4>
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
      <p className="text-xs text-text-muted mt-tight">{feature.description}</p>
    </div>
  );
}
