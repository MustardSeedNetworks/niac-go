import { AlertTriangle } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { iconSizes } from '../constants/sizes';
import { Button } from './Button';
import { Modal } from './Modal';

export interface ConfirmModalProps {
  isOpen: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  confirmTone?: 'red' | 'violet' | 'blue' | 'green';
  icon?: ReactNode;
  /** True while the confirmed action is in flight — disables the confirm
   * button and swaps in `confirmingLabel` so a slow delete/stop can't be
   * double-submitted. */
  confirming?: boolean;
  /** Label shown on the confirm button while `confirming` is true.
   * Defaults to `confirmLabel`. */
  confirmingLabel?: string;
}

export const ConfirmModal: FC<ConfirmModalProps> = ({
  isOpen,
  onConfirm,
  onCancel,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  confirmTone = 'red',
  icon,
  confirming = false,
  confirmingLabel,
}) => (
  <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
    <div className="stack-lg">
      <div className="flex items-center gap-default">
        {icon || (
          <AlertTriangle
            className={`${iconSizes.xl} ${
              confirmTone === 'red'
                ? 'text-status-error'
                : confirmTone === 'blue'
                  ? 'text-status-info'
                  : confirmTone === 'green'
                    ? 'text-status-success'
                    : 'text-brand-accent'
            }`}
          />
        )}
        <h2 className="heading-3 text-text-primary">{title}</h2>
      </div>
      <div className="text-text-secondary">{message}</div>
      <div className="flex justify-end gap-default pt-2">
        <Button variant="outline" onClick={onCancel}>
          {cancelLabel}
        </Button>
        <Button tone={confirmTone} onClick={onConfirm} disabled={confirming}>
          {confirming ? (confirmingLabel ?? confirmLabel) : confirmLabel}
        </Button>
      </div>
    </div>
  </Modal>
);
