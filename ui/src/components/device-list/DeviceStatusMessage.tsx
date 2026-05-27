import { AlertCircle, X } from 'lucide-react';
import type { FC } from 'react';
import { iconSizes } from '../../constants/sizes';

export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}

interface DeviceStatusMessageProps {
  message: StatusMessage | null;
  onDismiss: () => void;
}

export const DeviceStatusMessage: FC<DeviceStatusMessageProps> = ({ message, onDismiss }) => {
  if (!message) {
    return null;
  }

  return (
    <div
      className={`flex items-center gap-compact rounded-lg pad-sm ${
        message.type === 'success'
          ? 'border border-status-success/30 bg-status-success/10 text-status-success'
          : 'border border-status-error/30 bg-status-error/10 text-status-error'
      }`}
      role="alert"
    >
      {message.type === 'error' && <AlertCircle className={iconSizes.md} />}
      <span>{message.text}</span>
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
