import { Check, Eye, FileCode, FolderOpen, HardDrive, Star } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import {
  type ConfigItem,
  TEMPLATE_TYPE_ICON,
  TEMPLATE_TYPE_TINT,
  type ViewMode,
} from './ConfigPicker.types';

interface SharedItemProps {
  item: ConfigItem;
  selected: boolean;
  favorited: boolean;
  onSelect: (item: ConfigItem) => void;
  onToggleFavorite: (key: string) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}

export interface ConfigSections {
  local: ConfigItem[];
  favorites: ConfigItem[];
  all: ConfigItem[];
}

/**
 * ConfigsList renders the network list as up to three zones:
 *   1. Local upload (only when the user has picked a file on this page)
 *   2. Favorites   (only when there are starred entries — hidden during search)
 *   3. All         (everything else, or all matches when searching)
 *
 * Each zone gets a small heading with a count so the list stays scannable
 * even when there are dozens of saved networks.
 */
export const ConfigsList: FC<{
  sections: ConfigSections;
  loading: boolean;
  viewMode: ViewMode;
  isSelected: (item: ConfigItem) => boolean;
  isFavorite: (key: string) => boolean;
  onSelect: (item: ConfigItem) => void;
  onToggleFavorite: (key: string) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
  searching: boolean;
}> = ({
  sections,
  loading,
  viewMode,
  isSelected,
  isFavorite,
  onSelect,
  onToggleFavorite,
  onView,
  onClearLocal,
  searching,
}) => {
  const { t } = useTranslation('pages');

  if (loading) {
    return <SmallText className="text-text-muted">{t('configPicker.loadingNetworks')}</SmallText>;
  }

  const total = sections.local.length + sections.favorites.length + sections.all.length;
  if (total === 0) {
    return <SmallText className="text-text-muted">{t('configPicker.noMatches')}</SmallText>;
  }

  const renderSection = (label: string, items: ConfigItem[]) => {
    if (items.length === 0) return null;
    return (
      <section className="stack-sm" key={label}>
        <header className="flex items-baseline gap-compact">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted">
            {label}
          </h3>
          <span className="text-xs text-text-muted">· {items.length}</span>
        </header>
        {viewMode === 'grid' ? (
          <div className="grid grid-cols-1 gap-default sm:grid-cols-2 xl:grid-cols-3">
            {items.map((item) => (
              <ConfigCard
                key={item.key}
                item={item}
                selected={isSelected(item)}
                favorited={isFavorite(item.key)}
                onSelect={onSelect}
                onToggleFavorite={onToggleFavorite}
                onView={onView}
                onClearLocal={onClearLocal}
              />
            ))}
          </div>
        ) : (
          <ul className="divide-y divide-knob/5 overflow-hidden rounded-lg border border-surface-border bg-bg-base/40">
            {items.map((item) => (
              <ConfigRow
                key={item.key}
                item={item}
                selected={isSelected(item)}
                favorited={isFavorite(item.key)}
                onSelect={onSelect}
                onToggleFavorite={onToggleFavorite}
                onView={onView}
                onClearLocal={onClearLocal}
              />
            ))}
          </ul>
        )}
      </section>
    );
  };

  return (
    <div className="max-h-[480px] stack-lg overflow-y-auto pr-1">
      {renderSection(t('configPicker.localUploadSection'), sections.local)}
      {!searching && renderSection(t('configPicker.favoritesSection'), sections.favorites)}
      {renderSection(
        searching ? t('configPicker.resultsSection') : t('configPicker.allNetworksSection'),
        sections.all,
      )}
    </div>
  );
};

const FavoriteStar: FC<{
  itemKey: string;
  favorited: boolean;
  onToggle: (key: string) => void;
  compact?: boolean;
}> = ({ itemKey, favorited, onToggle, compact }) => {
  const { t } = useTranslation('pages');
  const label = favorited
    ? t('configPicker.removeFromFavorites')
    : t('configPicker.addToFavorites');
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onToggle(itemKey);
      }}
      aria-pressed={favorited}
      aria-label={label}
      title={label}
      className={`rounded p-1 transition-colors ${
        favorited
          ? 'text-status-warning hover:text-status-warning'
          : 'text-text-muted hover:text-status-warning'
      } ${compact ? '' : 'hover:bg-surface-hover'}`}
    >
      <Star
        className={compact ? iconSizes.sm : iconSizes.md}
        fill={favorited ? 'currentColor' : 'none'}
      />
    </button>
  );
};

