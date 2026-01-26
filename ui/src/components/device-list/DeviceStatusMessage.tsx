import { AlertCircle, X } from 'lucide-react';
import type { FC } from 'react';

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
      className={`flex items-center gap-2 rounded-lg p-3 ${
        message.type === 'success'
          ? 'border border-green-500/30 bg-green-500/10 text-green-300'
          : 'border border-red-500/30 bg-red-500/10 text-red-300'
      }`}
      role="alert"
    >
      {message.type === 'error' && <AlertCircle className="h-4 w-4" />}
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
