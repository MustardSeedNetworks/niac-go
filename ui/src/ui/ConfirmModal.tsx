import { AlertTriangle } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
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
}

export const ConfirmModal: FC<ConfirmModalProps> = ({
  isOpen,
  onConfirm,
  onCancel,
  title,
  message,
  confirmLabel,
  cancelLabel,
  confirmTone = 'red',
  icon,
}) => {
  const { t } = useTranslation();
  // Callers pass the title/message already (often containing dynamic
  // values), so they own translation of those. The confirm/cancel
  // buttons default to localised "Confirm" / "Cancel" when the
  // caller omits explicit labels.
  const resolvedConfirmLabel = confirmLabel ?? t('common:buttons.confirm', 'Confirm');
  const resolvedCancelLabel = cancelLabel ?? t('common:buttons.cancel', 'Cancel');
  return (
    <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
      <div className="space-y-4">
        <div className="flex items-center gap-3">
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
          <h2 className="text-lg font-semibold text-text-primary">{title}</h2>
        </div>
        <div className="text-text-secondary">{message}</div>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="outline" onClick={onCancel}>
            {resolvedCancelLabel}
          </Button>
          <Button tone={confirmTone} onClick={onConfirm}>
            {resolvedConfirmLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
};
