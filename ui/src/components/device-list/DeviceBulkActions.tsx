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
    <div className="flex items-center gap-4 p-3 rounded-lg bg-brand-primary/10 border border-brand-primary/30">
      <span className="text-sm text-brand-accent">
        {selectedCount} device
        {selectedCount !== 1 ? 's' : ''} selected
      </span>
      <Button
        variant="ghost"
        size="sm"
        leftIcon={<Trash2 className="h-4 w-4" />}
        onClick={onDeleteSelected}
        className="text-status-error hover:text-status-error hover:bg-status-error/20"
      >
        Delete Selected
      </Button>
      <Button variant="ghost" size="sm" onClick={onClearSelection}>
        Clear Selection
      </Button>
    </div>
  );
};
