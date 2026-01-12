import { type FC } from 'react';
import { Trash2 } from 'lucide-react';
import { Button } from '../../ui';

interface ConfirmDeleteModalProps {
  hostname: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDeleteModal: FC<ConfirmDeleteModalProps> = ({
  hostname,
  onConfirm,
  onCancel,
}) => {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onCancel}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 space-y-4">
          <div className="flex items-center gap-3 text-red-400">
            <Trash2 className="h-6 w-6" />
            <h2 className="text-lg font-semibold">Delete Device</h2>
          </div>
          <p className="text-gray-300">
            Are you sure you want to delete <strong>{hostname}</strong>? This action cannot be undone.
          </p>
          <div className="flex justify-end gap-3 pt-2">
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button tone="red" onClick={onConfirm}>
              Delete
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
