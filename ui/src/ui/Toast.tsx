import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';
import {
  type ComponentType,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../constants/sizes';
import { ToastContext, type ToastInput, type ToastType } from './toastContext';

const DEFAULT_DURATION_MS = 5_000;

const TOAST_ICONS: Record<ToastType, ComponentType<{ className?: string }>> = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

const TOAST_STYLES: Record<ToastType, string> = {
  success: 'border-status-success/30 bg-status-success/90 text-status-success',
  error: 'border-status-error/30 bg-status-error/90 text-status-error',
  warning: 'border-status-warning/30 bg-status-warning/90 text-status-warning',
  info: 'border-status-info/30 bg-status-info/90 text-status-info',
};

interface ToastItemData extends ToastInput {
  id: string;
  type: ToastType;
}

/**
 * ToastProvider owns the toast queue and renders it. Wrap the app once;
 * consumers go through useToast().
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItemData[]>([]);
  const nextId = useRef(0);

  const addToast = useCallback((input: ToastInput) => {
    nextId.current += 1;
    const id = `toast-${nextId.current}`;
    setToasts((prev) => [...prev, { ...input, id, type: input.type ?? 'info' }]);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const value = useMemo(() => ({ addToast, removeToast }), [addToast, removeToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastViewport toasts={toasts} removeToast={removeToast} />
    </ToastContext.Provider>
  );
}

function ToastViewport({
  toasts,
  removeToast,
}: {
  toasts: ToastItemData[];
  removeToast: (id: string) => void;
}) {
  const { t } = useTranslation('common');
  return (
    <section
      aria-live="polite"
      aria-atomic="false"
      aria-label={t('accessibility.notifications')}
      data-testid="toast-viewport"
      className="pointer-events-none fixed bottom-4 right-4 z-[200] flex w-full max-w-sm flex-col gap-compact"
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} removeToast={removeToast} />
      ))}
    </section>
  );
}

function ToastItem({
  toast,
  removeToast,
}: {
  toast: ToastItemData;
  removeToast: (id: string) => void;
}) {
  const { t } = useTranslation('common');
  const onClose = useCallback(() => removeToast(toast.id), [removeToast, toast.id]);
  const Icon = TOAST_ICONS[toast.type];
  const duration = toast.duration ?? DEFAULT_DURATION_MS;

  useEffect(() => {
    if (duration <= 0) return;
    const timer = setTimeout(onClose, duration);
    return () => clearTimeout(timer);
  }, [duration, onClose]);

  return (
    <div
      className={`pointer-events-auto flex items-start gap-default rounded-lg border px-4 py-row-lg shadow-lg backdrop-blur-sm ${TOAST_STYLES[toast.type]}`}
      role="alert"
    >
      <Icon className={`${iconSizes.lg} flex-shrink-0 mt-0.5`} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{toast.title}</p>
        {toast.message && <p className="text-sm opacity-80 mt-0.5">{toast.message}</p>}
        {toast.details && toast.details.length > 0 && (
          <ul className="mt-tight list-disc pl-4 text-xs opacity-80">
            {toast.details.map((detail) => (
              <li key={detail}>{detail}</li>
            ))}
          </ul>
        )}
      </div>
      <button
        type="button"
        onClick={onClose}
        className="flex-shrink-0 p-1 rounded hover:bg-surface-hover transition-colors"
        aria-label={t('accessibility.dismissNotification')}
      >
        <X className={iconSizes.md} />
      </button>
    </div>
  );
}
