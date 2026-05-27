/**
 * HelpItemCard
 *
 * Renders a single HelpItem with collapsible details. Used by Devices,
 * Protocols, Commands tabs.
 */

import { ChevronDown, ChevronRight } from 'lucide-react';
import type { ReactElement } from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, layout } from '../../styles/theme';
import type { HelpItem } from './types';

interface HelpItemCardProps {
  item: HelpItem;
  /** Optional initial expanded state. */
  defaultOpen?: boolean;
}

export function HelpItemCard({ item, defaultOpen = false }: HelpItemCardProps): ReactElement {
  const { t } = useTranslation('help');
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div className="bg-surface-hover rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'w-full text-left px-3 py-2.5 flex items-start gap-2',
          'hover:bg-surface-hover transition-colors',
        )}
        aria-expanded={open}
      >
        {open ? (
          <ChevronDown className="w-4 h-4 mt-0.5 text-text-muted shrink-0" aria-hidden="true" />
        ) : (
          <ChevronRight className="w-4 h-4 mt-0.5 text-text-muted shrink-0" aria-hidden="true" />
        )}
        <div className="flex-1 min-w-0">
          <div className={layout.flex.between}>
            <h4 className="text-sm font-medium text-text-primary">{item.name}</h4>
            <code className="text-xs text-text-muted ml-2 shrink-0">{item.standard}</code>
          </div>
          <p className="text-xs text-text-muted mt-0.5">{item.summary}</p>
        </div>
      </button>

      {open && (
        <div className="px-3 pb-3 pt-1 space-y-3 border-t border-surface-border">
          <section>
            <h5 className="text-xs font-semibold text-text-secondary mb-1">
              {t('itemCard.technical')}
            </h5>
            <p className="text-xs text-text-muted leading-relaxed">{item.techDesc}</p>
          </section>

          <section>
            <h5 className="text-xs font-semibold text-text-secondary mb-1">
              {t('itemCard.plainEnglish')}
            </h5>
            <p className="text-xs text-text-muted leading-relaxed">{item.laymanDesc}</p>
          </section>

          {item.whenToUse && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.whenToUse')}
              </h5>
              <p className="text-xs text-text-muted leading-relaxed">{item.whenToUse}</p>
            </section>
          )}

          {item.whenNotToUse && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.whenNotToUse')}
              </h5>
              <p className="text-xs text-text-muted leading-relaxed">{item.whenNotToUse}</p>
            </section>
          )}

          {item.parameters.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.parameters')}
              </h5>
              <div className="space-y-1">
                {item.parameters.map((p) => (
                  <div key={p.name} className="bg-surface-hover rounded p-2">
                    <div className={layout.flex.between}>
                      <span className="text-xs font-medium text-text-primary">{p.name}</span>
                      <code className="text-xs text-text-muted">{p.flag}</code>
                    </div>
                    <p className="text-xs text-text-muted mt-0.5">{p.laymanDesc}</p>
                    {p.example && (
                      <code className="block mt-1 text-xs text-brand-accent break-all">
                        {p.example}
                      </code>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {item.configFields.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.yamlFields')}
              </h5>
              <div className="space-y-1">
                {item.configFields.map((f) => (
                  <div key={f.path} className="bg-surface-hover rounded p-2">
                    <div className={layout.flex.between}>
                      <code className="text-xs text-text-primary">{f.path}</code>
                      <span className="text-xs text-text-muted">{f.type}</span>
                    </div>
                    <p className="text-xs text-text-muted mt-0.5">{f.description}</p>
                    {f.example && (
                      <pre className="mt-1 text-xs text-brand-accent whitespace-pre-wrap break-all">
                        {f.example}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {item.metrics.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.metrics')}
              </h5>
              <div className="space-y-1">
                {item.metrics.map((m) => (
                  <div key={m.name} className="bg-surface-hover rounded p-2">
                    <div className={layout.flex.between}>
                      <code className="text-xs text-text-primary">{m.name}</code>
                      <span className="text-xs text-text-muted">{m.unit}</span>
                    </div>
                    <p className="text-xs text-text-muted mt-0.5">
                      <span className="text-emerald-400">{t('itemCard.metricGoodLabel')}</span>{' '}
                      {m.goodRange}
                    </p>
                    <p className="text-xs text-text-muted">
                      <span className="text-amber-400">{t('itemCard.metricBadLabel')}</span>{' '}
                      {m.badMeaning}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          )}

          {item.examples.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.examples')}
              </h5>
              <div className="space-y-2">
                {item.examples.map((ex) => (
                  <div key={ex.command} className="bg-surface-hover rounded p-2">
                    <p className="text-xs text-text-muted mb-1">{ex.desc}</p>
                    <pre className="text-xs text-brand-accent whitespace-pre-wrap break-all">
                      {ex.command}
                    </pre>
                    {ex.output && (
                      <pre className="mt-1 text-xs text-text-muted whitespace-pre-wrap break-all border-l-2 border-surface-border pl-2">
                        {ex.output}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {item.tips.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.tips')}
              </h5>
              <ul className="space-y-1 pl-4 list-disc text-xs text-text-muted">
                {item.tips.map((tip) => (
                  <li key={tip}>{tip}</li>
                ))}
              </ul>
            </section>
          )}

          {item.seeAlso.length > 0 && (
            <section>
              <h5 className="text-xs font-semibold text-text-secondary mb-1">
                {t('itemCard.seeAlso')}
              </h5>
              <div className="flex flex-wrap gap-1">
                {item.seeAlso.map((ref) => (
                  <code
                    key={ref}
                    className="text-xs text-text-muted bg-surface-hover rounded px-1.5 py-0.5"
                  >
                    {ref}
                  </code>
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}
