import { AlertCircle, Trash2, X } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { Button } from '../../ui/Button';

export interface BulkActionsProps {
  selectedCount: number;
  onDeleteSelected: () => void;
  onClearSelection: () => void;
}

export const BulkActions: FC<BulkActionsProps> = ({
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

export interface DeleteProgressProps {
  current: number;
  total: number;
}

export const DeleteProgress: FC<DeleteProgressProps> = ({ current, total }) => {
  return (
    <div className="flex items-center gap-3 rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
      <div className="h-4 w-4 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
      <span className="text-sm text-blue-700 dark:text-blue-300">
        Deleting devices: {current} / {total}
      </span>
    </div>
  );
};

export interface StatusMessageProps {
  type: 'success' | 'error';
  text: ReactNode;
  onDismiss: () => void;
}

export const StatusMessage: FC<StatusMessageProps> = ({ type, text, onDismiss }) => {
  return (
    <div
      className={`flex items-center gap-2 rounded-lg p-3 ${
        type === 'success'
          ? 'border border-green-500/30 bg-green-500/10 text-green-300'
          : 'border border-red-500/30 bg-red-500/10 text-red-300'
      }`}
      role="alert"
    >
      {type === 'error' && <AlertCircle className="h-4 w-4" />}
      <span>{text}</span>
      <button
        type="button"
        onClick={onDismiss}
        className="ml-auto text-current hover:opacity-70"
        aria-label="Dismiss message"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
};
