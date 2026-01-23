import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';
import { type FC, useEffect } from 'react';
import { type Notification, useUIStore } from '../stores/ui-store';

const DEFAULT_DURATION_MS = 5_000;

const TOAST_ICONS: Record<Notification['type'], FC<{ className?: string }>> = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

const TOAST_STYLES: Record<Notification['type'], string> = {
  success: 'border-green-500/30 bg-green-900/90 text-green-100',
  error: 'border-red-500/30 bg-red-900/90 text-red-100',
  warning: 'border-amber-500/30 bg-amber-900/90 text-amber-100',
  info: 'border-blue-500/30 bg-blue-900/90 text-blue-100',
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
      <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{notification.title}</p>
        {notification.message && (
          <p className="text-sm opacity-80 mt-0.5">{notification.message}</p>
        )}
      </div>
      <button
        type="button"
        onClick={() => removeNotification(notification.id)}
        className="flex-shrink-0 p-1 rounded hover:bg-white/10 transition-colors"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
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
