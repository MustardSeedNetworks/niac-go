import { AlertTriangle } from 'lucide-react';
import type { FC, ReactNode } from 'react';
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
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  confirmTone = 'red',
  icon,
}) => (
  <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        {icon || (
          <AlertTriangle
            className={`h-6 w-6 ${
              confirmTone === 'red'
                ? 'text-red-400'
                : confirmTone === 'blue'
                  ? 'text-blue-400'
                  : confirmTone === 'green'
                    ? 'text-green-400'
                    : 'text-violet-400'
            }`}
          />
        )}
        <h2 className="text-lg font-semibold text-white">{title}</h2>
      </div>
      <div className="text-gray-300">{message}</div>
      <div className="flex justify-end gap-3 pt-2">
        <Button variant="outline" onClick={onCancel}>
          {cancelLabel}
        </Button>
        <Button tone={confirmTone} onClick={onConfirm}>
          {confirmLabel}
        </Button>
      </div>
    </div>
  </Modal>
);