const ConfigCard: FC<SharedItemProps> = ({
  item,
  selected,
  favorited,
  onSelect,
  onToggleFavorite,
  onView,
  onClearLocal,
}) => {
  const { t } = useTranslation('pages');
  const Icon =
    item.kind === 'builtin'
      ? (TEMPLATE_TYPE_ICON[item.template.type] ?? FileCode)
      : item.kind === 'saved'
        ? FolderOpen
        : HardDrive;
  const tint =
    item.kind === 'builtin'
      ? (TEMPLATE_TYPE_TINT[item.template.type] ?? TEMPLATE_TYPE_TINT.custom)
      : item.kind === 'saved'
        ? 'bg-status-success/15 text-status-success border-status-success/30'
        : 'bg-status-info/15 text-status-info border-status-info/30';

  return (
    <div
      className={`flex flex-col gap-default rounded-lg border pad-sm transition-colors ${
        selected
          ? 'border-brand-accent/50 bg-brand-primary/10'
          : 'border-surface-border bg-bg-base/40 hover:border-brand-primary/30'
      }`}
    >
      <div className="flex items-start justify-between gap-compact">
        <div className={`rounded-md border pad-xs ${tint}`}>
          <Icon className={iconSizes.lg} />
        </div>
        {item.kind !== 'local' && (
          <FavoriteStar itemKey={item.key} favorited={favorited} onToggle={onToggleFavorite} />
        )}
      </div>
      <div>
        <div className="font-semibold text-text-primary">{item.name}</div>
        <SmallText className="mt-0.5 line-clamp-2 text-text-muted">
          {item.description || t('configPicker.noDescription')}
        </SmallText>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {t('configPicker.deviceCount', { count: item.deviceCount })}
          </Tag>
        )}
        {item.kind === 'builtin' &&
          item.template.tags?.slice(0, 2).map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-[10px]">
              {tag}
            </Tag>
          ))}
      </div>
      <div className="flex gap-compact">
        {selected ? (
          <div className="flex flex-1 items-center justify-center gap-1.5 rounded bg-brand-primary/30 px-cell py-compact-md text-xs font-medium text-brand-accent ring-1 ring-brand-accent/60">
            <Check className={iconSizes.sm} />
            <span>{t('configPicker.selectedLabel')}</span>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => onSelect(item)}
            className="flex-1 rounded bg-brand-primary/20 px-cell py-compact-md text-xs font-medium text-brand-accent ring-1 ring-brand-accent/40 hover:bg-brand-primary/30"
            title={t('configPicker.selectNetworkTitle')}
          >
            {t('configPicker.selectButton')}
          </button>
        )}
        {item.kind === 'builtin' && (
          <button
            type="button"
            onClick={() => onView(item)}
            className="rounded border border-surface-border bg-bg-surface/60 px-cell py-compact-md text-xs font-medium text-text-primary hover:bg-surface-hover"
            title={t('configPicker.previewYamlTitle')}
          >
            <Eye className={iconSizes.sm} />
          </button>
        )}
        {item.kind === 'local' && (
          <button
            type="button"
            onClick={onClearLocal}
            className="rounded border border-status-error/30 bg-status-error/10 px-cell py-compact-md text-xs font-medium text-status-error hover:bg-status-error/20"
            title={t('configPicker.dropLocalFileTitle')}
          >
            {t('configPicker.clearButton')}
          </button>
        )}
      </div>
    </div>
  );
};

const ConfigRow: FC<SharedItemProps> = ({
  item,
  selected,
  favorited,
  onSelect,
  onToggleFavorite,
  onView,
  onClearLocal,
}) => {
  const { t } = useTranslation('pages');
  return (
    <li
      className={`flex items-center gap-default px-3 py-row transition-colors ${
        selected ? 'bg-brand-primary/10' : 'hover:bg-surface-hover'
      }`}
    >
      {item.kind !== 'local' && (
        <FavoriteStar
          itemKey={item.key}
          favorited={favorited}
          onToggle={onToggleFavorite}
          compact
        />
      )}
      <button
        type="button"
        onClick={() => onSelect(item)}
        className="flex-1 text-left"
        title={t('configPicker.selectItemTitle', { name: item.name })}
      >
        <div className="flex items-center gap-compact">
          <span className="font-medium text-text-primary">{item.name}</span>
          {item.kind !== 'local' && (
            <Tag colorScheme="purple" className="text-[10px]">
              {t('configPicker.deviceCount', { count: item.deviceCount })}
            </Tag>
          )}
          {item.kind === 'local' && (
            <Tag colorScheme="blue" className="text-[10px]">
              {t('configPicker.localTag')}
            </Tag>
          )}
        </div>
        {item.description && (
          <SmallText
            className={`mt-0.5 line-clamp-1 text-text-muted ${
              item.kind === 'saved' ? 'font-mono text-[11px] text-text-muted' : ''
            }`}
          >
            {item.description}
          </SmallText>
        )}
      </button>
      {item.kind === 'builtin' && (
        <button
          type="button"
          onClick={() => onView(item)}
          className="rounded p-1.5 text-text-muted hover:bg-surface-hover hover:text-text-primary"
          title={t('configPicker.previewTemplateYamlTitle')}
        >
          <Eye className={iconSizes.md} />
        </button>
      )}
      {item.kind === 'local' && (
        <button
          type="button"
          onClick={onClearLocal}
          className="text-xs font-medium text-status-error hover:text-status-error"
          title={t('configPicker.dropLocalFileTitle')}
        >
          {t('configPicker.clearButton')}
        </button>
      )}
    </li>
  );
};
