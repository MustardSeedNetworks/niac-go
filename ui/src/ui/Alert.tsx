import { type FC, type ReactNode } from 'react';
import { AlertCircle, CheckCircle, Info, AlertTriangle, X } from 'lucide-react';

export type AlertStatus = 'success' | 'error' | 'warning' | 'info';

export interface AlertProps {
  status: AlertStatus;
  children: ReactNode;
  onDismiss?: () => void;
  className?: string;
}

const statusConfig: Record<AlertStatus, {
  icon: typeof AlertCircle;
  containerClass: string;
  iconClass: string;
}> = {
  success: {
    icon: CheckCircle,
    containerClass: 'border-green-500/30 bg-green-500/10 text-green-300',
    iconClass: 'text-green-400',
  },
  error: {
    icon: AlertCircle,
    containerClass: 'border-red-500/30 bg-red-500/10 text-red-300',
    iconClass: 'text-red-400',
  },
  warning: {
    icon: AlertTriangle,
    containerClass: 'border-yellow-500/30 bg-yellow-500/10 text-yellow-300',
    iconClass: 'text-yellow-400',
  },
  info: {
    icon: Info,
    containerClass: 'border-blue-500/30 bg-blue-500/10 text-blue-300',
    iconClass: 'text-blue-400',
  },
};

export const Alert: FC<AlertProps> = ({
  status,
  children,
  onDismiss,
  className = '',
}) => {
  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div
      className={`flex items-center gap-2 rounded-lg border p-3 ${config.containerClass} ${className}`}
      role="alert"
    >
      <Icon className={`h-4 w-4 flex-shrink-0 ${config.iconClass}`} />
      <span className="flex-1">{children}</span>
      {onDismiss && (
        <button
          onClick={onDismiss}
          className="ml-auto text-current hover:opacity-70 transition-opacity"
          aria-label="Dismiss alert"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
};
