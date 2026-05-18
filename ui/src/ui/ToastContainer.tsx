import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';
import { type FC, useEffect } from 'react';
import { iconSizes } from '../constants/sizes';
import { type Notification, useUIStore } from '../stores/ui-store';

const DEFAULT_DURATION_MS = 5_000;

const TOAST_ICONS: Record<Notification['type'], FC<{ className?: string }>> = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

const TOAST_STYLES: Record<Notification['type'], string> = {
  success: 'border-status-success/30 bg-status-success/90 text-status-success',
  error: 'border-status-error/30 bg-status-error/90 text-status-error',
  warning: 'border-status-warning/30 bg-status-warning/90 text-status-warning',
  info: 'border-status-info/30 bg-status-info/90 text-status-info',
};

const Toast: FC<{ notification: Notification }> = ({ notification }) => {
  const removeNotification = useUIStore((s) => s.removeNotification);
  const Icon = TOAST_ICONS[notification.type];
  const duration = notification.duration ?? DEFAULT_DURATION_MS;

  useEffect(() => {
    if (duration <= 0) return;
    const timer = setTimeout(() => removeNotification(notification.id), duration);
    return () => clearTimeout(timer);
  }, [notification.id, duration, removeNotification]);

  return (
    <div
      className={`flex items-start gap-3 rounded-lg border px-4 py-3 shadow-lg backdrop-blur-sm ${TOAST_STYLES[notification.type]}`}
      role="alert"
    >
      <Icon className={`${iconSizes.lg} flex-shrink-0 mt-0.5`} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{notification.title}</p>
        {notification.message && (
          <p className="text-sm opacity-80 mt-0.5">{notification.message}</p>
        )}
      </div>
      <button
        type="button"
        onClick={() => removeNotification(notification.id)}
        className="flex-shrink-0 p-1 rounded hover:bg-surface-hover transition-colors"
        aria-label="Dismiss notification"
      >
        <X className={iconSizes.md} />
      </button>
    </div>
  );
};

export const ToastContainer: FC = () => {
  const notifications = useUIStore((s) => s.notifications);

  if (notifications.length === 0) return null;

  return (
    <div
      aria-live="polite"
      aria-atomic="false"
      className="fixed bottom-4 right-4 z-[200] flex flex-col gap-2 max-w-sm w-full"
    >
      {notifications.map((notification) => (
        <Toast key={notification.id} notification={notification} />
      ))}
    </div>
  );
};
