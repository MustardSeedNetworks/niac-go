import { Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { Button } from '../../ui/Button';

interface ConfirmDeleteModalProps {
  hostname: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDeleteModal: FC<ConfirmDeleteModalProps> = ({
  hostname,
  onConfirm,
  onCancel,
}) => (
  <div className="fixed inset-0 z-50 flex-center">
    <button
      type="button"
      className="absolute inset-0 bg-scrim/70 backdrop-blur-sm"
      onClick={onCancel}
      aria-label="Close delete confirmation"
    />
    <div
      className="mx-4 w-full max-w-md rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl"
      role="dialog"
      aria-modal="true"
    >
      <div className="pad-lg stack-lg">
        <div className="flex items-center gap-default text-status-error">
          <Trash2 className="h-6 w-6" />
          <h2 className="heading-3">Delete Device</h2>
        </div>
        <p className="text-text-secondary">
          Are you sure you want to delete <strong>{hostname}</strong>? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-default pt-2">
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
