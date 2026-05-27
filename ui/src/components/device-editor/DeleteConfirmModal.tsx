import { Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';

export interface DeleteConfirmModalProps {
  deviceHostname: string;
  deleting: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const DeleteConfirmModal: FC<DeleteConfirmModalProps> = ({
  deviceHostname,
  deleting,
  onConfirm,
  onCancel,
}) => {
  return (
    <div className="fixed inset-0 z-50 flex-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
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
            <Trash2 className={iconSizes.xl} />
            <h2 className="heading-3">Delete Device</h2>
          </div>
          <p className="text-text-secondary">
            Are you sure you want to delete <strong>{deviceHostname}</strong>? This action cannot be
            undone.
          </p>
          <div className="flex justify-end gap-default pt-2">
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              className="bg-status-error hover:bg-status-error text-text-primary"
              onClick={onConfirm}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
