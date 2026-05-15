import { Check, Eye, FileCode, FolderOpen, HardDrive } from 'lucide-react';
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
  onSelect: (item: ConfigItem) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}

/**
 * sourceTagFor stamps a one-word colour-coded badge onto a config row
 * so the user can tell at a glance whether they're looking at a
 * built-in template, a saved YAML, or the one-shot local upload.
 */
function sourceTagFor(item: ConfigItem) {
  if (item.kind === 'builtin')
    return (
      <Tag colorScheme="purple" className="text-[10px]">
        Built-in
      </Tag>
    );
  if (item.kind === 'saved')
    return (
      <Tag colorScheme="green" className="text-[10px]">
        Saved
      </Tag>
    );
  return (
    <Tag colorScheme="blue" className="text-[10px]">
      Local
    </Tag>
  );
}

/**
 * ConfigsList chooses between the card-grid view and the compact-list
 * view based on viewMode, and surfaces the loading + empty states.
 * The card / row presenters live below.
 */
export const ConfigsList: FC<{
  items: ConfigItem[];
  loading: boolean;
  viewMode: ViewMode;
  isSelected: (item: ConfigItem) => boolean;
  onSelect: (item: ConfigItem) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}> = ({ items, loading, viewMode, isSelected, onSelect, onView, onClearLocal }) => {
  if (loading) {
    return <SmallText className="text-gray-500">Loading configs…</SmallText>;
  }
  if (items.length === 0) {
    return (
      <SmallText className="text-gray-500">
        Nothing matches. Try a different search, switch the source filter, or upload a local YAML
        with the button above.
      </SmallText>
    );
  }

  if (viewMode === 'grid') {
    return (
      <div className="grid max-h-[420px] grid-cols-1 gap-3 overflow-y-auto pr-1 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <ConfigCard
            key={item.key}
            item={item}
            selected={isSelected(item)}
            onSelect={onSelect}
            onView={onView}
            onClearLocal={onClearLocal}
          />
        ))}
      </div>
    );
  }

  return (
    <ul className="max-h-72 divide-y divide-white/5 overflow-y-auto rounded-lg border border-white/10 bg-gray-950/40">
      {items.map((item) => (
        <ConfigRow
          key={item.key}
          item={item}
          selected={isSelected(item)}
          onSelect={onSelect}
          onView={onView}
          onClearLocal={onClearLocal}
        />
      ))}
    </ul>
  );
};

const ConfigCard: FC<SharedItemProps> = ({ item, selected, onSelect, onView, onClearLocal }) => {
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
        ? 'bg-emerald-500/15 text-emerald-200 border-emerald-400/30'
        : 'bg-blue-500/15 text-blue-200 border-blue-400/30';

  return (
    <div
      className={`flex flex-col gap-3 rounded-lg border p-3 transition-colors ${
        selected
          ? 'border-violet-400/50 bg-violet-500/10'
          : 'border-white/10 bg-gray-950/40 hover:border-violet-500/30'
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className={`rounded-md border p-2 ${tint}`}>
          <Icon className={iconSizes.lg} />
        </div>
        {sourceTagFor(item)}
      </div>
      <div>
        <div className="font-semibold text-white">{item.name}</div>
        <SmallText className="mt-0.5 line-clamp-2 text-gray-400">
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
          <div className="flex flex-1 items-center justify-center gap-1.5 rounded bg-violet-500/30 px-2 py-1.5 text-xs font-medium text-violet-50 ring-1 ring-violet-400/60">
            <Check className={iconSizes.sm} />
            <span>Will use this</span>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => onSelect(item)}
            className="flex-1 rounded bg-violet-500/20 px-2 py-1.5 text-xs font-medium text-violet-100 ring-1 ring-violet-400/40 hover:bg-violet-500/30"
            title="Pick this config — you'll still need to click Start Simulation below to run it"
          >
            Pick
          </button>
        )}
        {item.kind === 'builtin' && (
          <button
            type="button"
            onClick={() => onView(item)}
            className="rounded border border-white/10 bg-gray-900/60 px-2 py-1.5 text-xs font-medium text-gray-200 hover:bg-white/10"
            title="Preview YAML"
          >
            <Eye className={iconSizes.sm} />
          </button>
        )}
        {item.kind === 'local' && (
          <button
            type="button"
            onClick={onClearLocal}
            className="rounded border border-red-400/30 bg-red-500/10 px-2 py-1.5 text-xs font-medium text-red-200 hover:bg-red-500/20"
            title="Drop the local file"
          >
            Clear
          </button>
        )}
      </div>
    </div>
  );
};

const ConfigRow: FC<SharedItemProps> = ({ item, selected, onSelect, onView, onClearLocal }) => (
  <li
    className={`flex items-center gap-3 px-3 py-2 transition-colors ${
      selected ? 'bg-violet-500/10' : 'hover:bg-white/5'
    }`}
  >
    <button
      type="button"
      onClick={() => onSelect(item)}
      className="flex-1 text-left"
      title={`Use ${item.name}`}
    >
      <div className="flex items-center gap-2">
        <span className="font-medium text-white">{item.name}</span>
        {sourceTagFor(item)}
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {item.deviceCount} {item.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
        )}
        {item.kind === 'builtin' && (
          <Tag colorScheme="gray" className="text-[10px] capitalize">
            {item.template.type}
          </Tag>
        )}
      </div>
      {item.description && (
        <SmallText
          className={`mt-0.5 line-clamp-1 text-gray-400 ${
            item.kind === 'saved' ? 'font-mono text-[11px] text-gray-500' : ''
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
        className="rounded p-1.5 text-gray-400 hover:bg-white/10 hover:text-white"
        title="Preview the template YAML"
      >
        <Eye className={iconSizes.md} />
      </button>
    )}
    {item.kind === 'local' && (
      <button
        type="button"
        onClick={onClearLocal}
        className="text-xs font-medium text-red-300 hover:text-red-200"
        title="Drop the local file"
      >
        Clear
      </button>
    )}
  </li>
);
