import { Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { Button } from '../../ui/Button';

interface DeviceBulkActionsProps {
  selectedCount: number;
  onDeleteSelected: () => void;
  onClearSelection: () => void;
}

export const DeviceBulkActions: FC<DeviceBulkActionsProps> = ({
  selectedCount,
  onDeleteSelected,
  onClearSelection,
}) => {
  if (selectedCount === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-4 p-3 rounded-lg bg-violet-500/10 border border-violet-500/30">
      <span className="text-sm text-violet-200">
        {selectedCount} device
        {selectedCount !== 1 ? 's' : ''} selected
      </span>
      <Button
        variant="ghost"
        size="sm"
        leftIcon={<Trash2 className="h-4 w-4" />}
        onClick={onDeleteSelected}
        className="text-red-400 hover:text-red-300 hover:bg-red-500/20"
      >
        Delete Selected
      </Button>
      <Button variant="ghost" size="sm" onClick={onClearSelection}>
        Clear Selection
      </Button>
    </div>
  );
};
