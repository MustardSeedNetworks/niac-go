import { Check, Eye, FileCode, FolderOpen, HardDrive, Star } from 'lucide-react';
import type { FC } from 'react';
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
  if (loading) {
    return <SmallText className="text-text-muted">Loading networks…</SmallText>;
  }

  const total = sections.local.length + sections.favorites.length + sections.all.length;
  if (total === 0) {
    return (
      <SmallText className="text-text-muted">
        Nothing matches. Try a different search or upload a local YAML with the button above.
      </SmallText>
    );
  }

  const renderSection = (label: string, items: ConfigItem[]) => {
    if (items.length === 0) return null;
    return (
      <section className="space-y-2" key={label}>
        <header className="flex items-baseline gap-2">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-text-muted">
            {label}
          </h3>
          <span className="text-xs text-text-muted">· {items.length}</span>
        </header>
        {viewMode === 'grid' ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
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
          <ul className="divide-y divide-white/5 overflow-hidden rounded-lg border border-white/10 bg-bg-base/40">
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
    <div className="max-h-[480px] space-y-4 overflow-y-auto pr-1">
      {renderSection('Local upload', sections.local)}
      {!searching && renderSection('Favorites', sections.favorites)}
      {renderSection(searching ? 'Results' : 'All networks', sections.all)}
    </div>
  );
};

const FavoriteStar: FC<{
  itemKey: string;
  favorited: boolean;
  onToggle: (key: string) => void;
  compact?: boolean;
}> = ({ itemKey, favorited, onToggle, compact }) => (
  <button
    type="button"
    onClick={(e) => {
      e.stopPropagation();
      onToggle(itemKey);
    }}
    aria-pressed={favorited}
    aria-label={favorited ? 'Remove from favorites' : 'Add to favorites'}
    title={favorited ? 'Remove from favorites' : 'Add to favorites'}
    className={`rounded p-1 transition-colors ${
      favorited
        ? 'text-status-warning hover:text-status-warning'
        : 'text-text-muted hover:text-status-warning'
    } ${compact ? '' : 'hover:bg-white/5'}`}
  >
    <Star
      className={compact ? iconSizes.sm : iconSizes.md}
      fill={favorited ? 'currentColor' : 'none'}
    />
  </button>
);

const ConfigCard: FC<SharedItemProps> = ({
  item,
  selected,
  favorited,
  onSelect,
  onToggleFavorite,
  onView,
  onClearLocal,
}) => {
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
      className={`flex flex-col gap-3 rounded-lg border p-3 transition-colors ${
        selected
          ? 'border-brand-400/50 bg-brand-500/10'
          : 'border-white/10 bg-bg-base/40 hover:border-brand-500/30'
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className={`rounded-md border p-2 ${tint}`}>
          <Icon className={iconSizes.lg} />
        </div>
        {item.kind !== 'local' && (
          <FavoriteStar itemKey={item.key} favorited={favorited} onToggle={onToggleFavorite} />
        )}
      </div>
      <div>
        <div className="font-semibold text-text-primary">{item.name}</div>
        <SmallText className="mt-0.5 line-clamp-2 text-text-muted">
          {item.description || 'No description'}
        </SmallText>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {item.deviceCount} {item.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
        )}
        {item.kind === 'builtin' &&
          item.template.tags?.slice(0, 2).map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-[10px]">
              {tag}
            </Tag>
          ))}
      </div>
      <div className="flex gap-2">
        {selected ? (
          <div className="flex flex-1 items-center justify-center gap-1.5 rounded bg-brand-500/30 px-2 py-1.5 text-xs font-medium text-brand-50 ring-1 ring-brand-400/60">
            <Check className={iconSizes.sm} />
            <span>Selected</span>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => onSelect(item)}
            className="flex-1 rounded bg-brand-500/20 px-2 py-1.5 text-xs font-medium text-brand-100 ring-1 ring-brand-400/40 hover:bg-brand-500/30"
            title="Select this network — click Start Simulation below to run it"
          >
            Select
          </button>
        )}
        {item.kind === 'builtin' && (
          <button
            type="button"
            onClick={() => onView(item)}
            className="rounded border border-white/10 bg-bg-surface/60 px-2 py-1.5 text-xs font-medium text-text-primary hover:bg-white/10"
            title="Preview YAML"
          >
            <Eye className={iconSizes.sm} />
          </button>
        )}
        {item.kind === 'local' && (
          <button
            type="button"
            onClick={onClearLocal}
            className="rounded border border-status-error/30 bg-status-error/10 px-2 py-1.5 text-xs font-medium text-status-error hover:bg-status-error/20"
            title="Drop the local file"
          >
            Clear
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
}) => (
  <li
    className={`flex items-center gap-3 px-3 py-2 transition-colors ${
      selected ? 'bg-brand-500/10' : 'hover:bg-white/5'
    }`}
  >
    {item.kind !== 'local' && (
      <FavoriteStar itemKey={item.key} favorited={favorited} onToggle={onToggleFavorite} compact />
    )}
    <button
      type="button"
      onClick={() => onSelect(item)}
      className="flex-1 text-left"
      title={`Select ${item.name}`}
    >
      <div className="flex items-center gap-2">
        <span className="font-medium text-text-primary">{item.name}</span>
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {item.deviceCount} {item.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
        )}
        {item.kind === 'local' && (
          <Tag colorScheme="blue" className="text-[10px]">
            Local
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
        className="rounded p-1.5 text-text-muted hover:bg-white/10 hover:text-text-primary"
        title="Preview the template YAML"
      >
        <Eye className={iconSizes.md} />
      </button>
    )}
    {item.kind === 'local' && (
      <button
        type="button"
        onClick={onClearLocal}
        className="text-xs font-medium text-status-error hover:text-status-error"
        title="Drop the local file"
      >
        Clear
      </button>
    )}
  </li>
);
